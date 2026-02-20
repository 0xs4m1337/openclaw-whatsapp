#!/bin/bash
# Bridge agent trigger → spawns isolated agentTurn that auto-replies on WhatsApp
# Called with: wa-notify.sh '{name}' '{message}' '{chat_jid}' '{system_prompt}'
NAME="$1"
MSG="$2"
JID="$3"
SYSTEM_PROMPT="$4"

# Default system prompt if none configured
if [ -z "$SYSTEM_PROMPT" ]; then
    SYSTEM_PROMPT="You are a helpful WhatsApp assistant. Be concise and natural."
fi

# JSON-escape
MSG_ESC=$(printf '%s' "$MSG" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
NAME_ESC=$(printf '%s' "$NAME" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
JID_ESC=$(printf '%s' "$JID" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
SP_ESC=$(printf '%s' "$SYSTEM_PROMPT" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")

# Fetch last 10 messages for conversation context
HISTORY=$(curl -s "http://localhost:8555/chats/${JID}/messages?limit=10" | python3 -c "
import json,sys
msgs = json.loads(sys.stdin.read())
lines = []
for m in reversed(msgs):
    sender = 'You' if m.get('from','') == '' else '${NAME_ESC}'
    lines.append(f'{sender}: {m[\"content\"]}')
print('\\\\n'.join(lines))
" 2>/dev/null)

AT=$(date -u -d '+2 seconds' +%Y-%m-%dT%H:%M:%SZ)

PROMPT="${SP_ESC}\\n\\nRecent conversation:\\n${HISTORY}\\n\\nNew message from ${NAME_ESC}: \\\"${MSG_ESC}\\\"\\n\\nReply by running:\\nexec command: openclaw-whatsapp send \\\"${JID_ESC}\\\" \\\"YOUR_REPLY_HERE\\\"\\n\\nDo NOT announce the result — just send the reply and stop."

openclaw gateway call cron.add --params "{
  \"job\": {
    \"name\": \"wa-reply\",
    \"schedule\": {\"kind\": \"at\", \"at\": \"${AT}\"},
    \"payload\": {\"kind\": \"agentTurn\", \"message\": \"${PROMPT}\", \"model\": \"anthropic/claude-sonnet-4-5\", \"timeoutSeconds\": 20},
    \"sessionTarget\": \"isolated\",
    \"delivery\": {\"mode\": \"none\"},
    \"deleteAfterRun\": true
  }
}" --timeout 10000 2>/dev/null

exit 0
