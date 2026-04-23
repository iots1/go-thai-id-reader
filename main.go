package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ebfe/scard"
	"github.com/joho/godotenv"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *IDData `json:"data,omitempty"`
}

type IDData struct {
	CID       string `json:"cid"`
	NameTH    string `json:"name_th"`
	NameEN    string `json:"name_en"`
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender"`
	Address   string `json:"address"`
	Photo     string `json:"photo"`
}

var (
	SELECT_APPLET = []byte{0x00, 0xA4, 0x04, 0x00, 0x08, 0xA0, 0x00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01}
	CMD_CID       = []byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D}
	CMD_NAME_TH   = []byte{0x80, 0xB0, 0x00, 0x11, 0x02, 0x00, 0x64}
	CMD_NAME_EN   = []byte{0x80, 0xB0, 0x00, 0x75, 0x02, 0x00, 0x64}
	CMD_BIRTH     = []byte{0x80, 0xB0, 0x00, 0xD9, 0x02, 0x00, 0x08}
	CMD_GENDER    = []byte{0x80, 0xB0, 0x00, 0xE1, 0x02, 0x00, 0x01}
	CMD_ADDRESS   = []byte{0x80, 0xB0, 0x15, 0x79, 0x02, 0x00, 0x64}
)

// Status codes
const (
	CodeSuccess            = 200000
	CodeContextFail        = 400001
	CodeNoReaderFound      = 400002
	CodeCardUnresponsive   = 400003
	CodeReadFail           = 400004
)

func DecodeThai(b []byte) string {
	r := transform.NewReader(strings.NewReader(string(b)), charmap.Windows874.NewDecoder())
	data, _ := io.ReadAll(r)
	return strings.TrimSpace(strings.ReplaceAll(string(data), "#", " "))
}

func FormatBirthDate(b []byte) string {
	dateStr := strings.TrimSpace(string(b))
	if dateStr == "" || dateStr == "        " {
		return ""
	}
	// Thai date format: YYYYMMDD
	if len(dateStr) >= 8 {
		year, _ := strconv.Atoi(dateStr[0:4])
		month := dateStr[4:6]
		day := dateStr[6:8]
		// Convert Thai Buddhist year to Gregorian
		gregorianYear := year - 543
		return fmt.Sprintf("%04d-%s-%s", gregorianYear, month, day)
	}
	return dateStr
}

func transmit(card *scard.Card, cmd []byte) ([]byte, error) {
	time.Sleep(100 * time.Millisecond)
	res, err := card.Transmit(cmd)
	if err != nil {
		return nil, err
	}
	if len(res) < 2 {
		return res, nil
	}
	sw1 := res[len(res)-2]
	sw2 := res[len(res)-1]

	// 0x61 XX — more data; fetch with GET RESPONSE
	if sw1 == 0x61 {
		getResponse := []byte{0x00, 0xC0, 0x00, 0x00, sw2}
		return card.Transmit(getResponse)
	}
	// 0x6C XX — wrong Le; retry with exact length the card tells us
	if sw1 == 0x6C {
		retry := make([]byte, len(cmd))
		copy(retry, cmd)
		retry[len(retry)-1] = sw2
		return card.Transmit(retry)
	}
	return res, nil
}

func readPhoto(card *scard.Card) []byte {
	var photoData []byte
	offset := 0x017B
	photoSize := 5000

	for len(photoData) < photoSize {
		hi := byte((offset >> 8) & 0xFF)
		lo := byte(offset & 0xFF)
		cmd := []byte{0x80, 0xB0, hi, lo, 0x02, 0x00, 0xFF}

		res, err := transmit(card, cmd)
		if err != nil || len(res) < 2 {
			break
		}

		data := res[:len(res)-2]
		if len(data) == 0 {
			break
		}

		photoData = append(photoData, data...)
		offset += len(data)

		if len(data) < 0xFF {
			break
		}
	}

	return photoData
}

func listReadersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ctx, err := scard.EstablishContext()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	defer ctx.Release()
	readers, err := ctx.ListReaders()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"readers": readers})
}

func trimSW(b []byte) []byte {
	if len(b) >= 2 {
		return b[:len(b)-2]
	}
	return b
}

func swOK(b []byte) bool {
	return len(b) >= 2 && b[len(b)-2] == 0x90 && b[len(b)-1] == 0x00
}

