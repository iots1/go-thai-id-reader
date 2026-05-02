package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	defaultTokenFileContent = "access-token=1234234 refresh-token=testeste"

	tokenFilePath   string
	hisQueueName    string
	hisJobName      string
	workerQueueName string
	workerJobName   string
	redisAddr       string
	redisUser       string
	redisPassword   string
	redisDB         int
	authBaseURL     string
)

func initConfig() {
	getEnv := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}

	tokenFilePath   = getEnv("TOKEN_FILE_PATH", defaultTokenFilePath())
	hisQueueName    = getEnv("HIS_QUEUE_NAME", "srm-his")
	hisJobName      = getEnv("HIS_JOB_NAME", "add-srm-token")
	workerQueueName = getEnv("WORKER_QUEUE_NAME", "srm-go")
	workerJobName   = getEnv("WORKER_JOB_NAME", "request-token")
	redisAddr       = getEnv("REDIS_ADDR", "192.168.30.33:6379")
	redisUser       = getEnv("REDIS_USER", "default")
	redisPassword   = getEnv("REDIS_PASSWORD", "")

	if resolved, err := ensureTokenFile(tokenFilePath); err != nil {
		log.Printf("[main] cannot prepare token file (%s): %v", tokenFilePath, err)
	} else {
		tokenFilePath = resolved
	}

	db, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		db = 0
	}
	redisDB = db

	authBaseURL = getEnv("AUTH_BASE_URL", "https://api-meditech-dev.dudee-indeed.com")
}

func defaultTokenFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "token.txt"
	}
	return filepath.Join(home, "Documents", "token.txt")
}

func ensureTokenFile(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return path, err
	}

	documentsPath := defaultTokenFilePath()
	if documentsPath != path {
		if _, err := os.Stat(documentsPath); err == nil {
			log.Printf("[main] token file %s missing, falling back to %s", path, documentsPath)
			return documentsPath, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(documentsPath), 0755); err != nil {
		return path, err
	}
	if err := os.WriteFile(documentsPath, []byte(defaultTokenFileContent), 0644); err != nil {
		return path, err
	}

	log.Printf("[main] token file created with default values at %s", documentsPath)
	return documentsPath, nil
}

type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// parseTokenFile รองรับ 2 รูปแบบ:
//  1. แต่ละ key=value อยู่คนละบรรทัด
//  2. ทุก key=value อยู่บรรทัดเดียวกัน คั่นด้วยช่องว่าง
func parseTokenFile(content string) (*TokenData, error) {
	// normalize newlines → space แล้ว split ด้วย whitespace
	content = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(content)

	data := &TokenData{}
	for _, field := range strings.Fields(content) {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])

		switch key {
		case "access-token", "access_token", "assess-token", "assess_token":
			data.AccessToken = val
		case "refresh-token", "refresh_token":
			data.RefreshToken = val
		}
	}

	if data.AccessToken == "" && data.RefreshToken == "" {
		return nil, fmt.Errorf("no token data found in file")
	}
	return data, nil
}

func addBullMQJob(ctx context.Context, client *redis.Client, tokenData *TokenData) error {
	jobData, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("marshal job data: %w", err)
	}

	opts, _ := json.Marshal(map[string]interface{}{
		"attempts": 3,
		"delay":    0,
	})

	// BullMQ ใช้ INCR เพื่อ generate job ID
	jobID, err := client.Incr(ctx, fmt.Sprintf("bull:%s:id", hisQueueName)).Result()
	if err != nil {
		return fmt.Errorf("increment job id: %w", err)
	}

	jobKey := fmt.Sprintf("bull:%s:%d", hisQueueName, jobID)

	// เก็บ job data เป็น hash
	if err := client.HSet(ctx, jobKey, map[string]interface{}{
		"name":      hisJobName,
		"data":      string(jobData),
		"opts":      string(opts),
		"timestamp": time.Now().UnixMilli(),
		"delay":     0,
		"priority":  0,
		"attempts":  0,
	}).Err(); err != nil {
		return fmt.Errorf("store job hash: %w", err)
	}

	// push job ID เข้า wait queue
	if err := client.LPush(ctx, fmt.Sprintf("bull:%s:wait", hisQueueName), fmt.Sprintf("%d", jobID)).Err(); err != nil {
		return fmt.Errorf("push to wait queue: %w", err)
	}

	log.Printf("[cronjob] job added → queue=%s jobID=%d", hisQueueName, jobID)
	return nil
}

func runTokenJob() {
	log.Println("[cronjob] กำลังอ่าน token file...")

	content, err := os.ReadFile(tokenFilePath)
	if err != nil {
		log.Printf("[cronjob] อ่านไฟล์ไม่ได้: %v", err)
		return
	}

	tokenData, err := parseTokenFile(string(content))
	if err != nil {
		log.Printf("[cronjob] parse token ไม่ได้: %v", err)
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[cronjob] เชื่อมต่อ Redis ไม่ได้: %v", err)
		return
	}

	if err := addBullMQJob(ctx, client, tokenData); err != nil {
		log.Printf("[cronjob] add job ไม่ได้: %v", err)
		return
	}
}

func getTokensHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    405,
			"message": "method not allowed",
		})
		return
	}

	content, err := os.ReadFile(tokenFilePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": fmt.Sprintf("cannot read token file: %v", err),
		})
		return
	}

	tokenData, err := parseTokenFile(string(content))
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    422,
			"message": fmt.Sprintf("cannot parse token file: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "ok",
		"data":    tokenData,
	})
}

func checkRedis() bool {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[main] redis ping failed (%s): %v", redisAddr, err)
		return false
	}
	log.Printf("[main] redis ok (%s)", redisAddr)
	return true
}

func startCronJob() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		runTokenJob() 
		for range ticker.C {
			runTokenJob()
		}
	}()
}
