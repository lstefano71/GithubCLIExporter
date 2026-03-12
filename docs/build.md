# Build & Install (Go CLI)

Prerequisites

- Go 1.25+ installed (https://go.dev)

Build locally

Unix / macOS

```bash
cd go-cli
go build -o copilot-export .
./copilot-export --version
```

Windows (PowerShell)

```powershell
cd go-cli
go build -o copilot-export.exe .
.\copilot-export.exe --version
```

Install to your `GOBIN` (optional)

```bash
cd go-cli
go install
# binary will be in $GOBIN or GOPATH/bin
copilot-export --version
```

Cross-compilation

Use `GOOS`/`GOARCH` environment variables to build other platforms. Example (Linux amd64):

```bash
cd go-cli
GOOS=linux GOARCH=amd64 go build -o copilot-export-linux .
```

Packaging

Releases are produced from CI and attached to GitHub releases. See the project's Releases page for pre-built binaries.
