package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type Config struct {
	Host                string
	Port                string
	Secret              string
	TLSEnabled          bool
	TestMode            bool
	IdleTimeoutMinutes  int
	MinUptimeMinutes    int
	PollIntervalSeconds int
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      int         `json:"id"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type PlayersResult struct {
	Players []Player `json:"players"`
}

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Host:                getEnv("MINECRAFT_MGMT_HOST", "localhost"),
		Port:                getEnv("MINECRAFT_MGMT_PORT", "25566"),
		Secret:              os.Getenv("MINECRAFT_MGMT_SECRET"),
		TLSEnabled:          getEnvBool("MINECRAFT_MGMT_TLS_ENABLED", true),
		TestMode:            getEnvBool("TEST_MODE", false),
		IdleTimeoutMinutes:  getEnvInt("IDLE_TIMEOUT_MINUTES", 10),
		MinUptimeMinutes:    getEnvInt("MIN_UPTIME_MINUTES", 30),
		PollIntervalSeconds: getEnvInt("POLL_INTERVAL_SECONDS", 30),
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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
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

	if cfg.TestMode {
		log.Println("*** RUNNING IN TEST MODE - will not actually shut down server ***")
	}

	log.Printf("Configuration: host=%s, port=%s, tls=%v, idle_timeout=%dm, min_uptime=%dm, poll_interval=%ds",
		cfg.Host, cfg.Port, cfg.TLSEnabled, cfg.IdleTimeoutMinutes, cfg.MinUptimeMinutes, cfg.PollIntervalSeconds)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	conn := connectWithRetry(cfg)
	defer func() {
		log.Println("Closing connection to Minecraft server...")
		conn.Close()
		log.Println("Shutdown complete")
	}()

	log.Println("minecraft-watcher ready - connected to server")

	monitorPlayers(ctx, conn, cfg)
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

var requestID int64

func sendJSONRPC(conn *websocket.Conn, method string, params interface{}) (*JSONRPCResponse, error) {
	id := int(atomic.AddInt64(&requestID, 1))

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      id,
		Params:  params,
	}

	if err := conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp JSONRPCResponse
	if err := conn.ReadJSON(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s (data: %s)",
			resp.Error.Code, resp.Error.Message, resp.Error.Data)
	}

	return &resp, nil
}

func getPlayers(conn *websocket.Conn) ([]Player, error) {
	resp, err := sendJSONRPC(conn, "minecraft:players", nil)
	if err != nil {
		return nil, err
	}

	var players []Player
	if err := json.Unmarshal(resp.Result, &players); err != nil {
		return nil, fmt.Errorf("failed to parse players result: %w", err)
	}

	return players, nil
}

func shutdownServer(conn *websocket.Conn, testMode bool) error {
	if testMode {
		log.Println("TEST MODE: Would execute server shutdown now")
		return nil
	}

	log.Println("Sending shutdown command to server...")
	_, err := sendJSONRPC(conn, "minecraft:server/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("Server shutdown command sent successfully")
	return nil
}

func monitorPlayers(ctx context.Context, conn *websocket.Conn, cfg *Config) {
	startTime := time.Now()
	lastPlayerTime := time.Now()
	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Println("Starting player monitoring loop...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitoring stopped")
			return
		case <-ticker.C:
			players, err := getPlayers(conn)
			if err != nil {
				log.Printf("Error getting players: %v", err)
				continue
			}

			playerCount := len(players)
			if playerCount > 0 {
				lastPlayerTime = time.Now()
				playerNames := make([]string, len(players))
				for i, p := range players {
					playerNames[i] = p.Name
				}
				log.Printf("Players online (%d): %v", playerCount, playerNames)
			} else {
				timeSinceLastPlayer := time.Since(lastPlayerTime)
				log.Printf("No players online (idle for %v)", timeSinceLastPlayer.Round(time.Second))
			}

			// Check shutdown conditions
			uptime := time.Since(startTime)
			timeSinceLastPlayer := time.Since(lastPlayerTime)

			uptimeMinutes := int(uptime.Minutes())
			idleMinutes := int(timeSinceLastPlayer.Minutes())

			log.Printf("Status: uptime=%dm, idle=%dm (thresholds: min_uptime=%dm, idle_timeout=%dm)",
				uptimeMinutes, idleMinutes, cfg.MinUptimeMinutes, cfg.IdleTimeoutMinutes)

			if uptimeMinutes >= cfg.MinUptimeMinutes && idleMinutes >= cfg.IdleTimeoutMinutes {
				log.Printf("Shutdown conditions met: uptime=%dm >= %dm AND idle=%dm >= %dm",
					uptimeMinutes, cfg.MinUptimeMinutes, idleMinutes, cfg.IdleTimeoutMinutes)

				if err := shutdownServer(conn, cfg.TestMode); err != nil {
					log.Printf("Error shutting down server: %v", err)
					continue
				}

				if !cfg.TestMode {
					log.Println("Server shutdown initiated. Exiting.")
					os.Exit(0)
				}
			}
		}
	}
}
