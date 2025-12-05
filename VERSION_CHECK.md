# Version Update Notifications

CloudCtl automatically checks for new versions and notifies users when updates are available.

## How It Works

### Automatic Checks
- Runs in background on every command (non-blocking)
- Checks GitHub releases API once every 24 hours
- Caches last check in `~/.cloudctl/version_check.json`
- Silently fails if network unavailable (no error shown)

### Notification Display
When a new version is available:
```bash
💡 Update available: v1.0.0 → v1.1.0
   Download: https://github.com/chukul/cloudctl/releases/tag/v1.1.0
```

### Manual Check
Use the `version` command to force check:
```bash
cloudctl version
```

Output:
```bash
cloudctl version v1.0.0

💡 Update available: v1.0.0 → v1.1.0
   Download: https://github.com/chukul/cloudctl/releases/tag/v1.1.0
```

Or if up-to-date:
```bash
cloudctl version v1.0.0
✅ You're running the latest version
```

## Implementation Details

### Files
- `internal/version.go` - Version checking logic
- `cmd/version.go` - Version command
- `cmd/root.go` - Automatic check trigger

### Configuration
- **Check Interval**: 24 hours
- **Timeout**: 3 seconds
- **Cache Location**: `~/.cloudctl/version_check.json`
- **GitHub API**: `https://api.github.com/repos/chukul/cloudctl/releases/latest`

### Version Format
Uses semantic versioning (e.g., `v1.0.0`, `v1.2.3`)

## Updating the Version

When releasing a new version:

1. Update `internal/version.go`:
```go
const CurrentVersion = "v1.1.0" // Update this
```

2. Create GitHub release with matching tag:
```bash
git tag v1.1.0
git push origin v1.1.0
```

3. Create release on GitHub with tag `v1.1.0`

## Privacy & Performance

- **Non-blocking**: Runs in background goroutine
- **No tracking**: Only checks GitHub public API
- **Minimal overhead**: 3-second timeout, cached for 24 hours
- **Graceful failure**: No errors shown if check fails
- **No data sent**: Only reads from GitHub API

## Disabling (Future Enhancement)

To disable version checks, users could set:
```bash
export CLOUDCTL_SKIP_VERSION_CHECK=true
```

(Not currently implemented)
