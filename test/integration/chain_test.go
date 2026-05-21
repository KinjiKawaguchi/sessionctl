//go:build integration

package integration

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
	"github.com/KinjiKawaguchi/sessionctl/test/testhelper"
)

func setupChainEnv(t *testing.T) (*testhelper.SSHServer, *testhelper.TelnetServer) {
	t.Helper()

	// Start fake ASA (telnet)
	asaSrv, err := testhelper.NewTelnetServer(testhelper.ASAConfig())
	if err != nil {
		t.Fatalf("start ASA: %v", err)
	}
	t.Cleanup(func() { asaSrv.Close() })

	// Start fake switch (SSH) with telnet target
	switchSrv, err := testhelper.NewSSHServer(testhelper.CiscoSwitchConfig())
	if err != nil {
		t.Fatalf("start switch: %v", err)
	}
	// Map "10.1.31.251" to actual ASA address
	switchSrv.TelnetTargets["10.1.31.251"] = asaSrv.Addr()
	t.Cleanup(func() { switchSrv.Close() })

	return switchSrv, asaSrv
}

func TestChain_SSHToTelnet(t *testing.T) {
	switchSrv, _ := setupChainEnv(t)
	_, eng := connectSSH(t, switchSrv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switchPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}

	// Wait for switch prompt
	_, err := eng.Expect(ctx, switchPrompt)
	if err != nil {
		t.Fatalf("switch prompt: %v", err)
	}

	// Telnet to ASA
	eng.Send(ctx, "telnet 10.1.31.251\n")

	// Expect ASA login
	userPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Username:"}
	_, err = eng.Expect(ctx, userPrompt)
	if err != nil {
		t.Fatalf("ASA username prompt: %v", err)
	}

	eng.Send(ctx, "admin\n")

	passPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Password:"}
	_, err = eng.Expect(ctx, passPrompt)
	if err != nil {
		t.Fatalf("ASA password prompt: %v", err)
	}

	eng.Send(ctx, "admin123\n")

	asaPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"}
	_, err = eng.Expect(ctx, asaPrompt)
	if err != nil {
		t.Fatalf("ASA prompt: %v", err)
	}
}

func TestChain_CommandInChild(t *testing.T) {
	switchSrv, _ := setupChainEnv(t)
	_, eng := connectSSH(t, switchSrv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect through to ASA
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"})
	eng.Send(ctx, "telnet 10.1.31.251\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Username:"})
	eng.Send(ctx, "admin\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Password:"})
	eng.Send(ctx, "admin123\n")

	asaPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"}
	eng.Expect(ctx, asaPrompt)

	// Execute command on ASA
	eng.Send(ctx, "show version\n")
	output, err := eng.Expect(ctx, asaPrompt)
	if err != nil {
		t.Fatalf("command on ASA: %v", err)
	}
	if !strings.Contains(string(output), "Adaptive Security") {
		t.Fatalf("output %q missing ASA version", string(output))
	}
}

func TestChain_ChildCloseReturnsToParent(t *testing.T) {
	switchSrv, _ := setupChainEnv(t)
	_, eng := connectSSH(t, switchSrv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect through to ASA
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"})
	eng.Send(ctx, "telnet 10.1.31.251\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Username:"})
	eng.Send(ctx, "admin\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Password:"})
	eng.Send(ctx, "admin123\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "ciscoasa>"})

	// Exit ASA
	eng.Send(ctx, "exit\n")

	// Should return to Switch prompt
	switchPrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"}
	_, err := eng.Expect(ctx, switchPrompt)
	if err != nil {
		t.Fatalf("return to switch: %v", err)
	}
}

func TestChain_EnableMode(t *testing.T) {
	switchSrv, _ := setupChainEnv(t)
	_, eng := connectSSH(t, switchSrv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Switch>"})

	// Enable
	eng.Send(ctx, "enable\n")
	eng.Expect(ctx, expect.Pattern{Kind: expect.PatternExact, Text: "Password:"})
	eng.Send(ctx, "enable123\n")

	enablePrompt := expect.Pattern{Kind: expect.PatternExact, Text: "Switch#"}
	_, err := eng.Expect(ctx, enablePrompt)
	if err != nil {
		t.Fatalf("enable mode: %v", err)
	}

	// Verify command works in enable mode
	eng.Send(ctx, "show running-config\n")
	output, err := eng.Expect(ctx, enablePrompt)
	if err != nil {
		t.Fatalf("command in enable mode: %v", err)
	}
	if !strings.Contains(string(output), "hostname Switch") {
		t.Fatalf("output %q missing config", string(output))
	}
}

func TestChain_YamahaRTX(t *testing.T) {
	srv, err := testhelper.NewSSHServer(testhelper.YamahaRTXConfig())
	if err != nil {
		t.Fatalf("start RTX: %v", err)
	}
	t.Cleanup(func() { srv.Close() })

	_, eng := connectSSH(t, srv.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prompt := expect.Pattern{
		Kind:  expect.PatternRegex,
		Regex: regexp.MustCompile(`RTX1210[#>]`),
	}

	_, err = eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("RTX prompt: %v", err)
	}

	eng.Send(ctx, "show status\n")
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("RTX command: %v", err)
	}
	if !strings.Contains(string(output), "Rev.14") {
		t.Fatalf("output %q missing status info", string(output))
	}
}
