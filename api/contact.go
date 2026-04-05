package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/kypchumba/kypchumba.github.io/contact"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	allowedOrigin := strings.TrimSpace(os.Getenv("ALLOWED_ORIGIN"))
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}

	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg, err := contact.LoadConfig()
	if err != nil {
		log.Printf("config error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"error": "Server configuration error.",
		}); encodeErr != nil {
			log.Printf("write config error response failed: %v", encodeErr)
		}
		return
	}

	contact.Handle(cfg).ServeHTTP(w, r)
}
