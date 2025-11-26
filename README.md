# CloudCtl

```
   ________                _________ __  __
  / ____/ /___  __  ______/ / ____// /_/ /
 / /   / / __ \/ / / / __  / /    / __/ / 
/ /___/ / /_/ / /_/ / /_/ / /___ / /_/ /  
\____/_/\____/\__,_/\__,_/\____/ \__/_/   
```

A lightweight CLI tool for securely managing AWS AssumeRole sessions with MFA support and encrypted credential storage.

## Features

- 🔐 **Secure Credential Storage** - Encrypt AWS credentials with AES-256-GCM
- 🎯 **AssumeRole Support** - Easily assume IAM roles with MFA
- 🌐 **Console Access** - Generate and auto-open AWS Console sign-in URLs
- 📦 **Session Management** - List, switch, and manage multiple AWS sessions
- 🔑 **MFA Support** - Built-in multi-factor authentication support
- 🔄 **MFA Session Caching** - Enter MFA once, assume unlimited roles for 12 hours
- 🌍 **Region Support** - Default region configuration (ap-southeast-1)
- 🚀 **Quick Switch** - Fast profile switching with one command
- 🎨 **Color-Coded Status** - Visual indicators for session health (active/expiring/expired)
- 💻 **Shell Integration** - Display current session in your shell prompt
- 🔄 **Auto Refresh** - Renew sessions before they expire with bulk operations
- ⚡ **Shell Init** - One-command setup for seamless shell integration
- 🕐 **Local Timezone** - All timestamps display in your local timezone
- 🏷️ **Static Session Names** - Profile names shown in AWS Console instead of random IDs
- 🔒 **Masked MFA Input** - Asterisk display (`******`) for MFA codes with backspace support

## Installation

### Prerequisites

- Go 1.22 or higher
- AWS CLI configured with at least one profile
- Valid AWS credentials

### Build from Source

```bash
# Clone the repository
git clone https://github.com/chukul/cloudctl.git
cd cloudctl

# Build the binary
go build

# (Optional) Move to PATH
sudo mv cloudctl /usr/local/bin/
```

### Shell Integration (Recommended)

Set up shell integration for the best experience:

```bash
# Generate and add to your shell config
cloudctl init >> ~/.zshrc  # or ~/.bashrc for Bash

# Edit ~/.zshrc and set your encryption secret
export CLOUDCTL_SECRET="your-32-char-encryption-key"

# Reload your shell
source ~/.zshrc
```

After setup, you get:
- `ccs <profile>` - Quick switch without eval
- `ccl` - Alias for cloudctl login
- `ccst` - Alias for cloudctl status
- `ccr` - Alias for cloudctl refresh
- `ccc` - Alias for cloudctl console
- Shell prompt showing current session and remaining time

## Quick Start

### 1. MFA Session (Recommended for Multiple Roles)

If you need to assume multiple roles with MFA, get an MFA session first:

```bash
# Step 1: Get MFA session once (valid for 12 hours)
cloudctl mfa-login \
  --source default \
  --profile mfa-session \
  --mfa arn:aws:iam::123456789012:mfa/username
# Enter MFA code when prompted (displays as ******)

# Step 2: Use MFA session to assume multiple roles (no MFA needed!)
cloudctl login --source mfa-session --profile prod-admin --role arn:aws:iam::123456789012:role/AdminRole
cloudctl login --source mfa-session --profile dev-readonly --role arn:aws:iam::123456789012:role/ReadOnlyRole
cloudctl login --source mfa-session --profile staging --role arn:aws:iam::987654321098:role/StagingRole
```

### 2. Login and Assume Role

Assume an IAM role and store the credentials securely:

```bash
cloudctl login \
  --source <source-profile> \
  --profile <session-name> \
  --role <role-arn> \
  --secret "your-32-char-encryption-key" \
  --region ap-southeast-1
```

**Example:**
```bash
cloudctl login \
  --source default \
  --profile prod-admin \
  --role arn:aws:iam::123456789012:role/AdminRole \
  --secret "1234567890ABCDEF1234567890ABCDEF"
```

