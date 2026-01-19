# Go Thai ID API

A Go application that reads Thai National ID cards using PC/SC smart card readers and exposes the data through a REST API.

## Features

- Read Thai ID card data (CID, name, birth date, gender, address)
- Extract and return ID card photo as base64-encoded JPEG
- Cross-platform support (macOS, Linux, Windows)
- RESTful API with standardized response codes
- CORS enabled for web integration

## Requirements

- Go 1.25.6 or higher
- PC/SC compatible smart card reader
- Thai National ID Card

### System Dependencies

**macOS:**
- Xcode Command Line Tools
- PC/SC framework (built-in)

**Linux:**
- `libpcsclite-dev` package
  ```bash
  # Ubuntu/Debian
  sudo apt-get install libpcsclite-dev

  # Fedora/RHEL
  sudo dnf install pcsc-lite-devel
  ```

**Windows:**
- Windows Smart Card Support (built-in)
- Visual Studio Build Tools or MinGW (for compilation)

## Installation

### Clone the repository

```bash
git clone https://github.com/iots1/go-thai-id-api.git
cd go-thai-id-api
```

### Install Go dependencies

```bash
go mod download
```

## Building

### macOS

```bash
# Build for current architecture
go build -o thaiid main.go

# Build for specific architecture
go build -o thaiid-arm64 main.go        # Apple Silicon
go build -o thaiid-amd64 main.go        # Intel Mac
```

### Linux

```bash
# Build for current architecture
go build -o thaiid main.go

# Build for specific architecture
GOOS=linux GOARCH=amd64 go build -o thaiid-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o thaiid-arm64 main.go
```

### Windows

```bash
# Build for Windows (from Windows machine)
go build -o thaiid.exe main.go

# Cross-compile from Linux/macOS
GOOS=windows GOARCH=amd64 go build -o thaiid.exe main.go
```

## Running

### Start the server

```bash
# Run directly
go run main.go

# Or run the compiled binary
./thaiid
```

The API will be available at `http://localhost:8080/api/read`

```
🚀 Go Thai ID API: http://localhost:8080/api/read
```

## API Usage

### Endpoint

```
GET http://localhost:8080/api/read
```

### Success Response (Code: 200000)

```json
{
  "code": 200000,
  "message": "ID card read successfully",
  "data": {
    "cid": "9868971150439",
    "name_th": "นาย พลาธร ชัยศักดิ์",
    "name_en": "Mr. Plathorn Chaiyasak",
    "birth_date": "1989-04-15",
    "gender": "ชาย",
    "address": "555/55 หมู่ที่ 3 ตำบลXXX อำเภอXXX จังหวัดXXX",
    "photo": "data:image/jpeg;base64,/9j/4AAQSkZJRgABA..."
  }
}
```

### Error Responses

**No Reader Found (Code: 400002)**
```json
{
  "code": 400002,
  "message": "No card reader found",
  "data": null
}
```

**Card Unresponsive (Code: 400003)**
```json
{
  "code": 400003,
  "message": "Card unresponsive or not detected",
  "data": null
}
```

**PC/SC Context Failed (Code: 400001)**
```json
{
  "code": 400001,
  "message": "Failed to establish PC/SC context",
  "data": null
}
```

**Read Failed (Code: 400004)**
```json
{
  "code": 400004,
  "message": "Failed to read ID data from card",
  "data": null
}
```

### Status Codes

| Code   | Meaning                      |
|--------|------------------------------|
| 200000 | Success                      |
| 400001 | PC/SC Context Failed         |
| 400002 | No Card Reader Found         |
| 400003 | Card Unresponsive            |
| 400004 | Failed to Read ID Data       |

## Example Usage

### cURL

```bash
curl http://localhost:8080/api/read
```

### JavaScript/Fetch

```javascript
fetch('http://localhost:8080/api/read')
  .then(res => res.json())
  .then(data => {
    if (data.code === 200000) {
      console.log('ID Card Data:', data.data);
      console.log('Photo:', data.data.photo);
    } else {
      console.error('Error:', data.message);
    }
  });
```

### Python

```python
import requests

response = requests.get('http://localhost:8080/api/read')
data = response.json()

if data['code'] == 200000:
    print(f"Name: {data['data']['name_th']}")
    print(f"CID: {data['data']['cid']}")
else:
    print(f"Error: {data['message']}")
```

## Project Structure

```
.
├── main.go              # Main application
├── go.mod              # Go module definition
├── go.sum              # Go dependency checksums
└── README.md           # This file
```

## Troubleshooting

### Card reader not detected

- Verify the reader is connected and powered on
- Check system device manager for the reader
- Restart the reader/computer
- Try another USB port

### "Card Unresponsive" error

- Ensure the ID card is inserted correctly
- Try removing and reinserting the card
- Clean the card contacts
- Try another card to verify reader functionality

### Permission denied on Linux

- Add your user to the `pcscd` group:
  ```bash
  sudo usermod -a -G pcscd $USER
  newgrp pcscd
  ```
- Or run with sudo

### Build errors on Windows

- Install Visual Studio Build Tools or MinGW
- Ensure Windows SDK is installed
- For cross-compilation, ensure CGO is properly configured

## Dependencies

- `github.com/ebfe/scard` - Smart Card Interface
- `golang.org/x/text` - Text encoding/decoding

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please open an issue on the GitHub repository.

## Author

Created for reading Thai National ID cards with Go
