# Go Thai ID API

A Go application that reads Thai National ID cards using PC/SC smart card readers and exposes the data through a REST API.

## Features

- Read Thai ID card data (CID, name, birth date, gender, address)
- Extract and return ID card photo as base64-encoded JPEG
- Cross-platform support (macOS, Linux, Windows)
- RESTful API with standardized response codes
- CORS enabled for web integration
- **[Optional]** BullMQ integration — forward NHSO patient tokens to a remote server via Redis

## Requirements

- Go 1.21 or higher
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

## Configuration

Copy `.env.example` to `.env` and adjust as needed:

```bash
cp .env.example .env
```

| Variable          | Default              | Description                          |
|-------------------|----------------------|--------------------------------------|
| `PORT`            | `8080`               | HTTP server port                     |
| `TOKEN_FILE_PATH` | `./token.txt`        | Path to NHSO token file (see below)  |
| `REDIS_ADDR`      | `127.0.0.1:6379`     | Redis address                        |
| `REDIS_USER`      | `default`            | Redis username                       |
| `REDIS_PASSWORD`  | _(empty)_            | Redis password                       |
| `REDIS_DB`        | `0`                  | Redis database index                 |
| `HIS_QUEUE_NAME`  | `srm-his`            | BullMQ queue name (HIS server)       |
| `HIS_JOB_NAME`    | `add-srm-token`      | BullMQ job name (HIS server)         |
| `WORKER_QUEUE_NAME` | `srm-go`           | BullMQ queue name (this worker)      |
| `WORKER_JOB_NAME` | `request-token`      | BullMQ job name (this worker)        |

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
CGO_ENABLED=1 go build -o go-thai-id-api .

# Windows (from Windows)
set CGO_ENABLED=1
go build -o go-thai-id-api.exe .
```

### Creating a Release

```bash
# Tag and push to trigger GitHub Actions
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will automatically build for all platforms and create a release.

## Running

### Start the server

```bash
# Run directly
go run .

# Run without tray mode and keep the console visible
go run . --debug

# Or run the compiled binary
./go-thai-id-api
```

The API will be available at `http://localhost:8080`

```
Go Thai ID API Running at http://localhost:8080
```

If Redis is not reachable at startup, the server will still run but cronjob and worker will be disabled:

```
[main] redis ping failed (127.0.0.1:6379): ...
[main] WARNING: redis unavailable — cronjob and worker disabled (API still running)
```

## Optional: BullMQ / NHSO Token Integration

This feature automates forwarding patient treatment-rights tokens from the [NHSO (สปสช)](https://www.nhso.go.th/) desktop software to a HIS (Hospital Information System) server.

### How it works

1. The NHSO desktop app writes a token file (`.txt`) after a patient inserts their ID card. The file contains `access_token` and `refresh_token` used to verify the patient's healthcare coverage.
2. This service reads that token file every minute (**cronjob**) and also listens for on-demand trigger jobs (**worker**).
3. The token is pushed as a BullMQ job into a Redis queue, where the HIS server picks it up to perform the coverage check.

### Token file format

The token file supports two formats:

**Multi-line:**
```
access_token=eyJhbGciOi...
refresh_token=eyJhbGciOi...
```

**Single-line:**
```
access_token=eyJhbGciOi... refresh_token=eyJhbGciOi...
```

### Requirements

- Redis server
- A BullMQ-compatible consumer running on the HIS server

If Redis is not available, the application starts normally without the cronjob and worker. No configuration is required to disable this feature.

## API Usage

### Endpoints

| Method | Path               | Description               |
|--------|--------------------|---------------------------|
| `GET`  | `/api/v1/readers`  | Read inserted ID card     |
| `GET`  | `/api/v1/tokens`   | Read current token file   |

### Read ID Card

```
GET http://localhost:8080/api/v1/readers
```

**Success Response (Code: 200000)**

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

**Error Responses**

| Code   | Meaning                      |
|--------|------------------------------|
| 200000 | Success                      |
| 400001 | PC/SC Context Failed         |
| 400002 | No Card Reader Found         |
| 400003 | Card Unresponsive            |
| 400004 | Failed to Read ID Data       |

### Read Token File

```
GET http://localhost:8080/api/v1/tokens
```

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi..."
  }
}
```

## Example Usage

### cURL

```bash
curl http://localhost:8080/api/v1/readers
```

### JavaScript/Fetch

```javascript
fetch('http://localhost:8080/api/v1/readers')
  .then(res => res.json())
  .then(data => {
    if (data.code === 200000) {
      console.log('ID Card Data:', data.data);
    } else {
      console.error('Error:', data.message);
    }
  });
```

### Python

```python
import requests

response = requests.get('http://localhost:8080/api/v1/readers')
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
├── main.go          # HTTP server, ID card reader
├── cronjob.go       # Config, token parser, BullMQ job producer, cronjob
├── worker.go        # BullMQ worker (listens for on-demand trigger jobs)
├── go.mod
├── go.sum
├── .env.example
├── Makefile
├── README.md
├── .github/
│   └── workflows/
│       └── release.yml
└── scripts/
    ├── install.bat
    ├── uninstall.bat
    ├── install.sh
    └── uninstall.sh
```

## Pre-built Binaries

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

### Permission denied on Linux

```bash
sudo usermod -a -G pcscd $USER
newgrp pcscd
```

### Build errors on Windows

- Install Visual Studio Build Tools or MinGW
- Ensure Windows SDK is installed

## Dependencies

- `github.com/ebfe/scard` - Smart Card Interface
- `golang.org/x/text` - Text encoding/decoding
- `github.com/joho/godotenv` - .env file loader
- `github.com/redis/go-redis/v9` - Redis client (optional feature)

## License

MIT License

Copyright (c) 2026 iots1

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

Created for reading Thai National ID cards with Go
