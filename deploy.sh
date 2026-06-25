#!/bin/bash
# deploy.sh — build greetings for Linux, ship it to lab-core, restart the service.
set -e
SERVER="henry@lab.henryikoh.com"

echo "🔨 Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o greetings ./cmd/server

echo "📤 Uploading binary..."
# Upload to a temp name — you can't overwrite a binary while it's running.
scp greetings "$SERVER":/home/henry/greetings.new

echo "♻️  Swapping in new binary & restarting..."
# Atomic rename over the live binary, then restart to load it.
ssh "$SERVER" 'mv -f /home/henry/greetings.new /home/henry/greetings && sudo systemctl restart greetings'

echo "✅ Deployed → https://greetings.henryikoh.com"
