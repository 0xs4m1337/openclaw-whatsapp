# openclaw-whatsapp Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a standalone WhatsApp bridge service for OpenClaw agents — single Go binary with REST API, webhook delivery, SQLite FTS5 message store, QR web UI, and CLI.

**Architecture:** Single-process Go service wrapping whatsmeow for WhatsApp Web multi-device protocol. Bridge layer handles connection/QR/events, REST API exposes send/receive/search, SQLite stores messages with FTS5 for full-text search, webhooks push incoming messages to configurable URL. CLI uses cobra for start/stop/status/send commands.

**Tech Stack:** Go 1.22+, whatsmeow (WhatsApp Web), go-sqlite3 (CGO), chi (HTTP router), cobra (CLI), go-qrcode (QR generation), gopkg.in/yaml.v3 (config)

---

## Task 1: Go Module + Project Skeleton

**Files:**
- Create: `go.mod`
- Create: `main.go` (minimal — just prints version)
- Create: `.gitignore`

**Step 1: Initialize Go module and add dependencies**
**Step 2: Create .gitignore**
**Step 3: Create minimal main.go that prints version**
**Step 4: Verify build: `go build -o openclaw-whatsapp .`**
**Step 5: Commit**

---

## Task 2: Config Package

**Files:**
- Create: `config/config.go`

Loads config from YAML file + env var overrides. Flat struct, sensible defaults.

**Config struct fields:**
- Port (8555), DataDir (~/.openclaw-whatsapp), WebhookURL, WebhookFilters (DMOnly, IgnoreGroups), AutoReconnect (true), ReconnectInterval (30s), LogLevel (info)
- Env vars prefixed with `OC_WA_` (e.g. `OC_WA_PORT`, `OC_WA_WEBHOOK_URL`)

**Step 1: Write config/config.go**
**Step 2: Verify build: `go build ./...`**
**Step 3: Commit**

---

## Task 3: SQLite Message Store with FTS5

**Files:**
- Create: `store/db.go`

Single SQLite database at `{DataDir}/messages.db`. Two tables:
- `messages` — id, jid, chat_jid, sender_jid, sender_name, content, msg_type, media_path, timestamp, is_from_me, is_group, group_name
- `messages_fts` — FTS5 virtual table on (content, sender_name) for full-text search

**Functions:**
- `NewMessageStore(dbPath string) (*MessageStore, error)` — opens DB, creates tables
- `SaveMessage(msg *Message) error` — INSERT OR IGNORE (dedup by message ID)
- `GetMessages(chatJID string, limit, offset int) ([]Message, error)`
- `SearchMessages(query string, limit int) ([]Message, error)` — FTS5 search
- `GetChats(limit int) ([]Chat, error)` — distinct chats with last message
- `Close() error`

Uses `_journal_mode=WAL&_foreign_keys=on` pragmas.

**Step 1: Write store/db.go**
**Step 2: Verify build: `go build ./...`**
**Step 3: Commit**

---

## Task 4: Bridge Client (whatsmeow wrapper)

**Files:**
- Create: `bridge/client.go`

Single-device wrapper around whatsmeow. Adapted from Siventa device.go, simplified for single-device.

**Struct `Client`** with: whatsmeow client, sqlstore container, status, latestQR, qrChan, mutex, logger, event handler callback, startTime.

**Functions:**
- `NewClient(dataDir string, log *slog.Logger) (*Client, error)` — creates sqlstore container
- `Connect(ctx context.Context) error` — connect or start QR pairing
- `Disconnect()`, `Logout() error`
- `IsConnected() bool`, `GetStatus() Status`, `GetLatestQR() string`
- `GetClient() *whatsmeow.Client`
- `SendText(ctx, to, message) error`
- `SendFile(ctx, to, data, mimetype, filename, caption) error` — image/video/audio/document
- `GetJID() string`
- `processQRCodes()` — goroutine reading QR channel

JID parsing helper from Siventa's parseJID pattern.

**Step 1: Write bridge/client.go**
**Step 2: Verify build: `go build ./...`**
**Step 3: Commit**

---

## Task 5: Bridge Events + Webhook Delivery

**Files:**
- Create: `bridge/events.go`
- Create: `bridge/webhook.go`

**bridge/events.go:**
`MakeEventHandler(...)` handles: `*events.Message` (extract text/image/video/audio/document/sticker/location/contact, save to store, send webhook), `*events.Connected`, `*events.Disconnected`, `*events.LoggedOut`, `*events.StreamReplaced`.

Media download to `{DataDir}/media/{msgID}{ext}`.

