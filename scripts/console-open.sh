#!/bin/bash
# Usage: ./scripts/console-open.sh <profile> <secret>

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <profile> <secret>"
    exit 1
fi

PROFILE=$1
SECRET=$2

# Get credentials
SESSION_JSON=$(./cloudctl export --profile "$PROFILE" --secret "$SECRET" 2>/dev/null | grep -E "AWS_ACCESS_KEY_ID|AWS_SECRET_ACCESS_KEY|AWS_SESSION_TOKEN" | sed 's/export //' | awk -F= '{print "\""$1"\":\""$2"\","}' | tr '\n' ' ' | sed 's/AWS_ACCESS_KEY_ID/sessionId/; s/AWS_SECRET_ACCESS_KEY/sessionKey/; s/AWS_SESSION_TOKEN/sessionToken/; s/, $//')

if [ -z "$SESSION_JSON" ]; then
    echo "❌ Failed to load credentials"
    exit 1
fi

SESSION_JSON="{${SESSION_JSON}}"

# Get signin token
echo "🔐 Getting sign-in token..."
TOKEN_RESPONSE=$(curl -s "https://signin.aws.amazon.com/federation?Action=getSigninToken&Session=$(echo -n "$SESSION_JSON" | jq -sRr @uri)")
SIGNIN_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.SigninToken')

if [ "$SIGNIN_TOKEN" = "null" ] || [ -z "$SIGNIN_TOKEN" ]; then
    echo "❌ Failed to get sign-in token"
    echo "$TOKEN_RESPONSE"
    exit 1
fi

# Build console URL
CONSOLE_URL="https://signin.aws.amazon.com/federation?Action=login&Issuer=cloudctl&Destination=https%3A%2F%2Fconsole.aws.amazon.com%2F&SigninToken=${SIGNIN_TOKEN}"

echo "✅ Opening AWS Console in browser..."
echo "$CONSOLE_URL"

# Open in default browser
if command -v open &> /dev/null; then
    open "$CONSOLE_URL"
elif command -v xdg-open &> /dev/null; then
    xdg-open "$CONSOLE_URL"
else
    echo "Please open this URL manually:"
    echo "$CONSOLE_URL"
fi
