//go:build docker

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	mcptools "github.com/KinjiKawaguchi/sessionctl/internal/mcp"
)

// TestSCCS_Cat3560ToASA simulates the SCCS topology:
// Claude Code → SSH to Cat3560 (Docker) → telnet to ASA (Docker)
func TestSCCS_Cat3560ToASA(t *testing.T) {
	deps := mcptools.NewDeps()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: SSH to fake Cat3560 (docker-compose exposes port 2222)
	_, openOut, err := mcptools.HandleSessionOpen(ctx, nil, mcptools.SessionOpenInput{
		Name:     "cat3560",
		Host:     "127.0.0.1",
		Port:     2222,
		Protocol: "ssh",
		Username: "admin",
		Password: "admin123",
		Profile:  "cisco_ios",
	}, deps)
	if err != nil {
		t.Skipf("Docker fake-switch not running on :2222: %v", err)
	}
	t.Logf("session_open: %s", openOut.Output)

	// Step 2: Verify we're on the switch
	_, execOut, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "cat3560",
		Command:   "show version",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("show version on switch: %v", err)
	}
	if !strings.Contains(execOut.Output, "C3560CX") {
		t.Fatalf("not on switch: %s", execOut.Output)
	}
	t.Logf("on switch: confirmed C3560CX")

	// Step 3: Chain to ASA via telnet
	_, chainOut, err := mcptools.HandleSessionChain(ctx, nil, mcptools.SessionChainInput{
		SessionID: "cat3560",
		Command:   "telnet 10.1.31.251",
		Name:      "asa",
		Profile:   "cisco_ios",
	}, deps)
	if err != nil {
		t.Fatalf("chain to ASA: %v", err)
	}
	t.Logf("chain output: %s", chainOut.Output)

	// Step 4: Login to ASA
	_, interOut, err := mcptools.HandleSessionInteract(ctx, nil, mcptools.SessionInteractInput{
		SessionID: "asa",
		Input:     "admin",
		Expect:    "(?i)password",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("ASA username: %v", err)
	}
	t.Logf("after username: %s", interOut.Output)

	_, interOut2, err := mcptools.HandleSessionInteract(ctx, nil, mcptools.SessionInteractInput{
		SessionID: "asa",
		Input:     "admin123",
		Expect:    "ciscoasa>",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("ASA password: %v", err)
	}
	t.Logf("after password: %s", interOut2.Output)

	// Step 5: Execute command on ASA
	_, asaExec, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "asa",
		Command:   "show version",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("show version on ASA: %v", err)
	}
	if !strings.Contains(asaExec.Output, "Adaptive Security") {
		t.Fatalf("not on ASA: %s", asaExec.Output)
	}
	t.Logf("on ASA: confirmed Adaptive Security Appliance")

	// Step 6: Close ASA (should return to switch)
	_, _, err = mcptools.HandleSessionClose(ctx, nil, mcptools.SessionCloseInput{
		SessionID: "asa",
	}, deps)
	if err != nil {
		t.Fatalf("close ASA: %v", err)
	}

	// Step 7: Verify we're back on the switch
	_, switchExec, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "cat3560",
		Command:   "show version",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("show version after unchaining: %v", err)
	}
	if !strings.Contains(switchExec.Output, "C3560CX") {
		t.Fatalf("not back on switch after close: %s", switchExec.Output)
	}
	t.Logf("back on switch: confirmed C3560CX")

	// Step 8: Close switch
	mcptools.HandleSessionClose(ctx, nil, mcptools.SessionCloseInput{SessionID: "cat3560"}, deps)

	// Step 9: Verify all sessions closed
	_, listOut, _ := mcptools.HandleSessionList(ctx, nil, mcptools.SessionListInput{}, deps)
	if len(listOut.Sessions) != 0 {
		t.Fatalf("sessions still open: %d", len(listOut.Sessions))
	}
	t.Logf("all sessions closed")
}
