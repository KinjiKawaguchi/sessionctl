//go:build devnet

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	mcptools "github.com/KinjiKawaguchi/sessionctl/internal/mcp"
)

func newDeps() *mcptools.Deps {
	return mcptools.NewDeps()
}

func TestMCP_SessionOpenExecClose(t *testing.T) {
	host, _, user, pass := devnetConfig()
	deps := newDeps()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// session_open
	_, openOut, err := mcptools.HandleSessionOpen(ctx, nil, mcptools.SessionOpenInput{
		Name:     "cisco1",
		Host:     host,
		Port:     22,
		Protocol: "ssh",
		Username: user,
		Password: pass,
		Profile:  "cisco_ios",
	}, deps)
	if err != nil {
		t.Skipf("DevNet unavailable: %v", err)
	}
	t.Logf("session_open output: %s", openOut.Output)

	if openOut.SessionID != "cisco1" {
		t.Fatalf("session ID = %q, want %q", openOut.SessionID, "cisco1")
	}

	// session_exec: terminal length 0
	_, _, err = mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "cisco1",
		Command:   "terminal length 0",
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("session_exec terminal length: %v", err)
	}

	// session_exec: show version
	_, execOut, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "cisco1",
		Command:   "show version",
		Timeout:   15,
	}, deps)
	if err != nil {
		t.Fatalf("session_exec show version: %v", err)
	}
	t.Logf("session_exec output (first 200 chars): %.200s", execOut.Output)

	if !strings.Contains(execOut.Output, "Cisco") {
		t.Errorf("output missing 'Cisco'")
	}

	// session_list
	_, listOut, err := mcptools.HandleSessionList(ctx, nil, mcptools.SessionListInput{}, deps)
	if err != nil {
		t.Fatalf("session_list: %v", err)
	}
	if len(listOut.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(listOut.Sessions))
	}
	if listOut.Sessions[0].ID != "cisco1" {
		t.Fatalf("listed session ID = %q, want %q", listOut.Sessions[0].ID, "cisco1")
	}

	// session_close
	_, closeOut, err := mcptools.HandleSessionClose(ctx, nil, mcptools.SessionCloseInput{
		SessionID: "cisco1",
	}, deps)
	if err != nil {
		t.Fatalf("session_close: %v", err)
	}
	if !closeOut.Closed {
		t.Fatal("session not closed")
	}

	// Verify session is gone
	_, listOut2, _ := mcptools.HandleSessionList(ctx, nil, mcptools.SessionListInput{}, deps)
	if len(listOut2.Sessions) != 0 {
		t.Fatalf("expected 0 sessions after close, got %d", len(listOut2.Sessions))
	}
}

func TestMCP_SessionInteract(t *testing.T) {
	host, _, user, pass := devnetConfig()
	deps := newDeps()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Open session
	_, _, err := mcptools.HandleSessionOpen(ctx, nil, mcptools.SessionOpenInput{
		Name:     "cisco2",
		Host:     host,
		Port:     22,
		Username: user,
		Password: pass,
		Profile:  "cisco_ios",
	}, deps)
	if err != nil {
		t.Skipf("DevNet unavailable: %v", err)
	}

	// session_interact: send command and expect a pattern
	_, interactOut, err := mcptools.HandleSessionInteract(ctx, nil, mcptools.SessionInteractInput{
		SessionID: "cisco2",
		Input:     "show clock",
		Expect:    `[0-9]+:[0-9]+:[0-9]+`,
		Timeout:   10,
	}, deps)
	if err != nil {
		t.Fatalf("session_interact: %v", err)
	}
	t.Logf("interact output: %s", interactOut.Output)

	if !interactOut.Matched {
		t.Errorf("expected pattern match for clock output")
	}

	// Cleanup
	mcptools.HandleSessionClose(ctx, nil, mcptools.SessionCloseInput{SessionID: "cisco2"}, deps)
}

func TestMCP_SessionNotFound(t *testing.T) {
	deps := newDeps()
	ctx := context.Background()

	_, _, err := mcptools.HandleSessionExec(ctx, nil, mcptools.SessionExecInput{
		SessionID: "nonexistent",
		Command:   "show version",
	}, deps)
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
