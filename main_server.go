package main

import (
	"log"
	"os"
	"path/filepath"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/server"

	"github.com/joho/godotenv"
)

func main_server() {
	// Get current working directory for debugging
	if cwd, err := os.Getwd(); err == nil {
		log.Printf("Current working directory: %s", cwd)
	}

	// Try to load .env file from multiple possible locations
	envPaths := []string{
		".env",
		"./.env",
	}

	envLoaded := false
	for _, envPath := range envPaths {
		log.Printf("Trying to load .env from: %s", envPath)
		if _, err := os.Stat(envPath); err == nil {
			log.Printf("Found .env file at: %s", envPath)
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("Successfully loaded environment variables from %s", envPath)
				envLoaded = true
				break
			} else {
				log.Printf("Failed to load .env from %s: %v", envPath, err)
			}
		} else {
			log.Printf(".env file not found at %s: %v", envPath, err)
		}
	}

	if !envLoaded {
		// Try to find .env in the executable's directory
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			envPath := filepath.Join(execDir, ".env")
			log.Printf("Trying executable directory: %s", envPath)
			if _, err := os.Stat(envPath); err == nil {
				if err := godotenv.Load(envPath); err == nil {
					log.Printf("Successfully loaded environment variables from %s", envPath)
					envLoaded = true
				}
			}
		}
	}

	if !envLoaded {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Create and start server
	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
