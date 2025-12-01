#!/usr/bin/env bash

# This script sends a notification to a Microsoft Teams channel
set -e

# Arguments
MESSAGE="$1"

# Get OAuth token
TOKEN_RESPONSE=$(curl -s -X POST "$MS_TEAMS_TOKEN_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=$MS_TEAMS_CLIENT_ID" \
  -d "client_secret=$MS_TEAMS_CLIENT_SECRET" \
  -d "scope=$MS_TEAMS_SCOPE" \
  -d "grant_type=client_credentials" 2>&1)

echo "::add-mask::$TOKEN_RESPONSE"
ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')
echo "::add-mask::$ACCESS_TOKEN"

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
  echo "Failed to get access token from MS Teams OAuth endpoint"
  exit 1
fi
echo "Successfully obtained access token"

# Send notification to Teams
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$MS_TEAMS_WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{
    \"body\": {
      \"contentType\": \"html\",
      \"content\": \"${MESSAGE}\"
    }
  }")

if [ "$HTTP_STATUS" -eq 200 ] || [ "$HTTP_STATUS" -eq 201 ] || [ "$HTTP_STATUS" -eq 202 ]; then
  echo "Successfully sent notification to MS Teams (HTTP $HTTP_STATUS)"
else
  echo "Failed to send notification to MS Teams (HTTP $HTTP_STATUS)"
  exit 1
fi
