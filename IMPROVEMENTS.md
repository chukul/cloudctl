# UI/UX Improvements

## Implemented Changes

### #2: Better Visual Hierarchy in Status Display

**Before:**
```
PROFILE         ROLE ARN                                 EXPIRATION           REMAINING    STATUS  
----------------------------------------------------------------------------------------------------
prod-admin      arn:aws:iam::123456789012:role/AdminRole 2025-11-20 10:30:00  45m30s       ACTIVE
```

**After:**
```
Active Sessions
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
🟢 prod-admin ← current      AdminRole (123456789012)                       45m remaining
   Expires: 2025-11-20 10:30:00

🔒 mfa-session               MFA Session                                    11h45m remaining
   Expires: 2025-11-20 22:30:00

Expiring Soon
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
🟡 staging                   DevOpsRole (987654321098)                      12m remaining
   Expires: 2025-11-20 09:42:00

Expired Sessions
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
🔴 old-session               ReadOnlyRole (111222333444)                    expired
   Expires: 2025-11-19 18:00:00
```

**Features:**
- ✅ Status icons: 🟢 Active | 🟡 Expiring | 🔴 Expired | 🔒 MFA Session
- ✅ Grouped by status (Active → Expiring → Expired)
- ✅ Account ID extraction from role ARN
- ✅ Role name extraction (cleaner display)
- ✅ Current session highlighting (← current)
- ✅ Better time formatting (11h45m instead of 11h45m30s)
- ✅ Dim styling for expired/secondary info
- ✅ Helpful onboarding message when no sessions exist

### #5: Better Error Messages

#### Missing Parameters
**Before:**
```
❌ You must specify --source, --profile, and --role.
```

**After:**
```
❌ Missing required parameters
   --source: Source AWS profile or cloudctl session
   --profile: Name for this session
   --role: IAM role ARN to assume

💡 Example:
   cloudctl login --source default --profile prod-admin --role arn:aws:iam::123456789012:role/AdminRole
```

#### Profile Not Found
**Before:**
```
Failed to load source profile default: ...
```

**After:**
```
❌ Profile 'default' not found

💡 Available AWS profiles:
   • prod
   • dev
   • staging

💡 Available cloudctl sessions:
   • mfa-session
   • prod-admin

💡 To create a new profile:
   aws configure --profile default
```

#### MFA Authentication Failed
**Before:**
```
❌ Failed to get session token with MFA: ...
```

**After:**
```
❌ MFA authentication failed: InvalidClientTokenId: The security token included in the request is invalid

💡 Common issues:
   • Check your MFA code is current (not expired)
   • Verify MFA device ARN is correct
   • Ensure device time is synchronized
   • MFA ARN format: arn:aws:iam::<account-id>:mfa/<username>
```

#### AssumeRole Failed
**Before:**
```
❌ Failed to assume role: ...
```

**After:**
```
❌ Failed to assume role: AccessDenied: User is not authorized to perform: sts:AssumeRole

💡 Common issues:
   • Check the role ARN is correct
   • Verify the role's trust policy allows your source identity
   • Ensure your source credentials have sts:AssumeRole permission
   • Check if the role requires MFA (use --mfa flag)

💡 Role ARN format: arn:aws:iam::<account-id>:role/<role-name>
```

#### Switch Profile Not Found
**Before:**
```
❌ Failed to load session for profile 'prod': ...
```

**After:**
```
❌ Profile 'prod' not found

💡 Available profiles:
   • mfa-session
   • prod-admin
   • dev-readonly
```

#### Missing Secret
**Before:**
```
❌ You must specify --secret or set CLOUDCTL_SECRET environment variable
```

**After:**
```
❌ Encryption secret required

💡 Set the secret:
   export CLOUDCTL_SECRET="your-32-char-encryption-key"
   eval $(cloudctl switch prod-admin)
```

## Technical Changes

### Files Modified

1. **cmd/status.go**
   - Added session grouping by status
   - Added icon-based status indicators
   - Added account ID and role name extraction
   - Added current session detection and highlighting
   - Added helpful onboarding message for empty state
   - Improved time formatting

2. **cmd/login.go**
   - Added detailed parameter validation with examples
   - Added AWS profile listing on error
   - Added cloudctl session listing on error
   - Added MFA troubleshooting tips
   - Added AssumeRole troubleshooting tips
   - Added helper function `listAWSProfiles()`

3. **cmd/switch.go**
   - Added profile listing on error
   - Added helpful setup instructions
   - Improved secret missing message

4. **cmd/mfa-login.go**
   - Added detailed parameter validation
   - Added MFA troubleshooting tips
   - Added profile not found handling
   - Improved secret missing message

## Benefits

1. **Faster Troubleshooting** - Users can immediately see what's wrong and how to fix it
2. **Better Onboarding** - New users get helpful examples and guidance
3. **Reduced Support Burden** - Common issues are self-documented
4. **Professional UX** - Consistent emoji usage and formatting
5. **Improved Scannability** - Grouped sessions and icons make status easier to read
6. **Context Awareness** - Shows available options when something is missing

## Future Enhancements (Not Implemented)

- Interactive TUI mode with arrow key navigation
- Progress indicators for long operations
- Desktop notifications for expiring sessions
- Table filtering and sorting flags
- Daemon mode for background monitoring
- Quick action prompts after status display
