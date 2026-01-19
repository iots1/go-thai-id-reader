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

## Building from Source

> **Note:** This project requires CGO due to the `scard` library dependency.
> Cross-compilation is not supported. Each platform must be built on its native OS.

### Using Makefile (Recommended)

```bash
# Build for current platform
make build

# Build and create release package
make package

# Clean build artifacts
make clean

# Show all available commands
make help
```

### Manual Build

```bash
# macOS / Linux
CGO_ENABLED=1 go build -o go-thai-id-api main.go

# Windows (from Windows)
set CGO_ENABLED=1
go build -o go-thai-id-api.exe main.go
```

### Creating a Release

```bash
# Tag and push to trigger GitHub Actions
git tag v1.0.0
git push origin v1.0.0

# Or manually trigger from GitHub Actions page
```

GitHub Actions will automatically build for all platforms and create a release.

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
├── main.go                          # Main application
├── go.mod                           # Go module definition
├── go.sum                           # Go dependency checksums
├── Makefile                         # Build automation
├── README.md                        # This file
├── .github/
│   └── workflows/
│       └── release.yml              # GitHub Actions for auto-release
└── scripts/
    ├── install.bat                  # Windows installer
    ├── uninstall.bat                # Windows uninstaller
    ├── install.sh                   # macOS installer
    └── uninstall.sh                 # macOS uninstaller
```

## Pre-built Binaries (Recommended)

Download pre-built binaries from the [Releases](../../releases) page.

| Platform | Architecture | Download |
|----------|-------------|----------|
| Windows | 64-bit | `go-thai-id-api-windows-amd64.zip` |
| macOS | Apple Silicon (M1/M2/M3) | `go-thai-id-api-darwin-arm64.tar.gz` |
| macOS | Intel | `go-thai-id-api-darwin-amd64.tar.gz` |

### Quick Installation

**Windows:**
```batch
# 1. Download and extract go-thai-id-api-windows-amd64.zip
# 2. Right-click install.bat > Run as Administrator
```

**macOS:**
```bash
# 1. Download the appropriate .tar.gz file
tar -xzf go-thai-id-api-darwin-arm64.tar.gz  # Apple Silicon
# or
tar -xzf go-thai-id-api-darwin-amd64.tar.gz  # Intel

# 2. Run installer
./install.sh
```

### What the installer does

**Windows (`install.bat`):**
- Installs to `C:\Program Files\GoThaiIDAPI\`
- Creates Windows Service (auto-start on boot)
- Opens firewall port 8080

**macOS (`install.sh`):**
- Installs to `/usr/local/bin/`
- Creates LaunchAgent (auto-start on login)
- Logs stored in `~/Library/Logs/GoThaiIDAPI/`

### Uninstallation

**Windows:** Run `uninstall.bat` as Administrator

**macOS:** Run `./uninstall.sh`

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
