package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"ametie/internal/client"
	"ametie/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.NewConfigManager()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set hostname if not set
	if cfg.Get().Hostname == "" {
		hostname, _ := os.Hostname()
		cfg.SetHostname(hostname)
		cfg.Save()
	}

	// Create service
	service, err := client.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start service in goroutine
	go func() {
		if err := service.Start(); err != nil {
			log.Fatalf("Service error: %v", err)
		}
	}()

	// Wait for signal
	<-sigChan
	log.Println("Shutting down...")
	service.Stop()
}

