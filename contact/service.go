package contact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	maxNameLength       = 120
	maxEmailLength      = 254
	maxMessageLength    = 5000
	minNameLetters      = 3
	minEmailLength      = 5
	minMessageLength    = 5
	minSubmitTimeMS     = 1500
	minInteractionCount = 2
	rateLimitThreshold  = 5
	rateLimitWindow     = 10 * time.Minute
	securityLogFilePath = "form/text/logs.txt"
)

var (
	submissionLimiter = newRateLimiter(rateLimitWindow, rateLimitThreshold)
	securityLog       = newSecurityLogger(securityLogFilePath)
)

type Request struct {
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	Message          string   `json:"message"`
	Company          string   `json:"company"`
	TimeToSubmitMS   int64    `json:"timeToSubmitMs"`
	InteractionCount int      `json:"interactionCount"`
	InteractionTypes []string `json:"interactionTypes"`
}

type Config struct {
	GmailAddress     string
	GmailAppPassword string
	RecipientEmail   string
	AllowedOrigin    string
}

type rateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	entries map[string]rateLimitEntry
}

type rateLimitEntry struct {
	Count       int
	WindowStart time.Time
}

type securityLogger struct {
	mu   sync.Mutex
	path string
}

func LoadConfig() (Config, error) {
	gmailAddress := strings.TrimSpace(os.Getenv("GMAIL_ADDRESS"))
	if gmailAddress == "" {
		return Config{}, errors.New("missing GMAIL_ADDRESS environment variable")
	}

	gmailAppPassword := strings.TrimSpace(os.Getenv("GMAIL_APP_PASSWORD"))
	if gmailAppPassword == "" {
		return Config{}, errors.New("missing GMAIL_APP_PASSWORD environment variable")
	}

	recipientEmail := strings.TrimSpace(os.Getenv("CONTACT_RECIPIENT_EMAIL"))
	if recipientEmail == "" {
		recipientEmail = gmailAddress
	}

	allowedOrigin := strings.TrimSpace(os.Getenv("ALLOWED_ORIGIN"))
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}

	if err := securityLog.ensureReady(); err != nil {
		log.Printf("security log setup warning: %v", err)
	}

	return Config{
		GmailAddress:     gmailAddress,
		GmailAppPassword: gmailAppPassword,
		RecipientEmail:   recipientEmail,
		AllowedOrigin:    allowedOrigin,
	}, nil
}

func Handle(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applyCORS(w, cfg.AllowedOrigin)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "Method not allowed.",
			})
			return
		}

		clientIP := clientIPFromRequest(r)
		now := time.Now().UTC()

		allowed, retryAfter := submissionLimiter.Allow(clientIP, now)
		if !allowed {
			securityLog.Log("rate_limit_rejected", map[string]any{
				"ip":          clientIP,
				"retry_after": retryAfter.String(),
				"path":        r.URL.Path,
				"user_agent":  r.UserAgent(),
			})

			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "Too many submissions. Please wait a few minutes and try again.",
			})
			return
		}

		defer r.Body.Close()

		var payload Request
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			securityLog.Log("invalid_json", map[string]any{
				"ip":         clientIP,
				"path":       r.URL.Path,
				"user_agent": r.UserAgent(),
			})
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid JSON payload.",
			})
			return
		}

		payload.InteractionTypes = normalizeInteractionTypes(payload.InteractionTypes)

		if strings.TrimSpace(payload.Company) != "" {
			securityLog.Log("honeypot_rejected", map[string]any{
				"ip":                clientIP,
				"time_to_submit_ms": payload.TimeToSubmitMS,
				"interaction_count": payload.InteractionCount,
				"interaction_types": payload.InteractionTypes,
				"user_agent":        r.UserAgent(),
			})
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Unable to verify your submission.",
			})
			return
		}

		if payload.TimeToSubmitMS < minSubmitTimeMS {
			securityLog.Log("speed_rejected", map[string]any{
				"ip":                clientIP,
				"time_to_submit_ms": payload.TimeToSubmitMS,
				"interaction_count": payload.InteractionCount,
				"interaction_types": payload.InteractionTypes,
				"user_agent":        r.UserAgent(),
			})
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Submission was too fast to verify. Please try again.",
			})
			return
		}

		if payload.InteractionCount < minInteractionCount {
			securityLog.Log("interaction_rejected", map[string]any{
				"ip":                clientIP,
				"time_to_submit_ms": payload.TimeToSubmitMS,
				"interaction_count": payload.InteractionCount,
				"interaction_types": payload.InteractionTypes,
				"user_agent":        r.UserAgent(),
			})
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "We could not verify enough interaction on the form. Please try again.",
			})
			return
		}

		normalizedPayload, err := validateAndSanitize(payload)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		if err := SendEmail(ctx, cfg, normalizedPayload); err != nil {
			log.Printf("send email failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Unable to send your message right now.",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Message sent successfully.",
		})
	}
}

