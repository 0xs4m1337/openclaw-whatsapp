# openclaw-whatsapp

WhatsApp bridge for [OpenClaw](https://openclaw.ai) agents. Single Go binary — start, scan QR, done.

## Quick Start

```bash
# Start the bridge
./openclaw-whatsapp start

# Open browser, scan QR
open http://localhost:8555/qr

# Send a message
./openclaw-whatsapp send "+1234567890" "Hello from OpenClaw!"
```

## Features

- **Always-on connection** — auto-reconnect with exponential backoff
- **REST API** — send text/files, read messages, search, list chats/contacts
- **QR Web UI** — scan from browser, auto-refreshes every 3s
- **Webhook delivery** — incoming messages POST to your endpoint
- **Full-text search** — SQLite FTS5 across all messages
- **Media handling** — auto-downloads images, videos, audio, documents
- **Message deduplication** — no duplicate webhooks
- **Single binary** — no runtime dependencies

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status` | Connection status, uptime, version |
| `GET` | `/qr` | QR code web page for device linking |
| `GET` | `/qr/data` | QR code as base64 PNG (JSON) |
| `POST` | `/logout` | Unlink device |
| `POST` | `/send/text` | Send text message `{"to": "+...", "message": "..."}` |
| `POST` | `/send/file` | Send file (multipart: `file`, `to`, `caption`) |
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
CGO_ENABLED=1 go build -tags "sqlite_fts5" -ldflags="-s -w" -o openclaw-whatsapp .
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
