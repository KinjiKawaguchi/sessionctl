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

func TestMCP_SessionExec_NoStaleOutput(t *testing.T) {
	// Start fake switch
	srv, err := testhelper.NewSSHServer(testhelper.CiscoSwitchConfig())
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() { srv.Close() })

	host, port := splitHostPort(t, srv.Addr())
	deps := mcptools.NewDeps()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Open session
	_, _, err = mcptools.HandleSessionOpen(ctx, nil, mcptools.SessionOpenInput{
		Name:     "sw1",
		Host:     host,
		Port:     port,
		Username: "admin",
		Password: "admin123",
		Profile:  "cisco_ios",
	}, deps)
	if err != nil {
		t.Fatalf("session_open: %v", err)
	}

	// Execute show version — should return actual output, not empty
	_, execOut, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "sw1",
		Command:   "show version",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("session_exec: %v", err)
	}

	if !strings.Contains(execOut.Output, "C3560CX") {
		t.Fatalf("session_exec returned %q, expected output containing 'C3560CX'", execOut.Output)
	}

	// Execute a second command — should also return actual output
	_, execOut2, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "sw1",
		Command:   "show interfaces status",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("session_exec 2: %v", err)
	}

	if !strings.Contains(execOut2.Output, "Gi0/1") {
		t.Fatalf("session_exec 2 returned %q, expected output containing 'Gi0/1'", execOut2.Output)
	}
}
