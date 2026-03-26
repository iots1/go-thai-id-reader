package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPassword,
		DB:       redisDB,
	})
}

func completeJob(ctx context.Context, client *redis.Client, jobID string, result string) {
	now := time.Now().UnixMilli()
	jobKey := fmt.Sprintf("bull:%s:%s", workerQueueName, jobID)
	activeKey := fmt.Sprintf("bull:%s:active", workerQueueName)
	completedKey := fmt.Sprintf("bull:%s:completed", workerQueueName)

	client.HSet(ctx, jobKey, map[string]interface{}{
		"processedOn": now,
		"finishedOn":  now,
		"returnvalue": result,
	})
	client.LRem(ctx, activeKey, 0, jobID)
	client.ZAdd(ctx, completedKey, redis.Z{Score: float64(now), Member: jobID})
}

func failJob(ctx context.Context, client *redis.Client, jobID string, reason string) {
	now := time.Now().UnixMilli()
	jobKey := fmt.Sprintf("bull:%s:%s", workerQueueName, jobID)
	activeKey := fmt.Sprintf("bull:%s:active", workerQueueName)
	failedKey := fmt.Sprintf("bull:%s:failed", workerQueueName)

	client.HSet(ctx, jobKey, map[string]interface{}{
		"processedOn":  now,
		"finishedOn":   now,
		"failedReason": reason,
		"stacktrace":   "[]",
	})
	client.LRem(ctx, activeKey, 0, jobID)
	client.ZAdd(ctx, failedKey, redis.Z{Score: float64(now), Member: jobID})
}

func processJob(ctx context.Context, client *redis.Client, jobID string) {
	jobKey := fmt.Sprintf("bull:%s:%s", workerQueueName, jobID)

	fields, err := client.HGetAll(ctx, jobKey).Result()
	if err != nil || len(fields) == 0 {
		log.Printf("[worker] job %s not found in Redis", jobID)
		return
	}

	name := fields["name"]
	log.Printf("[worker] received job → id=%s name=%s", jobID, name)

	if name != workerJobName {
		log.Printf("[worker] job %s unknown name (%s ≠ %s), skipping", jobID, name, workerJobName)
		completeJob(ctx, client, jobID, "skipped")
		return
	}

	client.HSet(ctx, jobKey, "processedOn", time.Now().UnixMilli())

	runTokenJob()

	completeJob(ctx, client, jobID, "ok")
	log.Printf("[worker] job %s done", jobID)
}

func startWorker() {
	log.Printf("[worker] starting BullMQ worker → queue=%s jobName=%s", workerQueueName, workerJobName)

	waitKey := fmt.Sprintf("bull:%s:wait", workerQueueName)
	activeKey := fmt.Sprintf("bull:%s:active", workerQueueName)

	go func() {
		for {
			client := newRedisClient()
			ctx := context.Background()

			jobID, err := client.BLMove(ctx, waitKey, activeKey, "RIGHT", "LEFT", 0).Result()
			if err != nil {
				client.Close()
				log.Printf("[worker] Redis disconnected: %v — retrying in 5s", err)
				time.Sleep(5 * time.Second)
				continue
			}

			processJob(ctx, client, jobID)
			client.Close()
		}
	}()
}