**bridge/webhook.go:**
`WebhookSender` with HTTP client, target URL, dedup set (map + TTL cleanup).
- `NewWebhookSender(url string, filters config.WebhookFilters) *WebhookSender`
- `Send(payload *WebhookPayload) error` — POST JSON, dedup by message ID, apply filters
- Payload matches SPEC.md format

**Step 1: Write bridge/events.go**
**Step 2: Write bridge/webhook.go**
**Step 3: Verify build: `go build ./...`**
**Step 4: Commit**

---

## Task 6: Bridge QR Code + Auto-Reconnect

**Files:**
- Create: `bridge/qr.go`
- Create: `bridge/reconnect.go`

**bridge/qr.go:**
- `GenerateQRPNG(qrText string, size int) ([]byte, error)` — returns PNG bytes via go-qrcode

**bridge/reconnect.go:**
- `StartReconnectLoop(ctx context.Context, client *Client, interval time.Duration, log *slog.Logger)` — goroutine checking connection every interval, reconnects if disconnected with stored session, backs off on failure.

**Step 1: Write bridge/qr.go**
**Step 2: Write bridge/reconnect.go**
**Step 3: Verify build: `go build ./...`**
**Step 4: Commit**

---

## Task 7: REST API — Router + Status + QR Web Page

**Files:**
- Create: `api/router.go`
- Create: `api/status.go`
- Create: `api/qr.go`

**api/router.go:** chi router with all SPEC.md routes. Middleware: logging, CORS, recovery.

**api/status.go:** `GET /status` returns JSON (status, phone, uptime, version). `POST /logout` calls client.Logout().

**api/qr.go:** `GET /qr` serves clean HTML page with auto-refresh. Uses `fetch('/qr/data')` polling every 3s. QR displayed as `<img>` element created via DOM API (createElement, setAttribute — no innerHTML). Shows "Connected" state when linked. `GET /qr/data` returns JSON `{status, qr_png (base64), phone}`.

**Step 1: Write api/router.go**
**Step 2: Write api/status.go**
**Step 3: Write api/qr.go**
**Step 4: Verify build: `go build ./...`**
**Step 5: Commit**

---

## Task 8: REST API — Messaging + Search + Chats

**Files:**
- Create: `api/messages.go`
- Create: `api/contacts.go`

**api/messages.go:**
- `POST /send/text` — JSON body `{to, message}`
- `POST /send/file` — multipart form (file, to, caption)
- `GET /messages?chat=JID&limit=50&offset=0`
- `GET /messages/search?q=keyword&limit=20`

**api/contacts.go:**
- `GET /chats` — store.GetChats
- `GET /contacts` — whatsmeow client contacts store
- `GET /chats/{jid}/messages?limit=50`

**Step 1: Write api/messages.go**
**Step 2: Write api/contacts.go**
**Step 3: Verify build: `go build ./...`**
**Step 4: Commit**

---

## Task 9: CLI (Cobra) — Main Entrypoint

**Files:**
- Rewrite: `main.go` — cobra root + subcommands

**Subcommands:**
- `start` — loads config, creates store/client/webhook/API, runs until SIGTERM. Flag `--config` for config file path.
- `stop` — sends shutdown signal via PID file or admin endpoint
- `status` — calls GET /status, prints result
- `send <number> <message>` — calls POST /send/text
- `version` — prints version

The `start` command wires everything: config → store → client → webhook sender → event handler → connect → reconnect loop → HTTP server → signal wait → graceful shutdown.

**Step 1: Rewrite main.go**
**Step 2: Verify build and CLI: `./openclaw-whatsapp --help`, `./openclaw-whatsapp version`**
**Step 3: Commit**

---

## Task 10: Dockerfile

**Files:**
- Create: `Dockerfile`

Multi-stage: `golang:1.22-bookworm` builder (CGO_ENABLED=1), `debian:bookworm-slim` runtime.

**Step 1: Write Dockerfile**
**Step 2: Commit**

---

## Task 11: Final Build Verification

**Step 1: `go build -o openclaw-whatsapp .`** — must succeed
**Step 2: `go vet ./...`** — must pass
**Step 3: Test CLI: `./openclaw-whatsapp --help`, `./openclaw-whatsapp version`, `./openclaw-whatsapp start --help`**
**Step 4: Final commit**

---

## Parallel Execution Groups

**Group A (first):** Task 1 (go.mod — everything depends on this)
**Group B (parallel after A):** Task 2 (config), Task 3 (store), Task 6 (QR + reconnect)
**Group C (after B):** Task 4 (client), Task 5 (events + webhook)
**Group D (after C):** Task 7 (API status/QR), Task 8 (API messages/contacts)
**Group E (after D):** Task 9 (CLI wires all), Task 10 (Dockerfile)
**Group F (final):** Task 11 (verify)