func connectCard(ctx *scard.Context, reader string) (*scard.Card, error) {
	// Thai ID cards use T=0; try exclusive first (needed by some readers like MT65)
	protocols := []scard.Protocol{scard.ProtocolT0, scard.ProtocolAny}
	modes := []scard.ShareMode{scard.ShareExclusive, scard.ShareShared}
	for _, proto := range protocols {
		for _, mode := range modes {
			card, err := ctx.Connect(reader, mode, proto)
			if err == nil {
				return card, nil
			}
		}
	}
	return nil, fmt.Errorf("cannot connect to card on reader %q", reader)
}

func readIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx, err := scard.EstablishContext()
	if err != nil {
		resp := Response{Code: CodeContextFail, Message: "Failed to establish PC/SC context"}
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer ctx.Release()

	readers, _ := ctx.ListReaders()
	if len(readers) == 0 {
		resp := Response{Code: CodeNoReaderFound, Message: "No card reader found"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Try each detected reader until one works
	var card *scard.Card
	for _, reader := range readers {
		card, err = connectCard(ctx, reader)
		if err == nil {
			log.Printf("[reader] connected via %q", reader)
			break
		}
		log.Printf("[reader] skip %q: %v", reader, err)
	}
	if card == nil {
		resp := Response{Code: CodeCardUnresponsive, Message: "Card unresponsive or not detected"}
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer card.Disconnect(scard.LeaveCard)

	selResp, err := transmit(card, SELECT_APPLET)
	if err != nil || !swOK(selResp) {
		log.Printf("[reader] SELECT_APPLET failed: err=%v resp=%X", err, selResp)
		resp := Response{Code: CodeReadFail, Message: "Failed to select Thai ID applet — card may not be inserted or reader incompatible"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	cid, _ := transmit(card, CMD_CID)
	nTH, _ := transmit(card, CMD_NAME_TH)
	nEN, _ := transmit(card, CMD_NAME_EN)
	dob, _ := transmit(card, CMD_BIRTH)
	gen, _ := transmit(card, CMD_GENDER)
	addr, _ := transmit(card, CMD_ADDRESS)
	photo := readPhoto(card)

	if len(cid) < 2 {
		resp := Response{Code: CodeReadFail, Message: "Failed to read ID data from card"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	photoBase64 := ""
	if len(photo) > 0 {
		photoBase64 = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(photo)
	}

	gender := ""
	if len(gen) >= 1 {
		gender = map[byte]string{49: "ชาย", 50: "หญิง"}[gen[0]]
	}

	var nameTH, nameEN, birthDate, address string
	if len(nTH) >= 2 {
		nameTH = DecodeThai(trimSW(nTH))
	}
	if len(nEN) >= 2 {
		nameEN = strings.TrimSpace(strings.ReplaceAll(string(trimSW(nEN)), "#", " "))
	}
	if len(dob) >= 2 {
		birthDate = FormatBirthDate(trimSW(dob))
	}
	if len(addr) >= 2 {
		address = DecodeThai(trimSW(addr))
	}

	result := IDData{
		CID:       string(trimSW(cid)),
		NameTH:    nameTH,
		NameEN:    nameEN,
		BirthDate: birthDate,
		Gender:    gender,
		Address:   address,
		Photo:     photoBase64,
	}

	resp := Response{
		Code:    CodeSuccess,
		Message: "ID card read successfully",
		Data:    &result,
	}
	json.NewEncoder(w).Encode(resp)
}

func runApp() error {
	envPath := `C:\Program Files (x86)\dudee\\.env`
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("[main] .env not found at %s, using existing environment", envPath)
	}
	initConfig()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if checkRedis() {
		startCronJob()
		startWorker()
	} else {
		log.Println("[main] WARNING: Redis ไม่พร้อมใช้งาน — ข้าม cronjob และ worker (API ยังทำงานได้ปกติ)")
	}
	http.HandleFunc("/api/v1/readers", readIDHandler)
	http.HandleFunc("/api/v1/devices", listReadersHandler)
	http.HandleFunc("/api/v1/tokens", getTokensHandler)
	fmt.Printf("Go Thai ID API Running at http://localhost:%s\n", port)
	return http.ListenAndServe(":"+port, nil)
}

func main() {
	debugMode := flag.Bool("debug", false, "run without tray mode and keep the console visible")
	flag.Parse()

	if *debugMode {
		if err := runApp(); err != nil {
			log.Fatal(err)
		}
		return
	}

	runWithTray(runApp)
}