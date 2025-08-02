package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/muratdemir0/gopulse-messages/internal/config"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	configPath := filepath.Join(".config", fmt.Sprintf("%s.yaml", env))

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("Config loaded for env: %s\n", env)
	fmt.Printf("App Name: %s\n", cfg.App.Name)
	fmt.Printf("App Port: %d\n", cfg.App.Port)
	fmt.Printf("Webhook Host: %s\n", cfg.Webhook.Host)
	fmt.Printf("Webhook Path: %s\n", cfg.Webhook.Path)
}
