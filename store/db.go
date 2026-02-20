package store

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Message represents a single WhatsApp message stored in the database.
type Message struct {
	ID         string `json:"id"`
	ChatJID    string `json:"chat_jid"`
	SenderJID  string `json:"sender_jid"`
	SenderName string `json:"sender_name"`
	Content    string `json:"content"`
	MsgType    string `json:"msg_type"`
	MediaPath  string `json:"media_path,omitempty"`
	Timestamp  int64  `json:"timestamp"`
	IsFromMe   bool   `json:"is_from_me"`
	IsGroup    bool   `json:"is_group"`
	GroupName  string `json:"group_name,omitempty"`
}

// Chat represents a conversation summary for listing chats.
type Chat struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	LastMessage string `json:"last_message"`
	LastTime    int64  `json:"last_time"`
	IsGroup     bool   `json:"is_group"`
	UnreadCount int    `json:"unread_count"`
}

// MessageStore manages SQLite storage for WhatsApp messages.
type MessageStore struct {
	db *sql.DB
}

const createMessagesTable = `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    chat_jid TEXT NOT NULL,
    sender_jid TEXT NOT NULL,
    sender_name TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    msg_type TEXT NOT NULL DEFAULT 'text',
    media_path TEXT NOT NULL DEFAULT '',
    timestamp INTEGER NOT NULL,
    is_from_me INTEGER NOT NULL DEFAULT 0,
    is_group INTEGER NOT NULL DEFAULT 0,
    group_name TEXT NOT NULL DEFAULT ''
);
`

const createFTSTable = `
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    sender_name,
    content='messages',
    content_rowid='rowid'
);
`

const createFTSTrigger = `
CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content, sender_name)
    VALUES (new.rowid, new.content, new.sender_name);
END;
`

const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_messages_chat_jid ON messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
`

// NewMessageStore opens (or creates) the SQLite database at dbPath, initialises
// the schema (messages table, FTS5 virtual table, sync trigger), and returns a
// ready-to-use MessageStore.
func NewMessageStore(dbPath string) (*MessageStore, error) {
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify the connection is alive.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Run schema migrations.
	for _, stmt := range []string{
		createMessagesTable,
		createFTSTable,
		createFTSTrigger,
		createIndexes,
	} {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec schema statement: %w", err)
		}
	}

	return &MessageStore{db: db}, nil
}

// SaveMessage inserts a message into the database. If a message with the same
// ID already exists the insert is silently ignored (deduplication).
func (s *MessageStore) SaveMessage(msg *Message) error {
	const query = `
		INSERT OR IGNORE INTO messages
			(id, chat_jid, sender_jid, sender_name, content, msg_type, media_path, timestamp, is_from_me, is_group, group_name)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		msg.ID,
		msg.ChatJID,
		msg.SenderJID,
		msg.SenderName,
		msg.Content,
		msg.MsgType,
		msg.MediaPath,
		msg.Timestamp,
		boolToInt(msg.IsFromMe),
		boolToInt(msg.IsGroup),
		msg.GroupName,
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

// GetMessages returns messages for a given chat, ordered by timestamp
// descending (newest first). Use limit and offset for pagination.
func (s *MessageStore) GetMessages(chatJID string, limit, offset int) ([]Message, error) {
	const query = `
		SELECT id, chat_jid, sender_jid, sender_name, content, msg_type, media_path,
		       timestamp, is_from_me, is_group, group_name
		FROM messages
		WHERE chat_jid = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chatJID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// SearchMessages performs a full-text search across message content and sender
// names using the FTS5 index. Results are ranked by relevance.
func (s *MessageStore) SearchMessages(query string, limit int) ([]Message, error) {
	// Escape any double quotes in the query to avoid FTS5 syntax errors.
	escaped := strings.ReplaceAll(query, `"`, `""`)
	ftsQuery := fmt.Sprintf(`"%s"`, escaped)

	const q = `
		SELECT m.id, m.chat_jid, m.sender_jid, m.sender_name, m.content, m.msg_type,
		       m.media_path, m.timestamp, m.is_from_me, m.is_group, m.group_name
		FROM messages m
		JOIN messages_fts fts ON m.rowid = fts.rowid
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := s.db.Query(q, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetChats returns a list of distinct chats with their most recent message,
// ordered by the last message timestamp (newest first).
func (s *MessageStore) GetChats(limit int) ([]Chat, error) {
	const query = `
		SELECT
			m.chat_jid,
			COALESCE(
				CASE WHEN m.is_group = 1 THEN m.group_name ELSE m.sender_name END,
				m.chat_jid
			) AS name,
			m.content AS last_message,
			m.timestamp AS last_time,
			m.is_group
		FROM messages m
		INNER JOIN (
			SELECT chat_jid, MAX(timestamp) AS max_ts
			FROM messages
			GROUP BY chat_jid
		) latest ON m.chat_jid = latest.chat_jid AND m.timestamp = latest.max_ts
		ORDER BY m.timestamp DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("get chats: %w", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		var isGroup int
		if err := rows.Scan(&c.JID, &c.Name, &c.LastMessage, &c.LastTime, &isGroup); err != nil {
			return nil, fmt.Errorf("scan chat row: %w", err)
		}
		c.IsGroup = isGroup != 0
		chats = append(chats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat rows: %w", err)
	}

	return chats, nil
}

// Close closes the underlying database connection.
func (s *MessageStore) Close() error {
	return s.db.Close()
}

// --- helpers ----------------------------------------------------------------

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func scanMessages(rows *sql.Rows) ([]Message, error) {
	var msgs []Message
	for rows.Next() {
		var m Message
		var isFromMe, isGroup int
		if err := rows.Scan(
			&m.ID, &m.ChatJID, &m.SenderJID, &m.SenderName,
			&m.Content, &m.MsgType, &m.MediaPath,
			&m.Timestamp, &isFromMe, &isGroup, &m.GroupName,
		); err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		m.IsFromMe = isFromMe != 0
		m.IsGroup = isGroup != 0
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message rows: %w", err)
	}
	return msgs, nil
}
