package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ebfe/scard"
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
	if len(res) >= 2 && res[len(res)-2] == 0x61 {
		getResponse := []byte{0x00, 0xC0, 0x00, 0x00, res[len(res)-1]}
		return card.Transmit(getResponse)
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

	card, err := ctx.Connect(readers[0], scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		resp := Response{Code: CodeCardUnresponsive, Message: "Card unresponsive or not detected"}
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer card.Disconnect(scard.LeaveCard)

	transmit(card, SELECT_APPLET)

	cid, _ := transmit(card, CMD_CID)
	nTH, _ := transmit(card, CMD_NAME_TH)
	nEN, _ := transmit(card, CMD_NAME_EN)
	dob, _ := transmit(card, CMD_BIRTH)
	gen, _ := transmit(card, CMD_GENDER)
	addr, _ := transmit(card, CMD_ADDRESS)
	photo := readPhoto(card)

	if cid == nil || len(cid) < 2 {
		resp := Response{Code: CodeReadFail, Message: "Failed to read ID data from card"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	photoBase64 := ""
	if len(photo) > 0 {
		photoBase64 = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(photo)
	}

	result := IDData{
		CID:       string(cid[:len(cid)-2]),
		NameTH:    DecodeThai(nTH[:len(nTH)-2]),
		NameEN:    strings.TrimSpace(strings.ReplaceAll(string(nEN[:len(nEN)-2]), "#", " ")),
		BirthDate: FormatBirthDate(dob[:len(dob)-2]),
		Gender:    map[byte]string{49: "ชาย", 50: "หญิง"}[gen[0]],
		Address:   DecodeThai(addr[:len(addr)-2]),
		Photo:     photoBase64,
	}

	resp := Response{
		Code:    CodeSuccess,
		Message: "ID card read successfully",
		Data:    &result,
	}
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/api/read", readIDHandler)
	fmt.Println("🚀 Go Thai ID API: http://localhost:8080/api/read")
	log.Fatal(http.ListenAndServe(":8080", nil))
}