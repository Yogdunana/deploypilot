package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH connection to a remote server.
type Client struct {
	client *ssh.Client
}

// Config holds SSH connection parameters.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string // optional, use KeyBytes instead
	KeyBytes []byte // SSH private key bytes
	Timeout  time.Duration
}

// Connect establishes an SSH connection to the server.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: add known_hosts support in production
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
	defer session.Close()

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
		session.Close()
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
	defer session.Close()

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
		session.Close()
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
	defer session.Close()

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
		session.Close()
		return "", "", fmt.Errorf("command cancelled: %w", ctx.Err())
	case r := <-done:
		return outBuf.String(), errBuf.String(), r.err
	}
}