**With MFA (single role):**
```bash
cloudctl login \
  --source default \
  --profile prod-admin \
  --role arn:aws:iam::123456789012:role/AdminRole \
  --mfa arn:aws:iam::123456789012:mfa/username \
  --secret "1234567890ABCDEF1234567890ABCDEF"
```

**With auto-open console:**
```bash
cloudctl login \
  --source mfa-session \
  --profile uat_ca \
  --role arn:aws:iam::814348778342:role/CIMBTH_CloudAdministrator \
  --open
```

### 3. View Stored Sessions

List all stored sessions with their status:

```bash
cloudctl status --secret "1234567890ABCDEF1234567890ABCDEF"
```

**Output:**
```
PROFILE         ROLE ARN                                 EXPIRATION           REMAINING    STATUS  
----------------------------------------------------------------------------------------------------
prod-admin      arn:aws:iam::123456789012:role/AdminRole 2025-11-20 10:30:00  45m30s       ACTIVE
mfa-session                                              2025-11-20 22:30:00  11h45m       ACTIVE
```

**Status Colors:**
- 🟢 Green (ACTIVE) - Session has more than 15 minutes remaining
- 🟡 Yellow (EXPIRING) - Session expires in 15 minutes or less
- 🔴 Red (EXPIRED) - Session has expired

### 4. Quick Switch Between Profiles

Fast profile switching with one command:

```bash
# Set your secret once
export CLOUDCTL_SECRET="1234567890ABCDEF1234567890ABCDEF"

# Switch to any profile instantly
eval $(cloudctl switch prod-admin)
eval $(cloudctl switch dev-readonly)

# Or use the shell function (if init is configured)
ccs prod-admin

# Verify
aws sts get-caller-identity
```

**Important:** Unset `AWS_PROFILE` if it's already set in your environment:
```bash
unset AWS_PROFILE
eval $(cloudctl switch prod-admin)
```

### 5. Open AWS Console

Generate and open AWS Console in your browser:

```bash
# Open console in default region (ap-southeast-1)
cloudctl console --profile prod-admin --secret "1234567890ABCDEF1234567890ABCDEF" --open

# Open console in specific region
cloudctl console --profile prod-admin --secret "1234567890ABCDEF1234567890ABCDEF" --region us-east-1 --open

# Or use the alias (if init is configured)
ccc --profile prod-admin --open
```

**Note:** MFA sessions cannot be used for console access. Use an assumed role profile instead.

### 6. Refresh Sessions

Refresh sessions before they expire:

```bash
# Refresh single profile
cloudctl refresh --profile prod-admin --secret "1234567890ABCDEF1234567890ABCDEF"

# Refresh all active sessions
cloudctl refresh --all --secret "1234567890ABCDEF1234567890ABCDEF"

# Or use the alias (if init is configured)
ccr --all
```

**Note:** MFA sessions cannot be refreshed. Use `mfa-login` to create a new one.

### 7. Logout

Remove stored credentials:

```bash
# Remove specific profile
cloudctl logout --profile prod-admin

# Remove all profiles
cloudctl logout --all
```

## Commands Reference

### `mfa-login`

Get MFA session token to use for multiple role assumptions.

**Flags:**
- `--source` - Source AWS CLI profile for base credentials (required)
- `--profile` - Name to store the MFA session as (required)
- `--mfa` - MFA device ARN (required)
- `--secret` - Encryption key for credential storage (or set CLOUDCTL_SECRET env var)
- `--duration` - Session duration in seconds (default: 43200 = 12 hours, max: 129600 = 36 hours)

**Usage:**
```bash
# Get MFA session (valid for 12 hours)
cloudctl mfa-login --source default --profile mfa-session --mfa arn:aws:iam::123:mfa/user

# Use MFA session to assume roles without re-entering MFA
cloudctl login --source mfa-session --profile role1 --role arn:aws:iam::123:role/Role1
cloudctl login --source mfa-session --profile role2 --role arn:aws:iam::456:role/Role2
```

### `login`

