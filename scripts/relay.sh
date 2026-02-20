#!/bin/bash
# Relay script — called by bridge agent mode when a WhatsApp message arrives
# Wakes the OpenClaw agent AND triggers a response turn

FROM="$1"
NAME="$2"  
MESSAGE="$3"
CHAT_JID="$4"

# Wake the main session with the message as a user-facing event
openclaw gateway call wake --params "{\"text\":\"[WhatsApp from ${NAME}]: ${MESSAGE}\nReply via: curl -s -X POST http://localhost:8555/send/text -H 'Content-Type: application/json' -d '{\\\"to\\\":\\\"${CHAT_JID}\\\",\\\"message\\\":\\\"YOUR_REPLY\\\"}'\",\"mode\":\"now\"}" --timeout 5000 2>/dev/null
