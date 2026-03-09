package site

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAvailablePort(t *testing.T) {
	port, err := FindAvailablePort(1313)
	if err != nil {
		t.Fatalf("FindAvailablePort: %v", err)
	}
	if port < 1313 || port >= 65535 {
		t.Errorf("port %d out of range", port)
	}
}

func TestFindAvailablePortSkipsBusy(t *testing.T) {
	// Get two ports — they should be different or the same (both free)
	port1, err := FindAvailablePort(1313)
	if err != nil {
		t.Fatalf("first FindAvailablePort: %v", err)
	}
	port2, err := FindAvailablePort(1313)
	if err != nil {
		t.Fatalf("second FindAvailablePort: %v", err)
	}
	// Both should be valid
	if port1 < 1313 || port2 < 1313 {
		t.Errorf("ports out of range: %d, %d", port1, port2)
	}
}

func TestServerStateReadWrite(t *testing.T) {
	dir := t.TempDir()

	// No PID file — should return nil
	state := ReadServerState(dir)
	if state != nil {
		t.Error("expected nil state for empty dir")
	}

	// Write state
	if err := writeServerState(dir, &ServerState{PID: 12345, Port: 1313}); err != nil {
		t.Fatalf("writeServerState: %v", err)
	}

	// Read it back
	state = ReadServerState(dir)
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.PID != 12345 {
		t.Errorf("PID = %d, want 12345", state.PID)
	}
	if state.Port != 1313 {
		t.Errorf("Port = %d, want 1313", state.Port)
	}

	// Remove and verify
	removeServerState(dir)
	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Error("PID file should be removed")
	}
}

func TestServerStateIsRunning(t *testing.T) {
	// nil state — not running
	var s *ServerState
	if s.IsRunning() {
		t.Error("nil state should not be running")
	}

	// Current process PID — should be running
	s = &ServerState{PID: os.Getpid(), Port: 1313}
	if !s.IsRunning() {
		t.Error("current process should be running")
	}

	// Bogus PID — should not be running
	s = &ServerState{PID: 999999999, Port: 1313}
	if s.IsRunning() {
		t.Error("bogus PID should not be running")
	}
}

func TestServerStateMalformed(t *testing.T) {
	dir := t.TempDir()

	// Write garbage
	if err := os.WriteFile(filepath.Join(dir, pidFileName), []byte("garbage"), 0o600); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	state := ReadServerState(dir)
	if state != nil {
		t.Error("malformed PID file should return nil")
	}

	// Write only one line
	if err := os.WriteFile(filepath.Join(dir, pidFileName), []byte("12345\n"), 0o600); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	state = ReadServerState(dir)
	if state != nil {
		t.Error("incomplete PID file should return nil")
	}
}
