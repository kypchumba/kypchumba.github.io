package handler

import (
	"log"
	"net/http"

	"github.com/kypchumba/kypchumba.github.io/internal/contact"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	cfg, err := contact.LoadConfig()
	if err != nil {
		log.Printf("config error: %v", err)
		http.Error(w, "Server configuration error.", http.StatusInternalServerError)
		return
	}

	contact.Handle(cfg).ServeHTTP(w, r)
}
