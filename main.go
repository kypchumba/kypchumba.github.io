package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kypchumba/kypchumba.github.io/contact"
)

type appConfig struct {
	Port          string
	ContactConfig contact.Config
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/contact", contact.Handle(cfg.ContactConfig))
	mux.Handle("/", http.FileServer(http.Dir(".")))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("server listening on http://localhost:%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadConfig() (appConfig, error) {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	contactConfig, err := contact.LoadConfig()
	if err != nil {
		return appConfig{}, err
	}

	return appConfig{
		Port:          port,
		ContactConfig: contactConfig,
	}, nil
}
