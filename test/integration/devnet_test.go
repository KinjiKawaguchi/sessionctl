//go:build devnet

package integration

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
	"github.com/KinjiKawaguchi/sessionctl/internal/transport"
)

func devnetConfig() (host string, port int, user, pass string) {
	host = env("DEVNET_HOST", "sandbox-iosxe-latest-1.cisco.com")
	user = env("DEVNET_USER", "developer")
	pass = env("DEVNET_PASS", "C1sco12345")
	return host, 22, user, pass
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func connectDevNet(t *testing.T) *expect.Engine {
	t.Helper()

	host, port, user, pass := devnetConfig()

	tr, err := transport.DialSSH(transport.SSHConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
	})
	if err != nil {
		t.Skipf("DevNet Sandbox unavailable (set DEVNET_HOST/DEVNET_USER/DEVNET_PASS to override): %v", err)
	}
	t.Cleanup(func() { tr.Close() })

	eng := expect.NewEngine(tr.Stdout, tr.Stdin, 16384)
	t.Cleanup(func() { eng.Close() })
	return eng
}

// iosPrompt matches real IOS XE prompts like "csr1000v#" or "Cat8000V>"
var iosPrompt = expect.Pattern{
	Kind:  expect.PatternRegex,
	Regex: regexp.MustCompile(`[a-zA-Z0-9_.-]+[#>]\s*$`),
}

func TestDevNet_SSHConnect(t *testing.T) {
	eng := connectDevNet(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	output, err := eng.Expect(ctx, iosPrompt)
	if err != nil {
		t.Fatalf("expect prompt: %v", err)
	}
	t.Logf("initial output:\n%s", string(output))
}

func TestDevNet_ShowVersion(t *testing.T) {
	eng := connectDevNet(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eng.Expect(ctx, iosPrompt)

	// Disable paging first
	eng.Send(ctx, "terminal length 0\n")
	eng.Expect(ctx, iosPrompt)

	eng.Send(ctx, "show version\n")
	output, err := eng.Expect(ctx, iosPrompt)
	if err != nil {
		t.Fatalf("expect after show version: %v", err)
	}

	text := string(output)
	t.Logf("show version output:\n%s", text)

	if !strings.Contains(text, "Cisco") {
		t.Fatalf("output missing 'Cisco': %s", text)
	}
}

func TestDevNet_MultipleCommands(t *testing.T) {
	eng := connectDevNet(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	eng.Expect(ctx, iosPrompt)
	eng.Send(ctx, "terminal length 0\n")
	eng.Expect(ctx, iosPrompt)

	commands := []struct {
		cmd      string
		contains string
	}{
		{"show ip interface brief", "Interface"},
		{"show running-config | include hostname", "hostname"},
	}

	for _, c := range commands {
		eng.Send(ctx, c.cmd+"\n")
		output, err := eng.Expect(ctx, iosPrompt)
		if err != nil {
			t.Fatalf("%s: expect prompt: %v", c.cmd, err)
		}
		text := string(output)
		t.Logf("%s output:\n%s", c.cmd, text)

		if !strings.Contains(text, c.contains) {
			t.Errorf("%s: output missing %q", c.cmd, c.contains)
		}
	}
}

func TestDevNet_ChainViaDocker(t *testing.T) {
	// SSH to Docker test-shell container, then chain to DevNet Sandbox.
	// Requires: docker run -d --name testshell -p 2222:22 testshell

	jumpHost := env("JUMP_HOST", "127.0.0.1")
	jumpPort := 2222
	jumpUser := env("JUMP_USER", "testuser")
	jumpPass := env("JUMP_PASS", "testpass")

	localTr, err := transport.DialSSH(transport.SSHConfig{
		Host:     jumpHost,
		Port:     jumpPort,
		Username: jumpUser,
		Password: jumpPass,
	})
	if err != nil {
		t.Skipf("Docker test-shell not available (run: docker run -d -p 2222:22 testshell): %v", err)
	}
	defer localTr.Close()

	eng := expect.NewEngine(localTr.Stdout, localTr.Stdin, 16384)
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for local shell prompt
	localPrompt := expect.Pattern{
		Kind:  expect.PatternRegex,
		Regex: regexp.MustCompile(`[$#%>]\s*$`),
	}
	eng.Expect(ctx, localPrompt)

	// Chain: SSH from localhost to DevNet
	devnetHost, _, devnetUser, devnetPasswd := devnetConfig()
	eng.Send(ctx, fmt.Sprintf("ssh %s@%s\n", devnetUser, devnetHost))

	// Handle host key confirmation if needed
	patterns := []expect.Pattern{
		{Kind: expect.PatternExact, Text: "yes/no"},
		{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)password:\s*$`)},
		iosPrompt,
	}

	output, idx, err := eng.ExpectAny(ctx, patterns)
	if err != nil {
		t.Fatalf("expect after ssh command: %v", err)
	}

	switch idx {
	case 0: // host key confirmation
		eng.Send(ctx, "yes\n")
		_, err = eng.Expect(ctx, patterns[1]) // now expect password
		if err != nil {
			t.Fatalf("expect password after yes: %v", err)
		}
		eng.Send(ctx, devnetPasswd+"\n")
	case 1: // password prompt
		eng.Send(ctx, devnetPasswd+"\n")
	case 2: // already at IOS prompt (key-based auth)
		t.Logf("connected without password prompt")
	}

	// Should now be at IOS prompt
	iosOutput, err := eng.Expect(ctx, iosPrompt)
	if err != nil {
		t.Fatalf("expect IOS prompt: %v", err)
	}

	t.Logf("chained to DevNet:\n%s", string(output))
	t.Logf("IOS prompt:\n%s", string(iosOutput))

	// Verify we can run commands on the real device
	eng.Send(ctx, "show version | include Cisco\n")
	verOutput, err := eng.Expect(ctx, iosPrompt)
	if err != nil {
		t.Fatalf("show version via chain: %v", err)
	}

	if !strings.Contains(string(verOutput), "Cisco") {
		t.Fatalf("chained output missing 'Cisco': %s", string(verOutput))
	}
	t.Logf("chained command output:\n%s", string(verOutput))
}