Assume an AWS role and store credentials locally.

**Flags:**
- `--source` - Source AWS CLI profile or cloudctl session for base credentials (required)
- `--profile` - Name to store the new session as (required)
- `--role` - Target IAM role ARN to assume (required)
- `--mfa` - MFA device ARN (optional)
- `--secret` - Encryption key for credential storage (or set CLOUDCTL_SECRET env var)
- `--region` - AWS region (default: ap-southeast-1)
- `--open` - Automatically open AWS Console after successful login

**Usage:**
```bash
# Basic login
cloudctl login --source default --profile prod --role arn:aws:iam::123:role/Admin

# With MFA
cloudctl login --source default --profile prod --role arn:aws:iam::123:role/Admin --mfa arn:aws:iam::123:mfa/user

# With auto-open console
cloudctl login --source mfa-session --profile prod --role arn:aws:iam::123:role/Admin --open
```

### `status`

Show all stored AWS sessions with color-coded expiration status.

**Flags:**
- `--secret` - Encryption key to decrypt credentials (or set CLOUDCTL_SECRET env var)

**Usage:**
```bash
cloudctl status
# or
ccst  # if shell integration is configured
```

### `switch`

Quick switch to a profile and export credentials in one command.

**Flags:**
- `<profile>` - Profile name to switch to (required)
- `--secret` - Encryption key to decrypt credentials (or set CLOUDCTL_SECRET env var)

**Usage:**
```bash
export CLOUDCTL_SECRET="your-secret"
eval $(cloudctl switch prod-admin)

# Or use the shell function
ccs prod-admin
```

### `console`

Generate AWS Console sign-in URL from stored session.

**Flags:**
- `--profile` - Profile to generate console URL for (required)
- `--secret` - Encryption key to decrypt credentials (or set CLOUDCTL_SECRET env var)
- `--region` - AWS region for console (default: ap-southeast-1)
- `--open` - Automatically open URL in browser

**Usage:**
```bash
# Generate and open console
cloudctl console --profile prod-admin --open

# Or use the alias
ccc --profile prod-admin --open
```

**Note:** MFA sessions cannot be used for console access. Use an assumed role profile instead.

### `refresh`

Refresh AWS session credentials before expiration.

**Flags:**
- `--profile` - Profile to refresh (required unless using --all)
- `--all` - Refresh all active sessions
- `--secret` - Encryption key to decrypt credentials (or set CLOUDCTL_SECRET env var)

**Usage:**
```bash
# Refresh single profile
cloudctl refresh --profile prod-admin

# Refresh all active sessions
cloudctl refresh --all

# Or use the alias
ccr --all
```

**Note:** 
- MFA sessions cannot be refreshed. Use `mfa-login` to create a new one.
- Only sessions with source profile information can be refreshed.

### `list`

List all stored profile names.

**Usage:**
```bash
cloudctl list
```

### `init`

Generate shell integration code for easy setup.

**Usage:**
```bash
# Add to your shell config
cloudctl init >> ~/.zshrc

# Or view the output
cloudctl init
```

**What it provides:**
- `ccs <profile>` - Quick switch function without eval
- `ccl` - Alias for cloudctl login
- `ccst` - Alias for cloudctl status
- `ccr` - Alias for cloudctl refresh
- `ccc` - Alias for cloudctl console
- Shell prompt integration showing current session
- CLOUDCTL_SECRET environment variable setup

### `prompt`

Display current session info for shell prompt integration.

**Subcommands:**
- `cloudctl prompt` - Display formatted prompt string (e.g., ☁️ prod-admin (45m))
- `cloudctl prompt info` - Display detailed session info in JSON format
- `cloudctl prompt setup` - Show shell integration setup instructions

**Flags:**
- `--secret` - Encryption key to decrypt credentials (or set CLOUDCTL_SECRET env var)

**Usage:**
```bash
# In your shell prompt (PS1 or PROMPT)
$(cloudctl prompt)

# Get detailed info
cloudctl prompt info
```

### `logout`

Remove stored credentials.

