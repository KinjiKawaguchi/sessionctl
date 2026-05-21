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

func startTelnetServer(t *testing.T) *testhelper.TelnetServer {
	t.Helper()
	srv, err := testhelper.NewTelnetServer(testhelper.ASAConfig())
	if err != nil {
		t.Fatalf("start telnet server: %v", err)
	}
	t.Cleanup(func() { srv.Close() })
	return srv
}

func connectTelnet(t *testing.T, addr string) (transport.Transport, *expect.Engine) {
	t.Helper()
	host, port := splitHostPort(t, addr)

	tr, err := transport.DialTelnet(transport.TelnetConfig{
		Host: host,
		Port: port,
	})
	if err != nil {
		t.Fatalf("dial telnet: %v", err)
	}
	t.Cleanup(func() { tr.Close() })

	eng := expect.NewEngine(tr.Stdout, tr.Stdin, 8192)
	t.Cleanup(func() { eng.Close() })

	return tr, eng
}

func TestTelnet_LoginAndPrompt(t *testing.T) {
	srv := startTelnetServer(t)
	_, eng := connectTelnet(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Expect username prompt
	userPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Username:"}
	_, err := eng.Expect(ctx, userPrompt)
	if err != nil {
		t.Fatalf("expect username: %v", err)
	}

	// Send username
	eng.Send(ctx, "admin\n")

	// Expect password prompt
	passPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Password:"}
	_, err = eng.Expect(ctx, passPrompt)
	if err != nil {
		t.Fatalf("expect password: %v", err)
	}

	// Send password
	eng.Send(ctx, "admin123\n")

	// Expect device prompt
	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"}
	_, err = eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("expect prompt: %v", err)
	}
}

func TestTelnet_CommandExecution(t *testing.T) {
	srv := startTelnetServer(t)
	_, eng := connectTelnet(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Login
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Username:"})
	eng.Send(ctx, "admin\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Password:"})
	eng.Send(ctx, "admin123\n")

	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"}
	eng.Expect(ctx, prompt)

	// Send command
	eng.Send(ctx, "show version\n")
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("expect after command: %v", err)
	}
	if !strings.Contains(string(output), "Adaptive Security") {
		t.Fatalf("output %q missing ASA version", string(output))
	}
}

func TestTelnet_ConnectionClose(t *testing.T) {
	srv := startTelnetServer(t)
	_, eng := connectTelnet(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Login
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Username:"})
	eng.Send(ctx, "admin\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Password:"})
	eng.Send(ctx, "admin123\n")

	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"}
	eng.Expect(ctx, prompt)

	// Send exit
	eng.Send(ctx, "exit\n")

	// Should receive close message
	closePattern := expect.Pattern{Kind: expect.PatternExact, Text: "Connection closed."}
	_, err := eng.Expect(ctx, closePattern)
	if err != nil {
		t.Fatalf("expect close: %v", err)
	}
}
