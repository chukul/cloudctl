# CloudCtl Project Structure

## Root Directory
```
cloudctl/
├── main.go              # Entry point - calls cmd.Execute()
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── .goreleaser.yaml     # Release configuration
├── README.md            # Comprehensive documentation
└── cloudctl             # Built binary (gitignored)
```

## Command Layer (`cmd/`)
All CLI commands following Cobra patterns:

```
cmd/
├── root.go              # Root command, logo, and CLI setup
├── login.go             # AssumeRole with MFA support
├── mfa-login.go         # MFA session creation
├── status.go            # Session listing with visual status
├── switch.go            # Quick profile switching
├── refresh.go           # Session renewal
├── console.go           # AWS Console URL generation
├── logout.go            # Credential removal
├── list.go              # Profile listing
├── prompt.go            # Shell integration
├── init.go              # Shell setup generation
├── daemon.go            # Auto-refresh daemon (macOS)
├── version.go           # Version information
├── secret.go            # Keychain secret management
├── role.go              # IAM role aliasing
├── mfa.go               # MFA device aliasing
├── sync.go              # AWS credentials file sync
└── utils.go             # Shared utilities (MFA input masking)
```

## Internal Packages (`internal/`)
Core business logic and platform-specific code:

```
internal/
├── types.go             # AWSSession struct definition
├── storage.go           # Encrypted credential persistence
├── storage_test.go      # Storage layer tests
├── crypto.go            # AES-256-GCM encryption/decryption
├── crypto_test.go       # Encryption tests
├── aws.go               # AWS SDK helpers and STS operations
├── session.go           # Session management logic
├── version.go           # Version checking and updates
├── os_utils.go          # OS-specific utilities
├── keychain_darwin.go   # macOS Keychain integration
├── keychain_stub.go     # Non-macOS keychain stub
└── ui/                  # TUI components
    ├── input.go         # Interactive input handling
    ├── selector.go      # Profile/option selection
    └── spinner.go       # Loading spinners
```

## Configuration & Scripts
```
scripts/
└── console-open.sh      # Console URL opening script

.github/                 # GitHub Actions (if present)
.kiro/                   # Kiro steering files
└── steering/
    ├── product.md
    ├── tech.md
    └── structure.md
```

## Data Storage Structure
Runtime data stored in user home directory:
```
~/.cloudctl/
├── credentials.json     # Encrypted AWS sessions
├── mfa.json            # MFA device aliases
├── roles.json          # IAM role aliases
└── sessions/           # Session files (daemon usage)
```

## Architecture Patterns

### Command Structure
- Each command in `cmd/` follows Cobra patterns
- Commands use persistent flags for common options (--secret, --profile)
- Interactive prompts for missing required parameters
- Consistent error handling with helpful troubleshooting messages

### Internal Organization
- `types.go` - Core data structures (AWSSession)
- `storage.go` - All persistence operations with encryption
- `crypto.go` - Security layer (AES-256-GCM)
- `aws.go` - AWS SDK interactions and STS operations
- Platform-specific files use build tags (`_darwin.go`, `_stub.go`)

### Testing Strategy
- Unit tests for crypto and storage layers
- Test helpers for temporary directories
- Mocking for AWS SDK interactions
- Error case coverage (corrupt data, wrong keys)

### Security Architecture
- All sensitive data encrypted before storage
- Keychain integration for key management (macOS)
- No plaintext credentials in memory longer than necessary
- Secure file permissions (0600/0700) for config directories

## Common Patterns

### Session Management Flow
1. User provides source profile/credentials
2. Assume role via STS with optional MFA
3. Encrypt and store session credentials
4. Provide quick switching between stored sessions
5. Auto-refresh before expiration (daemon mode)

### Error Handling Pattern
```go
if err != nil {
    return fmt.Errorf("❌ Operation failed: %w\n\n💡 Common issues:\n   • Check X\n   • Verify Y", err)
}
```

### Interactive Prompt Pattern
```go
// Check for missing required parameter
if sourceProfile == "" {
    profiles := listAvailableProfiles()
    sourceProfile = promptUserSelection(profiles)
}
```