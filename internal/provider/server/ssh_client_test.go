package server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// testServer holds a test SSH server instance.
type testServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
}

// newTestServer creates and starts a test SSH server, returning port and PEM-encoded private key.
func newTestServer(t *testing.T) (int, []byte) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	block, err := ssh.MarshalPrivateKey(priv, "test")
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(block)

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil // accept any key for testing
		},
	}
	serverConfig.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	srv := &testServer{listener: listener, config: serverConfig}
	go srv.serve()

	t.Cleanup(func() { listener.Close() })
	return port, keyPEM
}

func (s *testServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *testServer) handleConn(conn net.Conn) {
	defer conn.Close()

	serverConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer serverConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown")
			continue
		}
		channel, reqs, err := newChan.Accept()
		if err != nil {
			continue
		}

		go s.handleSession(channel, reqs)
	}
}

func (s *testServer) handleSession(channel ssh.Channel, reqs <-chan *ssh.Request) {
	defer channel.Close()

	for req := range reqs {
		ok := false
		switch req.Type {
		case "exec":
			cmd := string(req.Payload[4:]) // skip length prefix
			cmd = strings.TrimSpace(cmd)

			req.Reply(true, nil) // reply first, before any I/O

			switch cmd {
			case "echo hello":
				channel.Write([]byte("hello\n"))
			case "whoami":
				channel.Write([]byte("testuser\n"))
			case "uname -s":
				channel.Write([]byte("Linux\n"))
			case "exit 1":
				channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{1}))
				return
			default:
				channel.Write([]byte("unknown command\n"))
			}

			// Signal successful exit
			channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
			return // close channel to send EOF
		case "shell":
			ok = true
		}
		req.Reply(ok, nil)
	}
}

// connectTestClient creates an SSH client connected to the test server.
func connectTestClient(t *testing.T, port int, keyPEM []byte) *Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := Connect(ctx, Config{
		Host:     "127.0.0.1",
		Port:     port,
		Username: "testuser",
		KeyBytes: keyPEM,
		Timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	return client
}

func TestConnect(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond) // wait for server to be ready

	client := connectTestClient(t, port, keyPEM)
	client.Close()
}

func TestConnectCancelled(t *testing.T) {
	port, _ := newTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Connect(ctx, Config{
		Host:     "127.0.0.1",
		Port:     port,
		Username: "testuser",
		Timeout:  3 * time.Second,
	})
	if err == nil {
		t.Error("Connect() should fail when context is cancelled")
	}
}

func TestRunCommand(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond)

	client := connectTestClient(t, port, keyPEM)
	defer client.Close()

	ctx := context.Background()
	output, err := client.RunCommand(ctx, "echo hello")
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("output = %q, want contain %q", output, "hello")
	}
}

func TestRunCommandWhoami(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond)

	client := connectTestClient(t, port, keyPEM)
	defer client.Close()

	ctx := context.Background()
	output, err := client.RunCommand(ctx, "whoami")
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}
	if !strings.Contains(output, "testuser") {
		t.Errorf("output = %q, want contain %q", output, "testuser")
	}
}

func TestRunCommandError(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond)

	client := connectTestClient(t, port, keyPEM)
	defer client.Close()

	ctx := context.Background()
	_, err := client.RunCommand(ctx, "exit 1")
	if err == nil {
		t.Error("RunCommand() should return error for exit 1")
	}
}

func TestRunCommandWithStdio(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond)

	client := connectTestClient(t, port, keyPEM)
	defer client.Close()

	ctx := context.Background()
	var buf bytes.Buffer
	err := client.RunCommandWithStdio(ctx, "echo hello", &buf, nil)
	if err != nil {
		t.Fatalf("RunCommandWithStdio() error = %v", err)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("stdout = %q, want contain %q", buf.String(), "hello")
	}
}

func TestRunCommandSplit(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond)

	client := connectTestClient(t, port, keyPEM)
	defer client.Close()

	ctx := context.Background()
	stdout, stderr, err := client.RunCommandSplit(ctx, "echo hello")
	if err != nil {
		t.Fatalf("RunCommandSplit() error = %v", err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("stdout = %q, want contain %q", stdout, "hello")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestConnectNoAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Connect(ctx, Config{
		Host:    "127.0.0.1",
		Port:    9999,
		Timeout: 1 * time.Second,
	})
	if err == nil {
		t.Error("Connect() should fail with no auth method")
	}
}

func TestClose(t *testing.T) {
	port, keyPEM := newTestServer(t)
	time.Sleep(100 * time.Millisecond)

	client := connectTestClient(t, port, keyPEM)

	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	// Double close should not panic
	client.Close()
}
