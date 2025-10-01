package test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestWatcherConnection(t *testing.T) {
	// Check required environment variables
	if os.Getenv("MINECRAFT_MGMT_SECRET") == "" {
		t.Skip("MINECRAFT_MGMT_SECRET not set, skipping system test")
	}

	// Build the watcher if not already built
	t.Log("Building watcher...")
	buildCmd := exec.Command("go", "build", "-o", "../minecraft-watcher", "../cmd/minecraft-watcher")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build watcher: %v\n%s", err, output)
	}

	// Set up environment for test mode
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "../minecraft-watcher")
	cmd.Env = append(os.Environ(),
		"TEST_MODE=true",
		"POLL_INTERVAL_SECONDS=5",
		"MIN_UPTIME_MINUTES=0",
		"IDLE_TIMEOUT_MINUTES=0",
	)

	// Capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}

	t.Log("Starting watcher in test mode...")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Collect output
	var foundTestMode, foundConnection, foundMonitoring bool
	scanner := bufio.NewScanner(stderr)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			t.Logf("watcher: %s", line)

			if strings.Contains(line, "TEST MODE") {
				foundTestMode = true
			}
			if strings.Contains(line, "Successfully connected") {
				foundConnection = true
			}
			if strings.Contains(line, "Players online") || strings.Contains(line, "No players online") {
				foundMonitoring = true
			}
		}
	}()

	// Also capture stdout (shouldn't have much)
	go func() {
		scannerOut := bufio.NewScanner(stdout)
		for scannerOut.Scan() {
			t.Logf("watcher stdout: %s", scannerOut.Text())
		}
	}()

	// Wait for process or timeout
	<-ctx.Done()

	// Give it a moment to flush logs
	time.Sleep(100 * time.Millisecond)

	// Verify expectations
	if !foundTestMode {
		t.Error("✗ Watcher did not indicate TEST MODE")
	} else {
		t.Log("✓ Watcher running in TEST MODE")
	}

	if !foundConnection {
		t.Error("✗ Watcher did not successfully connect to Minecraft server")
	} else {
		t.Log("✓ Watcher successfully connected to Minecraft server")
	}

	if !foundMonitoring {
		t.Error("✗ Watcher did not query player information")
	} else {
		t.Log("✓ Watcher successfully monitoring players")
	}

	// Clean up
	if err := cmd.Process.Kill(); err != nil {
		t.Logf("Warning: failed to kill process: %v", err)
	}
	cmd.Wait()
}
