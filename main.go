package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github-stars-notify/internal/config"
	"github-stars-notify/internal/service"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create the service
	svc, err := service.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the service in a goroutine
	go func() {
		log.Println("Starting GitHub Stars Notify service...")
		if err := svc.Start(ctx); err != nil {
			log.Fatalf("Service failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutting down...")

	// Cancel context to signal shutdown
	cancel()

	// Stop the service
	svc.Stop()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	log.Println("Service stopped")
}
