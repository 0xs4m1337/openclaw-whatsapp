package bridge

import (
	"context"
	"log/slog"
	"time"
)

// maxBackoff is the upper limit for exponential backoff between reconnect attempts.
const maxBackoff = 5 * time.Minute

// Reconnectable is implemented by the bridge client.
type Reconnectable interface {
	IsConnected() bool
	HasSession() bool // true if there's a stored WhatsApp session (not fresh/logged-out)
	Connect(ctx context.Context) error
}

// StartReconnectLoop runs a goroutine that checks connection every interval.
// If disconnected and has stored session, attempts reconnect with exponential backoff.
//
// The loop:
//  1. Ticker fires every interval.
//  2. If connected, reset backoff and continue.
//  3. If no stored session (fresh device or logged out), skip.
//  4. Attempt reconnect with a per-attempt timeout equal to the current backoff.
//  5. On failure, double the backoff (capped at 5 minutes).
//  6. On success, reset the backoff.
//  7. Stop when ctx is cancelled.
func StartReconnectLoop(ctx context.Context, client Reconnectable, interval time.Duration, log *slog.Logger) {
	go reconnectLoop(ctx, client, interval, log)
}

func reconnectLoop(ctx context.Context, client Reconnectable, interval time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	backoff := interval
	if backoff < time.Second {
		backoff = time.Second
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("reconnect loop stopped")
			return
		case <-ticker.C:
			if client.IsConnected() {
				// Connection is healthy; reset backoff.
				backoff = interval
				if backoff < time.Second {
					backoff = time.Second
				}
				continue
			}

			if !client.HasSession() {
				// No stored session — nothing to reconnect to.
				log.Debug("no stored session, skipping reconnect")
				continue
			}

			log.Info("connection lost, attempting reconnect", "backoff", backoff)

			// Create a child context with timeout for this attempt.
			attemptCtx, cancel := context.WithTimeout(ctx, backoff)
			err := client.Connect(attemptCtx)
			cancel()

			if err != nil {
				log.Warn("reconnect failed", "error", err, "next_backoff", backoff*2)
				// Double the backoff, capped at maxBackoff.
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			} else {
				log.Info("reconnected successfully")
				backoff = interval
				if backoff < time.Second {
					backoff = time.Second
				}
			}
		}
	}
}
