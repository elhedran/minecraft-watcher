package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Config struct {
	Host       string
	Port       string
	Secret     string
	TLSEnabled bool
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Host:       getEnv("MINECRAFT_MGMT_HOST", "localhost"),
		Port:       getEnv("MINECRAFT_MGMT_PORT", "25566"),
		Secret:     os.Getenv("MINECRAFT_MGMT_SECRET"),
		TLSEnabled: getEnvBool("MINECRAFT_MGMT_TLS_ENABLED", true),
	}

	if cfg.Secret == "" {
		return nil, fmt.Errorf("MINECRAFT_MGMT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func main() {
	log.Println("minecraft-watcher starting...")

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Configuration loaded: host=%s, port=%s, tls=%v", cfg.Host, cfg.Port, cfg.TLSEnabled)

	conn := connectWithRetry(cfg)
	defer conn.Close()

	log.Println("minecraft-watcher ready - connected to server")

	// Keep running
	select {}
}

func connectWithRetry(cfg *Config) *websocket.Conn {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	attempt := 1

	for {
		log.Printf("Attempting connection to Minecraft server (attempt %d)...", attempt)

		conn, err := connectToServer(cfg)
		if err != nil {
			log.Printf("Connection failed: %v", err)
			log.Printf("Retrying in %v...", backoff)
			time.Sleep(backoff)

			// Exponential backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			attempt++
			continue
		}

		log.Println("Successfully connected to Minecraft server")
		return conn
	}
}

func connectToServer(cfg *Config) (*websocket.Conn, error) {
	scheme := "ws"
	if cfg.TLSEnabled {
		scheme = "wss"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Path:   "/",
	}

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.Secret))

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Skip TLS verification for now (should be configurable in production)
	if cfg.TLSEnabled {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	conn, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
