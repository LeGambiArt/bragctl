package site

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/LeGambiArt/bragctl/internal/config"
	"github.com/LeGambiArt/bragctl/internal/ui"
)

const (
	pidFileName  = ".server.pid"
	lockFileName = ".server.lock"
)

// ServerState holds the state of a running server.
type ServerState struct {
	PID  int
	Port int
}

// pidFilePath returns the PID file path for a site.
func pidFilePath(sitePath string) string {
	return filepath.Join(sitePath, pidFileName)
}

// lockFilePath returns the lock file path for a site.
func lockFilePath(sitePath string) string {
	return filepath.Join(sitePath, lockFileName)
}

// acquireServerLock acquires an exclusive lock on the server lock file.
// Returns the lock file handle which must be released by calling releaseLock().
func acquireServerLock(sitePath string) (*os.File, error) {
	lockPath := lockFilePath(sitePath)
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // sitePath is validated
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lockFile.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, fmt.Errorf("another operation is already in progress (lock held)")
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	return lockFile, nil
}

// releaseLock releases the server lock file.
func releaseLock(lockFile *os.File) {
	if lockFile != nil {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		_ = lockFile.Close()
	}
}

// logDir returns the logs directory.
func logDir() string {
	return filepath.Join(config.BaseDir(), "logs")
}

// LogFilePath returns the log file path for a site.
func LogFilePath(siteName string) string {
	return filepath.Join(logDir(), siteName+".log")
}

// ReadServerState reads the PID file for a site. Returns nil if not running.
func ReadServerState(sitePath string) *ServerState {
	data, err := os.ReadFile(pidFilePath(sitePath)) //nolint:gosec // known config path
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil
	}

	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return nil
	}
	port, err := strconv.Atoi(lines[1])
	if err != nil {
		return nil
	}

	return &ServerState{PID: pid, Port: port}
}

// writeServerState writes the PID file for a site.
func writeServerState(sitePath string, state *ServerState) error {
	content := fmt.Sprintf("%d\n%d\n", state.PID, state.Port)
	return os.WriteFile(pidFilePath(sitePath), []byte(content), 0o600)
}

// removeServerState removes the PID file for a site.
func removeServerState(sitePath string) {
	_ = os.Remove(pidFilePath(sitePath))
}

// IsRunning checks if a server process is still alive.
func (s *ServerState) IsRunning() bool {
	if s == nil {
		return false
	}
	process, err := os.FindProcess(s.PID)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal
	return process.Signal(syscall.Signal(0)) == nil
}

// FindAvailablePort finds a free TCP port starting from startPort.
func FindAvailablePort(startPort int) (int, error) {
	for port := startPort; port < 65535; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available ports found starting from %d", startPort)
}

// StartBackground starts a server in the background.
// It re-execs the current binary with --foreground, redirecting output to a log file.
func StartBackground(siteName, sitePath string, opts ServeOpts) error {
	// Acquire lock to prevent TOCTOU race with concurrent StartBackground calls
	lockFile, err := acquireServerLock(sitePath)
	if err != nil {
		return err
	}
	defer releaseLock(lockFile)

	// Check if already running
	state := ReadServerState(sitePath)
	if state.IsRunning() {
		return fmt.Errorf("site %q already running on port %d (PID %d)", siteName, state.Port, state.PID)
	}

	// Clean up stale PID file
	removeServerState(sitePath)

	// Find available port if not specified
	port := opts.Port
	if port == 0 {
		var err error
		port, err = FindAvailablePort(1313)
		if err != nil {
			return err
		}
	}

	// Prepare log file
	if err := os.MkdirAll(logDir(), 0o750); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logPath := LogFilePath(siteName)
	logFile, err := os.Create(logPath) //nolint:gosec // known log path
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}

	// Find our own executable for re-exec
	self, err := os.Executable()
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("find executable: %w", err)
	}

	// Build command: bragctl serve <site> --foreground --port <port> --bind <bind>
	args := []string{
		"serve", siteName,
		"--foreground",
		"--port", strconv.Itoa(port),
		"--bind", opts.Bind,
	}

	cmd := exec.Command(self, args...) //nolint:gosec // re-exec ourselves
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start server: %w", err)
	}

	// Don't wait — let it run in the background
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
	}()

	// Save state
	if err := writeServerState(sitePath, &ServerState{PID: cmd.Process.Pid, Port: port}); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}

	bind := opts.Bind
	if bind == "" {
		bind = "127.0.0.1"
	}
	ui.Success("Server started for %q on http://%s:%d", siteName, bind, port)
	ui.Dim("  Logs: %s", logPath)

	return nil
}

// StopServer stops a running background server.
func StopServer(siteName, sitePath string) error {
	// Acquire lock to prevent race with concurrent operations
	lockFile, err := acquireServerLock(sitePath)
	if err != nil {
		return err
	}
	defer releaseLock(lockFile)

	state := ReadServerState(sitePath)
	if state == nil {
		return fmt.Errorf("site %q is not running", siteName)
	}

	if !state.IsRunning() {
		removeServerState(sitePath)
		return fmt.Errorf("site %q is not running (stale PID)", siteName)
	}

	process, err := os.FindProcess(state.PID)
	if err != nil {
		removeServerState(sitePath)
		return fmt.Errorf("find process %d: %w", state.PID, err)
	}

	// Graceful shutdown — ignore error (process may already be dead)
	_ = process.Signal(syscall.SIGTERM)

	removeServerState(sitePath)
	ui.Success("Server stopped for %q", siteName)

	return nil
}
