package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/provider/server"
	"golang.org/x/crypto/ssh"
)

func (b *Bridge) getRemoteExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	host := toString(row["host"])
	port := toInt(row["port"])

	// Look up credential if associated
	var password, keyStr string
	if credID := toString(row["credential_id"]); credID != "" {
		credRow := make(map[string]interface{})
		if err := b.DB.Table("credentials").Where("id = ?", credID).Take(&credRow).Error; err == nil {
			encrypted := toString(credRow["encrypted_value"])
			credType := toString(credRow["type"])
			if b.EncryptionKey != nil && encrypted != "" {
				if decrypted, err := crypto.Decrypt(b.EncryptionKey, encrypted); err == nil {
					// Route decrypted value based on credential type
					// Type "ssh_key" -> KeyBytes (SSH private key)
					// Type "ssh" (password) or others -> Password field
					if credType == "ssh_key" {
						keyStr = decrypted
					} else {
						password = decrypted
					}
				}
			}
		}
	}

	// SSH username: prefer server record field, fall back to env var
	username := toString(row["username"])
	if username == "" {
		username = os.Getenv("DEPLOYPILOT_SSH_DEFAULT_USER")
	}
	if username == "" {
		return nil, fmt.Errorf("SSH username not configured for server %s (configure DEPLOYPILOT_SSH_DEFAULT_USER or set server username)", serverID)
	}
	cfg := server.Config{
		Host:               host,
		Port:               port,
		Username:           username,
		Password:           password,
		KeyBytes:           []byte(keyStr),
		Timeout:            30 * time.Second,
		KnownHostsPath:     b.SSHKnownHostsPath,
		StrictHostChecking: b.SSHStrictHostKeyChecking,
	}

	client, err := server.Connect(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed to %s:%d: %w. "+
			"Suggestions: check host/port, verify security group allows TCP/%d, "+
			"confirm sshd is running, and ensure credentials are correct",
			host, port, err, port)
	}

	return &sshClientExecutor{Client: client}, nil
}

// RemoteExecutor is the interface for remote command execution (used by WebSocket terminal).
type RemoteExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
	CreateInteractiveSession(ctx context.Context, termType string, rows, cols int) (InteractiveSession, error)
	Close() error
}

// InteractiveSession represents a persistent interactive SSH session with PTY.
type InteractiveSession interface {
	// StdinPipe returns a writer connected to the session's stdin.
	StdinPipe() io.Writer
	// SetWindowSize resizes the PTY.
	SetWindowSize(rows, cols int) error
	// Output returns a channel that receives stdout+stderr output.
	Output() <-chan []byte
	// Done returns a channel that is closed when the session exits.
	Done() <-chan struct{}
	// Close terminates the session.
	Close() error
}


// sshClientExecutor wraps server.Client to implement deployer.CommandExecutor.
type sshClientExecutor struct {
	Client *server.Client
}

func (e *sshClientExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	return e.Client.RunCommand(ctx, cmd)
}

func (e *sshClientExecutor) CreateInteractiveSession(ctx context.Context, termType string, rows, cols int) (InteractiveSession, error) {
	session, err := e.Client.CreateSession(ctx, true, termType, rows, cols)
	if err != nil {
		return nil, err
	}
	return newInteractiveSession(session), nil
}

func (e *sshClientExecutor) Close() error {
	return e.Client.Close()
}

// interactiveSession wraps an ssh.Session to implement InteractiveSession.
type interactiveSession struct {
	session *ssh.Session
	stdin   io.WriteCloser
	output  chan []byte
	done    chan struct{}
}

func newInteractiveSession(session *ssh.Session) *interactiveSession {
	is := &interactiveSession{
		session: session,
		output:  make(chan []byte, 256),
		done:    make(chan struct{}),
	}
	// Get stdin pipe
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		close(is.done)
		return is
	}
	is.stdin = stdinPipe

	// Pipe stdout and stderr to output channel
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		close(is.done)
		return is
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		close(is.done)
		return is
	}

	// Start shell
	if err := session.Shell(); err != nil {
		close(is.done)
		return is
	}

	// Read stdout in goroutine
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				is.output <- data
			}
			if err != nil {
				break
			}
		}
	}()

	// Read stderr in goroutine
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				is.output <- data
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for session to exit
	go func() {
		_ = session.Wait()
		close(is.done)
	}()

	return is
}

func (is *interactiveSession) StdinPipe() io.Writer {
	return is.stdin
}

func (is *interactiveSession) SetWindowSize(rows, cols int) error {
	return is.session.WindowChange(rows, cols)
}

func (is *interactiveSession) Output() <-chan []byte {
	return is.output
}

func (is *interactiveSession) Done() <-chan struct{} {
	return is.done
}

func (is *interactiveSession) Close() error {
	_ = is.stdin.Close()
	return is.session.Close()
}


// ---------- 3. ListApps ----------


// GetAppDeploymentHistory returns the deployment history for an app by app ID.


// ---------- 16. CreateCredential ----------


// ---------- 38. BatchDNS ----------


// generateID returns a unique deployment record ID.
