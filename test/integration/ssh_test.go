//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
	"github.com/KinjiKawaguchi/sessionctl/internal/transport"
	"github.com/KinjiKawaguchi/sessionctl/test/testhelper"
)

func startSSHServer(t *testing.T) *testhelper.SSHServer {
	t.Helper()
	srv, err := testhelper.NewSSHServer(testhelper.CiscoSwitchConfig())
	if err != nil {
		t.Fatalf("start SSH server: %v", err)
	}
	t.Cleanup(func() { srv.Close() })
	return srv
}

func connectSSH(t *testing.T, addr string) (transport.Transport, *expect.Engine) {
	t.Helper()
	host, port := splitHostPort(t, addr)

	tr, err := transport.DialSSH(transport.SSHConfig{
		Host:     host,
		Port:     port,
		Username: "admin",
		Password: "admin123",
	})
	if err != nil {
		t.Fatalf("dial SSH: %v", err)
	}
	t.Cleanup(func() { tr.Close() })

	eng := expect.NewEngine(tr.Stdout, tr.Stdin, 8192)
	t.Cleanup(func() { eng.Close() })

	return tr, eng
}

func TestSSH_PasswordAuth(t *testing.T) {
	srv := startSSHServer(t)
	_, eng := connectSSH(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should receive banner + prompt
	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("expect prompt: %v", err)
	}
	if !strings.Contains(string(output), "Cisco IOS") {
		t.Fatalf("output %q missing banner", string(output))
	}
}

func TestSSH_CommandExecution(t *testing.T) {
	srv := startSSHServer(t)
	_, eng := connectSSH(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait for initial prompt
	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}
	eng.Expect(ctx, prompt)

	// Send command
	eng.Send(ctx, "show version\n")

	// Expect output + next prompt
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("expect after command: %v", err)
	}
	if !strings.Contains(string(output), "C3560CX") {
		t.Fatalf("output %q missing version info", string(output))
	}
}

func TestSSH_SessionPersistence(t *testing.T) {
	srv := startSSHServer(t)
	_, eng := connectSSH(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}
	eng.Expect(ctx, prompt)

	// First command
	eng.Send(ctx, "show version\n")
	eng.Expect(ctx, prompt)

	// Second command on same session
	eng.Send(ctx, "show interfaces status\n")
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("second command: %v", err)
	}
	if !strings.Contains(string(output), "Gi0/1") {
		t.Fatalf("output %q missing interface info", string(output))
	}
}

func TestSSH_AuthFailure(t *testing.T) {
	srv := startSSHServer(t)
	host, port := splitHostPort(t, srv.Addr())

	_, err := transport.DialSSH(transport.SSHConfig{
		Host:     host,
		Port:     port,
		Username: "admin",
		Password: "wrongpass",
	})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSH_PromptDetection(t *testing.T) {
	srv := startSSHServer(t)
	_, eng := connectSSH(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait for prompt after banner
	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}
	_, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("initial prompt: %v", err)
	}

	// Send command, expect prompt returns after output
	eng.Send(ctx, "show version\n")
	_, err = eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("prompt after command: %v", err)
	}
}