func SendEmail(ctx context.Context, cfg Config, payload Request) error {
	auth := smtp.PlainAuth("", cfg.GmailAddress, cfg.GmailAppPassword, "smtp.gmail.com")
	timestamp := time.Now().Format(time.RFC1123)
	subject := fmt.Sprintf("Portfolio Contact Form: %s", payload.Name)
	body := strings.Join([]string{
		fmt.Sprintf("Sender name: %s", payload.Name),
		fmt.Sprintf("Sender email: %s", payload.Email),
		fmt.Sprintf("Submitted in: %d ms", payload.TimeToSubmitMS),
		fmt.Sprintf("Interaction count: %d", payload.InteractionCount),
		fmt.Sprintf("Interaction types: %s", strings.Join(payload.InteractionTypes, ", ")),
		fmt.Sprintf("Timestamp: %s", timestamp),
		"",
		"Message:",
		payload.Message,
	}, "\r\n")

	message := strings.Join([]string{
		fmt.Sprintf("From: %s", cfg.GmailAddress),
		fmt.Sprintf("To: %s", cfg.RecipientEmail),
		fmt.Sprintf("Reply-To: %s", payload.Email),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		fmt.Sprintf("Subject: %s", subject),
		"",
		body,
	}, "\r\n")

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(
			"smtp.gmail.com:587",
			auth,
			cfg.GmailAddress,
			[]string{cfg.RecipientEmail},
			[]byte(message),
		)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func validateAndSanitize(payload Request) (Request, error) {
	name := sanitizeSingleLine(payload.Name)
	email := sanitizeSingleLine(payload.Email)
	message := sanitizeMessage(payload.Message)

	if name == "" || email == "" || message == "" {
		return Request{}, errors.New("Name, email, and message are required.")
	}

	if len([]rune(name)) > maxNameLength {
		return Request{}, fmt.Errorf("Name must be %d characters or fewer.", maxNameLength)
	}

	if countLetters(name) < minNameLetters {
		return Request{}, errors.New("Enter real name.")
	}

	if len([]rune(email)) < minEmailLength {
		return Request{}, errors.New("Enter real email.")
	}

	if len([]rune(email)) > maxEmailLength {
		return Request{}, fmt.Errorf("Email must be %d characters or fewer.", maxEmailLength)
	}

	if len([]rune(message)) < minMessageLength {
		return Request{}, errors.New("Type something.")
	}

	if len([]rune(message)) > maxMessageLength {
		return Request{}, fmt.Errorf("Message must be %d characters or fewer.", maxMessageLength)
	}

	parsedAddress, err := mail.ParseAddress(email)
	if err != nil || strings.TrimSpace(parsedAddress.Address) == "" {
		return Request{}, errors.New("Please provide a valid email address.")
	}

	return Request{
		Name:             html.EscapeString(name),
		Email:            parsedAddress.Address,
		Message:          html.EscapeString(message),
		TimeToSubmitMS:   payload.TimeToSubmitMS,
		InteractionCount: payload.InteractionCount,
		InteractionTypes: payload.InteractionTypes,
	}, nil
}

func countLetters(value string) int {
	count := 0
	for _, r := range value {
		if unicode.IsLetter(r) {
			count++
		}
	}

	return count
}

func sanitizeSingleLine(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}

func sanitizeMessage(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeInteractionTypes(types []string) []string {
	seen := make(map[string]struct{}, len(types))
	normalized := make([]string, 0, len(types))

	for _, interactionType := range types {
		cleaned := sanitizeSingleLine(strings.ToLower(interactionType))
		if cleaned == "" {
			continue
		}

		if _, exists := seen[cleaned]; exists {
			continue
		}

		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}

	return normalized
}

func clientIPFromRequest(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}

		if header == "X-Forwarded-For" {
			parts := strings.Split(value, ",")
			value = strings.TrimSpace(parts[0])
		}

		return value
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return "unknown"
}

func newRateLimiter(window time.Duration, limit int) *rateLimiter {
	return &rateLimiter{
		window:  window,
		limit:   limit,
		entries: make(map[string]rateLimitEntry),
	}
}

func (rl *rateLimiter) Allow(key string, now time.Time) (bool, time.Duration) {
	if strings.TrimSpace(key) == "" {
		key = "unknown"
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for entryKey, entry := range rl.entries {
		if now.Sub(entry.WindowStart) > rl.window {
			delete(rl.entries, entryKey)
		}
	}

	entry := rl.entries[key]
	if entry.WindowStart.IsZero() || now.Sub(entry.WindowStart) > rl.window {
		entry = rateLimitEntry{
			Count:       0,
			WindowStart: now,
		}
	}

	entry.Count++
	rl.entries[key] = entry

	if entry.Count > rl.limit {
		retryAfter := entry.WindowStart.Add(rl.window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}

	return true, 0
}

func newSecurityLogger(path string) *securityLogger {
	return &securityLogger{path: path}
}

func (logger *securityLogger) ensureReady() error {
	if logger == nil {
		return nil
	}

	return os.MkdirAll(filepath.Dir(logger.path), 0o755)
}

func (logger *securityLogger) Log(event string, fields map[string]any) {
	if logger == nil {
		return
	}

	logger.mu.Lock()
	defer logger.mu.Unlock()

	if err := logger.ensureReady(); err != nil {
		log.Printf("security log setup failed: %v", err)
		return
	}

	payload := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"event":     event,
	}

	for key, value := range fields {
		payload[key] = value
	}

	line, err := json.Marshal(payload)
	if err != nil {
		log.Printf("security log marshal failed: %v", err)
		return
	}

	file, err := os.OpenFile(logger.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("security log open failed: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(string(line) + "\n"); err != nil {
		log.Printf("security log write failed: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json failed: %v", err)
	}
}

func applyCORS(w http.ResponseWriter, allowedOrigin string) {
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
