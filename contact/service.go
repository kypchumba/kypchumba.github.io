package contact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Request struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

type Config struct {
	GmailAddress     string
	GmailAppPassword string
	RecipientEmail   string
	AllowedOrigin    string
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

		defer r.Body.Close()

		var payload Request
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid JSON payload.",
			})
			return
		}

		payload.Name = strings.TrimSpace(payload.Name)
		payload.Email = strings.TrimSpace(payload.Email)
		payload.Message = strings.TrimSpace(payload.Message)

		if payload.Name == "" || payload.Email == "" || payload.Message == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Name, email, and message are required.",
			})
			return
		}

		if !strings.Contains(payload.Email, "@") {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Please provide a valid email address.",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		if err := SendEmail(ctx, cfg, payload); err != nil {
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
