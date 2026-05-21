//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	mcptools "github.com/KinjiKawaguchi/sessionctl/internal/mcp"
	"github.com/KinjiKawaguchi/sessionctl/test/testhelper"
)

func TestSCCS_InProcess_Cat3560ToASA(t *testing.T) {
	// Start in-process fake devices
	asaSrv, err := testhelper.NewTelnetServer(testhelper.ASAConfig())
	if err != nil {
		t.Fatalf("start ASA: %v", err)
	}
	t.Cleanup(func() { asaSrv.Close() })

	switchSrv, err := testhelper.NewSSHServer(testhelper.CiscoSwitchConfig())
	if err != nil {
		t.Fatalf("start switch: %v", err)
	}
	switchSrv.TelnetTargets["10.1.31.251"] = asaSrv.Addr()
	t.Cleanup(func() { switchSrv.Close() })

	host, port := splitHostPort(t, switchSrv.Addr())
	deps := mcptools.NewDeps()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Open switch
	_, _, err = mcptools.HandleSessionOpen(ctx, nil, mcptools.SessionOpenInput{
		Name: "sw", Host: host, Port: port,
		Username: "admin", Password: "admin123", Profile: "cisco_ios",
	}, deps)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// 2. Verify switch
	_, out, _ := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "sw", Command: "show version", Timeout: 10,
	}, deps)
	t.Logf("switch show version: %.100s", out.Output)

	// 3. Chain to ASA
	mcptools.HandleSessionChain(ctx, nil, mcptools.SessionChainInput{
		SessionID: "sw", Command: "telnet 10.1.31.251", Name: "asa", Profile: "cisco_ios",
	}, deps)

	// 4. Login
	mcptools.HandleSessionInteract(ctx, nil, mcptools.SessionInteractInput{
		SessionID: "asa", Input: "admin", Expect: "(?i)password", Timeout: 5,
	}, deps)
	mcptools.HandleSessionInteract(ctx, nil, mcptools.SessionInteractInput{
		SessionID: "asa", Input: "admin123", Expect: "ciscoasa>", Timeout: 5,
	}, deps)

	// 5. Command on ASA
	_, asaOut, _ := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "asa", Command: "show version", Timeout: 10,
	}, deps)
	if !strings.Contains(asaOut.Output, "Adaptive Security") {
		t.Fatalf("ASA output: %q", asaOut.Output)
	}
	t.Logf("ASA confirmed")

	// 6. Close ASA
	_, _, err = mcptools.HandleSessionClose(ctx, nil, mcptools.SessionCloseInput{SessionID: "asa"}, deps)
	if err != nil {
		t.Fatalf("close asa: %v", err)
	}
	t.Logf("ASA closed")

	// 7. Back on switch
	_, swOut, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "sw", Command: "show version", Timeout: 10,
	}, deps)
	if err != nil {
		t.Fatalf("exec after close: %v", err)
	}
	if !strings.Contains(swOut.Output, "C3560CX") {
		t.Fatalf("not back on switch: %q", swOut.Output)
	}
	t.Logf("back on switch: confirmed")
}
