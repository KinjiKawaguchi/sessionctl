//go:build docker

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
	"github.com/KinjiKawaguchi/sessionctl/internal/transport"
)

func TestDockerChain_CloseReturnsToParent(t *testing.T) {
	// Connect to Docker fake-switch on port 2222
	tr, err := transport.DialSSH(transport.SSHConfig{
		Host: "127.0.0.1", Port: 2222,
		Username: "admin", Password: "admin123",
	})
	if err != nil {
		t.Skipf("Docker fake-switch not running: %v", err)
	}
	defer tr.Close()

	eng := expect.NewEngine(tr.Stdout, tr.Stdin, 8192)
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}

	// Wait for switch prompt
	eng.Expect(ctx, prompt)
	t.Logf("on switch")

	// Telnet to ASA
	eng.Send(ctx, "telnet 10.1.31.251\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Username:"})
	eng.Send(ctx, "admin\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Password:"})
	eng.Send(ctx, "admin123\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"})
	t.Logf("on ASA")

	// Exit ASA
	eng.Send(ctx, "exit\n")

	// Wait for switch prompt
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("return to switch: %v", err)
	}
	t.Logf("back on switch: %s", string(output))

	// Verify command works
	eng.Buffer().Clear()
	eng.Send(ctx, "show version\n")
	verOut, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("show version: %v", err)
	}
	if !strings.Contains(string(verOut), "C3560CX") {
		t.Fatalf("output: %q", string(verOut))
	}
	t.Logf("command works after unchaining")
}
