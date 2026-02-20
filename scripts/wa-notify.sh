#!/bin/bash
# Bridge agent trigger → spawns isolated agentTurn that auto-replies on WhatsApp
NAME="$1"
MSG="$2"
JID="$3"

# JSON-escape
MSG_ESC=$(printf '%s' "$MSG" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
NAME_ESC=$(printf '%s' "$NAME" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
JID_ESC=$(printf '%s' "$JID" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")

# Fetch last 10 messages for conversation context
HISTORY=$(curl -s "http://localhost:8555/chats/${JID}/messages?limit=10" | python3 -c "
import json,sys
msgs = json.loads(sys.stdin.read())
lines = []
for m in reversed(msgs):
    sender = 'Lucy' if m.get('from','') == '' else '${NAME_ESC}'
    lines.append(f'{sender}: {m[\"content\"]}')
print('\\\\n'.join(lines))
" 2>/dev/null)

AT=$(date -u -d '+2 seconds' +%Y-%m-%dT%H:%M:%SZ)

PROMPT="You are Lucy 🌙 — a sharp, witty AI assistant. You received a WhatsApp DM.\\n\\nRecent conversation history:\\n${HISTORY}\\n\\nLatest message from ${NAME_ESC}: \\\"${MSG_ESC}\\\"\\n\\nReply by running:\\nexec command: openclaw-whatsapp send \\\"${JID_ESC}\\\" \\\"YOUR_REPLY_HERE\\\"\\n\\nRules:\\n- Be concise and natural (1-3 sentences)\\n- Match their energy and language\\n- You have conversation context above — use it\\n- Sign off with 🌙\\n- Do NOT announce the result — just send and stop"

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
