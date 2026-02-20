package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// AgentTrigger handles waking an OpenClaw agent when a message arrives.
type AgentTrigger struct {
	enabled       bool
	mode          string // "command" or "http"
	command       string
	httpURL       string
	replyEndpoint string
	systemPrompt  string
	ignoreFromMe  bool
	dmOnly        bool
	allowlist     map[string]bool
	blocklist     map[string]bool
	timeout       time.Duration
	client        *http.Client
	log           *slog.Logger
}

// AgentPayload is the JSON body sent to the agent in HTTP mode.
type AgentPayload struct {
	From          string `json:"from"`
	Name          string `json:"name,omitempty"`
	Message       string `json:"message"`
	ChatJID       string `json:"chat_jid"`
	Type          string `json:"type"`
	IsGroup       bool   `json:"is_group"`
	GroupName     string `json:"group_name,omitempty"`
	MessageID     string `json:"message_id"`
	Timestamp     int64  `json:"timestamp"`
	ReplyEndpoint string `json:"reply_endpoint,omitempty"`
	SystemPrompt  string `json:"system_prompt,omitempty"`
}

// NewAgentTrigger creates a new AgentTrigger. If enabled is false, Trigger is a
// no-op.
func NewAgentTrigger(enabled bool, mode, command, httpURL, replyEndpoint, systemPrompt string, ignoreFromMe, dmOnly bool, allowlist, blocklist []string, timeout time.Duration, log *slog.Logger) *AgentTrigger {
	al := make(map[string]bool)
	for _, v := range allowlist {
		al[normalizeNumber(v)] = true
	}
	bl := make(map[string]bool)
	for _, v := range blocklist {
		bl[normalizeNumber(v)] = true
	}
	return &AgentTrigger{
		enabled:       enabled,
		mode:          mode,
		command:       command,
		httpURL:       httpURL,
		replyEndpoint: replyEndpoint,
		systemPrompt:  systemPrompt,
		ignoreFromMe:  ignoreFromMe,
		dmOnly:        dmOnly,
		allowlist:     al,
		blocklist:     bl,
		timeout:       timeout,
		client:        &http.Client{Timeout: timeout},
		log:           log,
	}
}

// normalizeNumber strips @s.whatsapp.net suffix for comparison.
func normalizeNumber(s string) string {
	s = strings.TrimSuffix(s, "@s.whatsapp.net")
	s = strings.TrimPrefix(s, "+")
	return s
}

// SystemPrompt returns the configured system prompt.
func (a *AgentTrigger) SystemPrompt() string {
	return a.systemPrompt
}

// Trigger fires the agent for an incoming message. It sends a typing indicator,
// then runs the configured command or HTTP call asynchronously.
func (a *AgentTrigger) Trigger(client *Client, payload *WebhookPayload) {
	if !a.enabled {
		return
	}

	// Apply filters.
	if a.dmOnly && payload.ChatType == "group" {
		a.log.Debug("agent skipping group message (dm_only)", "message_id", payload.MessageID)
		return
	}

	sender := normalizeNumber(payload.From)
	if len(a.blocklist) > 0 && a.blocklist[sender] {
		a.log.Debug("agent skipping blocklisted sender", "from", payload.From, "message_id", payload.MessageID)
		return
	}
	if len(a.allowlist) > 0 && !a.allowlist[sender] {
		a.log.Debug("agent skipping non-allowlisted sender", "from", payload.From, "message_id", payload.MessageID)
		return
	}

	// Send typing indicator.
	a.sendTyping(client, payload.From)

	// Run async — don't block the event loop.
	go func() {
		defer a.clearTyping(client, payload.From)

		switch a.mode {
		case "http":
			a.triggerHTTP(payload)
		default:
			a.triggerCommand(payload)
		}
	}()
}

