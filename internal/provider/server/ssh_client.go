package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client wraps an SSH connection to a remote server.
type Client struct {
	client *ssh.Client
}

// Config holds SSH connection parameters.
type Config struct {
	Host              string
	Port              int
	Username          string
	Password          string // optional, use KeyBytes instead
	KeyBytes          []byte // SSH private key bytes
	Timeout           time.Duration
	KnownHostsPath    string // path to known_hosts file
	StrictHostChecking bool  // strict mode for host key verification
}

// hostKeyCallback returns an appropriate HostKeyCallback based on the configuration.
// If knownHostsPath is empty, it falls back to InsecureIgnoreHostKey with a warning.
func hostKeyCallback(knownHostsPath string, strict bool) (ssh.HostKeyCallback, error) {
	if knownHostsPath == "" {
		// No known_hosts file configured — use insecure mode with warning
		slog.Warn("no SSH known_hosts file configured, host key verification is disabled")
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Ensure known_hosts file exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			return nil, fmt.Errorf("failed to create known_hosts directory: %w", err)
		}
		if err := os.WriteFile(knownHostsPath, nil, 0600); err != nil {
			return nil, fmt.Errorf("failed to create known_hosts file: %w", err)
		}
	}

	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts: %w", err)
	}

	if strict {
		return callback, nil
	}

	// Non-strict mode: warn on unknown hosts but allow the connection
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			// New host — append to known_hosts
			slog.Warn("new SSH host, adding to known_hosts", "host", hostname, "key_type", key.Type())
			f, fErr := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0600)
			if fErr != nil {
				return fmt.Errorf("failed to open known_hosts for writing: %w", fErr)
			}
			defer func() {
				if cerr := f.Close(); cerr != nil {
					slog.Warn("failed to close known_hosts file", "error", cerr)
				}
			}()
			line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
			if _, fErr = fmt.Fprintln(f, line); fErr != nil {
				return fmt.Errorf("failed to write to known_hosts: %w", fErr)
			}
			return nil
		}
		return err // Host key mismatch — reject
	}, nil
}

// Connect establishes an SSH connection to the server.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	cb, err := hostKeyCallback(cfg.KnownHostsPath, cfg.StrictHostChecking)
	if err != nil {
		return nil, fmt.Errorf("failed to set up host key callback: %w", err)
	}

	var authMethods []ssh.AuthMethod

	// Prefer key-based auth
	if len(cfg.KeyBytes) > 0 {
		signer, err := ssh.ParsePrivateKey(cfg.KeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Fall back to password auth
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method provided (need key or password)")
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: cb,
		Timeout:         cfg.Timeout,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))

	// Use context for cancellation
	type result struct {
		client *ssh.Client
		err    error
	}
	done := make(chan result, 1)
	go func() {
		client, err := ssh.Dial("tcp", addr, sshConfig)
		done <- result{client, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("SSH connection cancelled: %w", ctx.Err())
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("SSH dial failed (%s@%s): %w", cfg.Username, addr, r.err)
		}
		return &Client{client: r.client}, nil
	}
}

// RunCommand executes a command on the remote server and returns its output.
func (c *Client) RunCommand(ctx context.Context, cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer func() {
		if cerr := session.Close(); cerr != nil {
			slog.Warn("failed to close SSH session after RunCommand", "error", cerr)
		}
	}()

	// Use context for cancellation
	type result struct {
		output string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		out, err := session.CombinedOutput(cmd)
		done <- result{string(out), err}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return "", fmt.Errorf("command cancelled: %w", ctx.Err())
	case r := <-done:
		if r.err != nil {
			return r.output, fmt.Errorf("command failed: %w", r.err)
		}
		return r.output, nil
	}
}

// RunCommandWithStdio executes a command and streams stdout/stderr to the writer.
func (c *Client) RunCommandWithStdio(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer func() {
		if cerr := session.Close(); cerr != nil {
			slog.Warn("failed to close SSH session after RunCommandWithStdio", "error", cerr)
		}
	}()

	if stdout != nil {
		session.Stdout = stdout
	}
	if stderr != nil {
		session.Stderr = stderr
	}

	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		done <- result{session.Run(cmd)}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return fmt.Errorf("command cancelled: %w", ctx.Err())
	case r := <-done:
		return r.err
	}
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// CreateSession creates a new SSH session with optional PTY support.
// If pty is true, a pseudo-terminal is requested with the given termType, rows, and cols.
func (c *Client) CreateSession(ctx context.Context, pty bool, termType string, rows, cols int) (*ssh.Session, error) {
	if c.client == nil {
		return nil, fmt.Errorf("SSH client is nil")
	}
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	if pty {
		if termType == "" {
			termType = "xterm"
		}
		if rows == 0 {
			rows = 24
		}
		if cols == 0 {
			cols = 80
		}
		if err := session.RequestPty(termType, rows, cols, ssh.TerminalModes{}); err != nil {
			_ = session.Close()
			return nil, fmt.Errorf("failed to request PTY: %w", err)
		}
	}
	return session, nil
}

// IsConnected returns true if the underlying SSH connection is alive.
func (c *Client) IsConnected() bool {
	if c.client == nil {
		return false
	}
	_, _, err := c.client.SendRequest("keepalive@golang.org", true, nil)
	return err == nil
}

// RunCommandSplit executes a command and returns stdout and stderr separately.
func (c *Client) RunCommandSplit(ctx context.Context, cmd string) (stdout, stderr string, err error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer func() {
		if cerr := session.Close(); cerr != nil {
			slog.Warn("failed to close SSH session after RunCommandSplit", "error", cerr)
		}
	}()

	var outBuf, errBuf strings.Builder
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		done <- result{session.Run(cmd)}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return "", "", fmt.Errorf("command cancelled: %w", ctx.Err())
	case r := <-done:
		return outBuf.String(), errBuf.String(), r.err
	}
}
