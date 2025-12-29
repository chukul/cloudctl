# CloudCtl Product Overview

CloudCtl is a lightweight CLI tool for securely managing AWS AssumeRole sessions with MFA support and encrypted credential storage.

## Core Purpose
- Secure AWS credential management with AES-256-GCM encryption
- Simplified AssumeRole workflow with MFA support
- Session management and auto-refresh capabilities
- macOS Keychain integration with Touch ID support

## Key Features
- **Secure Storage**: Encrypted AWS credentials with system keychain integration
- **MFA Support**: MFA session caching (12-hour validity) for multiple role assumptions
- **Session Management**: List, refresh, switch between active sessions
- **Console Access**: Generate AWS console URLs instantly
- **Shell Integration**: Display current session in shell prompt
- **Auto-Refresh Daemon**: Background service to keep sessions alive (macOS)
- **Role/MFA Aliasing**: Save frequently used roles and MFA devices with friendly names

## Target Users
- DevOps engineers managing multiple AWS accounts
- Developers working with AWS services requiring role switching
- Teams needing secure, auditable AWS credential management
- macOS users wanting Touch ID integration for passwordless operation

## Security Model
- All credentials encrypted with AES-256-GCM
- 32-character encryption keys
- macOS Keychain integration for key storage
- No plaintext credential storage
- Session-based access with configurable expiration

## Business Logic
- **MFA Session Caching**: 12-hour MFA sessions enable multiple role assumptions without re-entering MFA
- **Auto-Refresh**: Background daemon prevents session expiration
- **Shell Integration**: Current session displayed in prompt with remaining time
- **Credential Sync**: Export to `~/.aws/credentials` for tool compatibility
- **Role Aliasing**: Friendly names for frequently used IAM roles and MFA devices