// triggerCommand executes a shell command with template variables substituted.
func (a *AgentTrigger) triggerCommand(payload *WebhookPayload) {
	if a.command == "" {
		a.log.Warn("agent command mode enabled but no command configured")
		return
	}

	cmd := a.expandTemplate(a.command, payload)

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	a.log.Info("agent triggering command", "command", cmd, "message_id", payload.MessageID)

	proc := exec.CommandContext(ctx, "sh", "-c", cmd)
	output, err := proc.CombinedOutput()
	if err != nil {
		a.log.Error("agent command failed", "error", err, "output", string(output), "message_id", payload.MessageID)
		return
	}

	a.log.Info("agent command completed", "output", string(output), "message_id", payload.MessageID)
}

// triggerHTTP POSTs message details to the configured HTTP endpoint.
func (a *AgentTrigger) triggerHTTP(payload *WebhookPayload) {
	if a.httpURL == "" {
		a.log.Warn("agent http mode enabled but no http_url configured")
		return
	}

	agentPayload := &AgentPayload{
		From:          payload.From,
		Name:          payload.Name,
		Message:       payload.Message,
		ChatJID:       payload.From,
		Type:          payload.Type,
		IsGroup:       payload.ChatType == "group",
		GroupName:     payload.GroupName,
		MessageID:     payload.MessageID,
		Timestamp:     payload.Timestamp,
		ReplyEndpoint: a.replyEndpoint,
		SystemPrompt:  a.systemPrompt,
	}

	body, err := json.Marshal(agentPayload)
	if err != nil {
		a.log.Error("agent marshal payload failed", "error", err, "message_id", payload.MessageID)
		return
	}

	a.log.Info("agent triggering http", "url", a.httpURL, "message_id", payload.MessageID)

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.httpURL, bytes.NewReader(body))
	if err != nil {
		a.log.Error("agent http request creation failed", "error", err, "message_id", payload.MessageID)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Error("agent http delivery failed", "error", err, "message_id", payload.MessageID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		a.log.Info("agent http delivered", "status", resp.StatusCode, "message_id", payload.MessageID)
	} else {
		a.log.Warn("agent http non-2xx response", "status", resp.StatusCode, "message_id", payload.MessageID)
	}
}

// expandTemplate replaces {var} placeholders in the command template.
// Values are shell-escaped to prevent injection.
func (a *AgentTrigger) expandTemplate(tmpl string, p *WebhookPayload) string {
	isGroup := "false"
	if p.ChatType == "group" {
		isGroup = "true"
	}

	replacements := map[string]string{
		"{from}":          shellEscape(p.From),
		"{name}":          shellEscape(p.Name),
		"{message}":       shellEscape(p.Message),
		"{chat_jid}":      shellEscape(p.From),
		"{type}":          shellEscape(p.Type),
		"{is_group}":      isGroup,
		"{group_name}":    shellEscape(p.GroupName),
		"{message_id}":    shellEscape(p.MessageID),
		"{system_prompt}": shellEscape(a.systemPrompt),
	}

	result := tmpl
	for k, v := range replacements {
		result = strings.ReplaceAll(result, k, v)
	}
	return result
}

// shellEscape escapes a string for safe use in a shell command by replacing
// single quotes with the standard escape sequence.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\"'\"'")
}

// sendTyping sends a composing (typing) indicator to the given chat.
func (a *AgentTrigger) sendTyping(client *Client, chatJID string) {
	wc := client.GetClient()
	if wc == nil {
		return
	}

	jid, err := types.ParseJID(chatJID)
	if err != nil {
		a.log.Debug("agent typing: could not parse JID", "jid", chatJID, "error", err)
		return
	}

	if err := wc.SendChatPresence(context.Background(), jid, "composing", ""); err != nil {
		a.log.Debug("agent typing indicator failed", "error", err, "chat", chatJID)
	}
}

// clearTyping sends a paused indicator to clear the typing state.
func (a *AgentTrigger) clearTyping(client *Client, chatJID string) {
	wc := client.GetClient()
	if wc == nil {
		return
	}

	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return
	}

	if err := wc.SendChatPresence(context.Background(), jid, "paused", ""); err != nil {
		a.log.Debug("agent clear typing failed", "error", err, "chat", chatJID)
	}
}
