---
inclusion: always
---

# CloudCtl Code Conventions

## Go Code Style

### Error Handling
- Always provide helpful error messages with context
- Include troubleshooting hints in user-facing errors
- Use `fmt.Errorf` with error wrapping: `fmt.Errorf("failed to encrypt AccessKey: %w", err)`
- Validate user input early and provide clear feedback

### Security Practices
- Never log or print sensitive data (credentials, keys, tokens)
- Use secure file permissions: `0600` for files, `0700` for directories
- Clear sensitive data from memory when possible
- Always encrypt before storing credentials

### CLI Command Patterns
- Use Cobra's persistent flags for common options (`--secret`, `--profile`)
- Implement interactive prompts for missing required parameters
- Provide helpful command examples in usage text
- Include both short and long descriptions for commands

### Testing Conventions
- Create temporary directories for storage tests using `setupTestDir()`
- Test both success and failure scenarios
- Include edge cases (corrupt data, wrong keys, missing files)
- Use table-driven tests for multiple similar test cases

## User Experience Guidelines

### Visual Design
- Use consistent color coding:
  - 🟢 Green: Active sessions (>15min remaining)
  - 🟡 Yellow: Expiring sessions (≤15min remaining)
  - 🔴 Red: Expired sessions
  - 🔒 Lock: MFA sessions
- Include helpful onboarding messages for empty states
- Show progress with spinners for long-running operations

### Error Messages
- Start with ❌ emoji for errors
- Follow with 💡 for troubleshooting tips
- List available options when user provides invalid input
- Include example commands or ARN formats

### Interactive Elements
- Use fuzzy search for profile/role selection
- Mask MFA input with asterisks (`******`)
- Support backspace in masked input fields
- Provide clear instructions for interactive prompts

## File Organization

### Command Files (`cmd/`)
- One command per file
- Include comprehensive flag definitions
- Implement input validation and interactive prompts
- Follow naming pattern: `<command>.go` (e.g., `login.go`, `status.go`)

### Internal Packages
- Keep platform-specific code in separate files (`_darwin.go`, `_stub.go`)
- Use descriptive function names that indicate their purpose
- Group related functionality in logical packages (`ui/`, storage functions)
- Export only what's needed by other packages

### Configuration Storage
- Use JSON for structured data (credentials, roles, MFA devices)
- Store all user data in `~/.cloudctl/` directory
- Encrypt sensitive data before writing to disk
- Use consistent field naming across storage files