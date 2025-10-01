package test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type Config struct {
	Host       string
	Port       string
	Secret     string
	TLSEnabled bool
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

func TestSystemConnection(t *testing.T) {
	t.Log("Loading configuration...")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Configuration error: %v", err)
	}

	t.Logf("Connecting to Minecraft server at %s://%s:%s",
		map[bool]string{true: "wss", false: "ws"}[cfg.TLSEnabled],
		cfg.Host, cfg.Port)

	conn, err := connectToServer(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	t.Log("✓ Successfully connected to Minecraft server")
}

func TestPlayerQuery(t *testing.T) {
	t.Log("Loading configuration...")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Configuration error: %v", err)
	}

	t.Log("Connecting to server...")
	conn, err := connectToServer(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	t.Log("Querying player list...")
	players, err := getPlayers(conn)
	if err != nil {
		t.Fatalf("Failed to query players: %v", err)
	}

	t.Logf("✓ Successfully queried player list")
	t.Logf("✓ Player count: %d", len(players))

	if len(players) > 0 {
		t.Log("✓ Players online:")
		for _, player := range players {
			t.Logf("  - %s (UUID: %s)", player.Name, player.ID)
		}
	} else {
		t.Log("✓ No players currently online")
	}
}
