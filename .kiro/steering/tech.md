# CloudCtl Technology Stack

## Language & Runtime
- **Go 1.24.0** - Primary language
- **CGO_ENABLED=1** - Required for macOS Keychain integration

## Key Dependencies
- **AWS SDK for Go v2** - AWS service integration
  - `github.com/aws/aws-sdk-go-v2` - Core SDK
  - `github.com/aws/aws-sdk-go-v2/service/sts` - STS operations for AssumeRole
  - `github.com/aws/aws-sdk-go-v2/config` - AWS configuration loading
- **Cobra** (`github.com/spf13/cobra`) - CLI framework and command structure
- **Charm Libraries** - Modern TUI components
  - `github.com/charmbracelet/bubbletea` - TUI framework
  - `github.com/charmbracelet/bubbles` - UI components (spinner, input)
  - `github.com/charmbracelet/lipgloss` - Styling and layout
- **Keychain** (`github.com/keybase/go-keychain`) - macOS Keychain integration
- **Terminal** (`golang.org/x/term`) - Terminal input handling (MFA masking)

## Build System
- **GoReleaser** - Release automation and cross-platform builds
- **Homebrew** - macOS package distribution via custom tap
- **Target Platforms**: macOS (amd64, arm64)

## Common Commands

### Development
```bash
# Build for current platform
go build

# Build with version info
go build -ldflags "-s -w -X 'github.com/chukul/cloudctl/internal.CurrentVersion=dev'"

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Cross-Platform Building
```bash
# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o cloudctl-macos-intel

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o cloudctl-macos-arm64
```

### Release
```bash
# Create release with GoReleaser
goreleaser release --clean

# Test release build locally
goreleaser build --snapshot --clean
```

### Installation
```bash
# From Homebrew
brew tap chukul/homebrew-tap
brew install cloudctl

# From source
git clone https://github.com/chukul/cloudctl.git
cd cloudctl
go build
sudo mv cloudctl /usr/local/bin/
```

## Architecture Notes
- **Encryption**: AES-256-GCM for credential storage
- **Storage**: JSON files in `~/.cloudctl/` directory
- **Platform-specific**: macOS Keychain integration via CGO
- **CLI Pattern**: Cobra command structure with persistent flags
- **TUI**: Bubble Tea for interactive components (spinners, selectors)

## Development Workflow
```bash
# Setup development environment
go mod tidy
go mod download

# Run specific tests
go test ./internal/crypto_test.go -v
go test ./internal/storage_test.go -v

# Check for race conditions
go test -race ./...

# Format code
go fmt ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

## Debugging
- Use `CLOUDCTL_DEBUG=1` environment variable for verbose logging
- Check `~/.cloudctl/` directory for storage issues
- Verify AWS CLI profiles with `aws configure list-profiles`
- Test encryption/decryption with known keys in tests