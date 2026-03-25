package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/rs/zerolog/log"
)

// NotificationService defines the interface for sending notifications.
// This abstraction allows swapping Discord for Slack, email, etc. in the future.
type NotificationService interface {
	// SendContactNotification formats and sends a notification about a new
	// contact form submission. It is non-blocking — the actual HTTP call runs
	// in a background goroutine.
	SendContactNotification(ctx context.Context, senderName, senderEmail, messageBody string)

	// SendSponsorNotification formats and sends a notification about a new
	// sponsorship payment. It is non-blocking.
	SendSponsorNotification(ctx context.Context, sponsorName, email, currency string, amount float64)

	// SendRaw sends an arbitrary string message to the notification channel.
	// It is non-blocking.
	SendRaw(ctx context.Context, message string)

	// Shutdown waits for all in-flight notification goroutines to finish.
	// Call this during graceful application shutdown.
	Shutdown()
}

// discordPayload is the JSON body Discord's webhook API expects.
type discordPayload struct {
	Content string `json:"content"`
}

// DiscordNotificationService sends notifications via a Discord webhook URL.
// All sends are dispatched asynchronously in goroutines so the calling code
// is never blocked by network I/O.
type DiscordNotificationService struct {
	webhookURL string
	client     *http.Client
	wg         sync.WaitGroup
	random     *rand.Rand
	sendMu     sync.Mutex
}

type rateLimitResponse struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

// NewDiscordNotificationService creates a new DiscordNotificationService.
// If the webhook URL is empty, all send operations become silent no-ops and
// a warning is logged at construction time.
func NewDiscordNotificationService(cfg config.DiscordConfig) *DiscordNotificationService {
	if cfg.WebhookURL == "" {
		log.Warn().Msg("discord: webhook URL is not configured — notifications will be silently skipped")
	} else {
		log.Info().Msg("discord: notification service initialized")
	}

	return &DiscordNotificationService{
		webhookURL: cfg.WebhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SendContactNotification formats a rich markdown message about a new contact
// form submission and sends it to Discord asynchronously.
func (d *DiscordNotificationService) SendContactNotification(ctx context.Context, senderName, senderEmail, messageBody string) {
	msg := fmt.Sprintf(
		"📬 **New Portfolio Contact Message!**\n"+
			"> **Name:** %s\n"+
			"> **Email:** %s\n"+
			"> **Message:**\n"+
			"```text\n%s\n```",
		senderName,
		senderEmail,
		messageBody,
	)

	d.sendAsync(ctx, msg)
}

// SendSponsorNotification formats a rich markdown message about a new
// sponsorship payment and sends it to Discord asynchronously.
func (d *DiscordNotificationService) SendSponsorNotification(ctx context.Context, sponsorName, email, currency string, amount float64) {
	msg := fmt.Sprintf(
		"🎉 **NEW SPONSOR ALERT!** 🎉\n"+
			"> **Name:** %s\n"+
			"> **Amount:** %.2f %s\n"+
			"> **Email:** %s\n"+
			"The outbox pipeline processed this payment successfully!",
		sponsorName,
		amount,
		currency,
		email,
	)

	d.sendAsync(ctx, msg)
}

// SendRaw sends an arbitrary string message to Discord asynchronously.
func (d *DiscordNotificationService) SendRaw(ctx context.Context, message string) {
	d.sendAsync(ctx, message)
}

// Shutdown blocks until every in-flight notification goroutine has completed.
// This prevents the process from exiting before all Discord webhook calls
// have finished.
func (d *DiscordNotificationService) Shutdown() {
	log.Info().Msg("discord: waiting for in-flight notifications to finish")
	d.wg.Wait()
	log.Info().Msg("discord: all notifications drained — shutdown complete")
}

// sendAsync dispatches the actual HTTP POST in a separate goroutine so the
// caller is never blocked. The goroutine is tracked via the WaitGroup so
// Shutdown can wait for it.
func (d *DiscordNotificationService) sendAsync(ctx context.Context, message string) {
	if d.webhookURL == "" {
		log.Debug().Msg("discord: skipping notification — webhook URL not configured")
		return
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("discord: recovered from panic in notification goroutine")
			}
		}()

		// Use a detached context for the send operation so that retries can
		// continue even if the original request context is cancelled.
		detachedCtx := context.WithoutCancel(ctx)
		if err := d.send(detachedCtx, message); err != nil {
			log.Error().Err(err).Msg("discord: failed to send notification")
		}
	}()
}

// send performs the synchronous HTTP POST to the Discord webhook endpoint.
// It returns an error if the request fails or Discord responds with a
// non-2xx status code. It handles HTTP 429 (Too Many Requests) by retrying
// with exponential backoff and jitter, respecting the Retry-After header.
func (d *DiscordNotificationService) send(ctx context.Context, message string) error {
	payload := discordPayload{Content: message}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: failed to marshal payload: %w", err)
	}

	// Retry configuration
	const maxRetries = 5
	initialBackoff := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		d.sendMu.Lock()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
		if err != nil {
			d.sendMu.Unlock()
			return fmt.Errorf("discord: failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := d.client.Do(req)
		if err != nil {
			d.sendMu.Unlock()
			return fmt.Errorf("discord: request failed: %w", err)
		}

		// Success!
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			d.sendMu.Unlock()
			log.Info().Msg("discord: notification sent successfully")
			return nil
		}

		// Handle Rate Limiting (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			waitTime, err := parseDiscordRetryAfter(resp)
			if err != nil {
				resp.Body.Close()
				d.sendMu.Unlock()
				return err
			}
			if waitTime <= 0 {
				waitTime = initialBackoff * time.Duration(1<<attempt)
			}
			resp.Body.Close()
			d.sendMu.Unlock()

			// Add jitter (±20%) to prevent synchronized retries.
			jitter := 0.8 + 0.4*d.random.Float64()
			finalWait := time.Duration(float64(waitTime) * jitter)

			// Minimum wait to be safe.
			if finalWait < 500*time.Millisecond {
				finalWait = 500 * time.Millisecond
			}

			log.Warn().
				Int("status", resp.StatusCode).
				Int("attempt", attempt+1).
				Dur("retry_after", finalWait).
				Msg("discord: rate limited — retrying")

			select {
			case <-time.After(finalWait):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// For other errors, log and potentially retry if it's a 5xx.
		status := resp.StatusCode
		resp.Body.Close()
		d.sendMu.Unlock()

		if status >= 500 && attempt < maxRetries {
			time.Sleep(initialBackoff * time.Duration(1<<attempt))
			continue
		}

		return fmt.Errorf("discord: unexpected status code %d", status)
	}

	return fmt.Errorf("discord: failed after %d attempts", maxRetries+1)
}

func parseDiscordRetryAfter(resp *http.Response) (time.Duration, error) {
	for _, header := range []string{"Retry-After", "X-RateLimit-Reset-After"} {
		if value := resp.Header.Get(header); value != "" {
			if wait, ok := parseRetryAfterSeconds(value); ok {
				return wait, nil
			}
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("discord: failed to read rate limit body: %w", err)
	}
	if len(body) == 0 {
		return 0, nil
	}

	var payload rateLimitResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, nil
	}
	if payload.RetryAfter <= 0 {
		return 0, nil
	}

	return time.Duration(payload.RetryAfter * float64(time.Second)), nil
}

func parseRetryAfterSeconds(value string) (time.Duration, bool) {
	var seconds float64
	if _, err := fmt.Sscanf(value, "%f", &seconds); err != nil {
		return 0, false
	}

	return time.Duration(seconds * float64(time.Second)), true
}
