# openclaw-whatsapp

WhatsApp bridge for [OpenClaw](https://openclaw.ai) agents. Single Go binary — start, scan QR, done.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/openclaw/whatsapp/main/install.sh | bash
```

Or build from source:

```bash
make build
sudo make install
```

## Quick Start

```bash
# Start the bridge
openclaw-whatsapp start

# Open browser, scan QR
open http://localhost:8555/qr

# Send a message
openclaw-whatsapp send "+1234567890" "Hello from OpenClaw!"
```

## Run as Service (systemd)

```bash
make install-service
systemctl --user start openclaw-whatsapp
journalctl --user -u openclaw-whatsapp -f
```

## Features

- **Always-on connection** — auto-reconnect with exponential backoff
- **REST API** — send text/files, read messages, search, list chats/contacts
- **QR Web UI** — scan from browser, auto-refreshes every 3s
- **Webhook delivery** — incoming messages POST to your endpoint
- **Full-text search** — SQLite FTS5 across all messages
- **Media handling** — auto-downloads images, videos, audio, documents
- **Agent mode** — trigger OpenClaw agents on incoming messages (command or HTTP)
- **Message deduplication** — no duplicate webhooks
- **Single binary** — pure Go, no CGO, cross-compiles everywhere

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status` | Connection status, uptime, version |
| `GET` | `/qr` | QR code web page for device linking |
| `GET` | `/qr/data` | QR code as base64 PNG (JSON) |
| `POST` | `/logout` | Unlink device |
| `POST` | `/send/text` | Send text message `{"to": "+...", "message": "..."}` |
| `POST` | `/send/file` | Send file (multipart: `file`, `to`, `caption`) |
| `POST` | `/reply` | Agent reply `{"to": "jid", "message": "...", "quote_message_id": "..."}` |
| `GET` | `/messages?chat=JID&limit=50` | Get messages for a chat |
| `GET` | `/messages/search?q=keyword` | Full-text search |
| `GET` | `/chats` | List all chats with last message |
| `GET` | `/chats/{jid}/messages` | Messages for specific chat |
| `GET` | `/contacts` | List contacts |

## Configuration

Create `config.yaml` or use environment variables:

```yaml
port: 8555
data_dir: ~/.openclaw-whatsapp
webhook_url: http://localhost:1337/webhook/whatsapp
webhook_filters:
  dm_only: false
  ignore_groups: []
auto_reconnect: true
reconnect_interval: 30s
log_level: info
```

### Agent Mode

Trigger an OpenClaw agent whenever a WhatsApp message arrives. Supports command execution or HTTP POST:

```yaml
agent:
  enabled: true
  mode: "command"         # "command" or "http"
  command: "openclaw gateway wake --text 'WhatsApp from {name} ({from}): {message}' --mode now"
  http_url: ""            # POST endpoint for "http" mode
  reply_endpoint: "http://localhost:8555/reply"  # so agent knows where to send replies
  ignore_from_me: true    # don't trigger on own messages
  dm_only: false          # only trigger on DMs, not groups
  timeout: 30s            # command/HTTP timeout
```

**Command mode** — runs a shell command with template variables:
- `{from}` — sender JID
- `{name}` — sender push name
- `{message}` — message text
- `{chat_jid}` — chat JID
- `{type}` — message type (text, image, etc.)
- `{is_group}` — "true" or "false"
- `{group_name}` — group name (empty for DMs)
- `{message_id}` — WhatsApp message ID

**HTTP mode** — POSTs JSON to `http_url` with all message details plus `reply_endpoint`.

When agent triggers, a typing indicator is shown in the chat until the agent completes.

Agents can reply via `POST /reply`:
```json
{"to": "971558762351@s.whatsapp.net", "message": "Hello!", "quote_message_id": "optional"}
```

Environment variables: `OC_WA_AGENT_ENABLED`, `OC_WA_AGENT_MODE`, `OC_WA_AGENT_COMMAND`, `OC_WA_AGENT_HTTP_URL`, `OC_WA_AGENT_REPLY_ENDPOINT`, `OC_WA_AGENT_TIMEOUT`.

Environment variables use `OC_WA_` prefix: `OC_WA_PORT`, `OC_WA_WEBHOOK_URL`, etc.

## CLI

```bash
openclaw-whatsapp start [-c config.yaml]  # Start service
openclaw-whatsapp status [--addr URL]      # Check connection
openclaw-whatsapp send NUMBER MESSAGE      # Quick send
openclaw-whatsapp stop                     # Stop service
openclaw-whatsapp version                  # Print version
```

## Build

```bash
make build              # Build for current platform
make release            # Cross-compile all platforms
make clean              # Remove build artifacts
```

## Docker

```bash
docker build -t openclaw-whatsapp .
docker run -p 8555:8555 -v wa-data:/app/data openclaw-whatsapp
```

## Webhook Payload

```json
{
  "from": "971558762351@s.whatsapp.net",
  "name": "Sam",
  "message": "Hey!",
  "timestamp": 1708387200,
  "type": "text",
  "media_url": "",
  "chat_type": "dm",
  "group_name": "",
  "message_id": "ABC123"
}
```

## License

MIT