**Flags:**
- `--profile` - Profile to remove
- `--all` - Remove all profiles

**Usage:**
```bash
# Remove specific profile
cloudctl logout --profile prod-admin

# Remove all profiles
cloudctl logout --all
```

## Configuration

### Encryption Key

CloudCtl uses AES-256-GCM encryption for storing credentials. Your encryption key should be:
- Exactly 32 characters long
- Kept secure and not shared
- The same key must be used for storing and retrieving credentials

**Example key generation:**
```bash
# Generate a random 32-character key
openssl rand -hex 16
```

**Set as environment variable:**
```bash
export CLOUDCTL_SECRET="1234567890ABCDEF1234567890ABCDEF"
```

### Storage Location

Credentials are stored in:
```
~/.cloudctl/credentials.json  # Encrypted credentials
~/.cloudctl/sessions/         # Session files
```

These files contain encrypted credentials and should be kept secure.

## Security Best Practices

1. **Use Strong Encryption Keys** - Generate random 32-character keys
2. **Don't Commit Secrets** - Never commit your encryption key or credentials
3. **Rotate Sessions** - Regularly refresh your assumed role sessions
4. **Use MFA** - Enable MFA for sensitive role assumptions
5. **Limit Session Duration** - Use appropriate session durations (default: 1 hour for roles, 12 hours for MFA)
6. **Secure Storage** - Ensure `~/.cloudctl/` directory has proper permissions (0700)

## Troubleshooting

### "The config profile (X) could not be found"

This error occurs when `AWS_PROFILE` is set in your environment. Unset it:
```bash
unset AWS_PROFILE
eval $(cloudctl switch prod-admin)
```

### "Failed to load source profile"

Ensure your source profile exists in `~/.aws/credentials`:
```bash
cat ~/.aws/credentials
aws configure list-profiles
```

### "Invalid secret key"

The encryption key used for decryption doesn't match the one used for encryption. Ensure you're using the same 32-character key.

### "Failed to assume role"

Check that:
1. Your source profile has valid credentials
2. The role ARN is correct
3. Your source credentials have permission to assume the role
4. The role's trust policy allows your source identity

### MFA Code Not Working

- Ensure your device time is synchronized (MFA codes are time-based)
- Wait for a fresh code if the current one is about to expire
- Verify the MFA device ARN is correct

### Console URL Not Opening

- Check that you're using an assumed role profile (not an MFA session)
- Verify the session hasn't expired
- Ensure your browser is set as the default application for URLs

## Development

### Project Structure

```
cloudctl/
├── cmd/              # Command implementations
│   ├── console.go    # Console sign-in command
│   ├── init.go       # Shell integration command
│   ├── list.go       # List profiles command
│   ├── login.go      # Login/assume role command
│   ├── logout.go     # Logout command
│   ├── mfa-login.go  # MFA session command
│   ├── prompt.go     # Shell prompt command
│   ├── refresh.go    # Refresh session command
│   ├── root.go       # Root command and CLI setup
│   ├── status.go     # Status command
│   ├── switch.go     # Quick switch command
│   └── utils.go      # Shared utilities (MFA input)
├── internal/         # Internal packages
│   ├── aws.go        # AWS SDK helpers
│   ├── crypto.go     # Encryption/decryption
│   ├── session.go    # Session types
│   ├── storage.go    # Credential storage
│   └── types.go      # Type definitions
├── go.mod
├── go.sum
├── main.go
└── README.md
```

### Building

```bash
# Build for current platform
go build

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o cloudctl-linux
GOOS=darwin GOARCH=arm64 go build -o cloudctl-macos
GOOS=windows GOARCH=amd64 go build -o cloudctl.exe
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is open source and available under the MIT License.

## Author

**Chuchai Kul**  
Email: chuchaik@outlook.com

## Acknowledgments

- Inspired by [Leapp](https://www.leapp.cloud/)
- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Uses [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)

## Support

For issues, questions, or contributions, please visit:
- GitHub Issues: https://github.com/chukul/cloudctl/issues
- GitHub Repository: https://github.com/chukul/cloudctl
