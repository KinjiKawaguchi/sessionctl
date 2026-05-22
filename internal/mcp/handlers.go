package mcptools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/device"
	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
	"github.com/KinjiKawaguchi/sessionctl/internal/session"
	"github.com/KinjiKawaguchi/sessionctl/internal/transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Deps holds shared dependencies for all handlers. Data-only, no interfaces.
type Deps struct {
	Store    *session.Store
	Profiles *device.ProfileTable
	counter  int
}

// NewDeps creates handler dependencies.
func NewDeps() *Deps {
	return &Deps{
		Store:    session.NewStore(),
		Profiles: device.DefaultTable(),
	}
}

func (d *Deps) nextID(prefix string) string {
	d.counter++
	return fmt.Sprintf("%s-%d", prefix, d.counter)
}

// HandleSessionOpen opens a new SSH or telnet session.
func HandleSessionOpen(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SessionOpenInput,
	deps *Deps,
) (*mcp.CallToolResult, SessionOpenOutput, error) {
	protocol := input.Protocol
	if protocol == "" {
		protocol = "ssh"
	}

	profile := input.Profile
	if profile == "" {
		profile = "unix"
	}

	var tr transport.Transport
	var err error

	switch protocol {
	case "ssh":
		tr, err = transport.DialSSH(transport.SSHConfig{
			Host:     input.Host,
			Port:     input.Port,
			Username: input.Username,
			Password: input.Password,
		})
	case "telnet":
		tr, err = transport.DialTelnet(transport.TelnetConfig{
			Host: input.Host,
			Port: input.Port,
		})
	default:
		return nil, SessionOpenOutput{}, fmt.Errorf("unknown protocol: %s", protocol)
	}

	if err != nil {
		return nil, SessionOpenOutput{}, fmt.Errorf("connect: %w", err)
	}

	stripANSI := input.StripANSI == nil || *input.StripANSI
	eng := expect.NewEngine(tr.Stdout, tr.Stdin, 1048576, expect.WithStripANSI(stripANSI))

	connIdx := deps.Store.AddConnection(session.ConnectionEntry{
		Engine: eng,
		Closer: &tr,
	})

	sessionID := input.Name
	if sessionID == "" {
		sessionID = deps.nextID("sess")
	}

	deps.Store.AddSession(session.SessionEntry{
		ID:         sessionID,
		ConnIndex:  connIdx,
		Depth:      0,
		Host:       input.Host,
		ProfileKey: profile,
	})

	// Wait briefly for initial output (banner + prompt)
	waitCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	prof, _ := deps.Profiles.Lookup(profile)
	var initialOutput string
	if len(prof.PromptPatterns) > 0 {
		output, _ := eng.Expect(waitCtx, prof.PromptPatterns[0])
		initialOutput = string(output)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Session %s opened to %s\n%s", sessionID, input.Host, initialOutput)},
		},
	}, SessionOpenOutput{
		SessionID: sessionID,
		Host:      input.Host,
		Profile:   profile,
		Output:    initialOutput,
	}, nil
}

// HandleSessionExec executes a command on an existing session.
func HandleSessionExec(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SessionExecInput,
	deps *Deps,
) (*mcp.CallToolResult, SessionExecOutput, error) {
	entry, ok := deps.Store.GetSession(input.SessionID)
	if !ok {
		return nil, SessionExecOutput{}, fmt.Errorf("session not found: %s", input.SessionID)
	}

	conn := deps.Store.GetConnection(entry.ConnIndex)
	eng := conn.Engine

	timeout := time.Duration(input.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Flush stale data to prevent matching a leftover prompt
	eng.Buffer().Clear()

	if err := eng.Send(execCtx, input.Command+"\n"); err != nil {
		return nil, SessionExecOutput{}, fmt.Errorf("send: %w", err)
	}

	prof, _ := deps.Profiles.Lookup(entry.ProfileKey)

	// Build watch patterns: device prompts + interactive prompts (Password: etc)
	watchPatterns := make([]expect.Pattern, 0, len(prof.PromptPatterns)+len(prof.LoginPatterns))
	watchPatterns = append(watchPatterns, prof.PromptPatterns...)
	promptBoundary := len(watchPatterns)
	watchPatterns = append(watchPatterns, prof.LoginPatterns...)

	var output []byte
	var matchIdx int
	if len(watchPatterns) > 0 {
		if prof.PagerPattern.Kind != 0 || prof.PagerPattern.Text != "" || prof.PagerPattern.Regex != nil {
			pager := expect.Pager{Pattern: prof.PagerPattern, Response: prof.PagerResponse}
			output, matchIdx, _ = eng.ExpectWithPager(execCtx, watchPatterns, pager)
		} else {
			output, matchIdx, _ = eng.ExpectAny(execCtx, watchPatterns)
		}
	}

	prompted := matchIdx >= promptBoundary

	text := string(output)
	if prompted {
		text += "\n[interactive prompt detected — use session_interact to respond]"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, SessionExecOutput{Output: text, Prompted: prompted}, nil
}

// HandleSessionInteract sends input and optionally waits for a pattern.
func HandleSessionInteract(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SessionInteractInput,
	deps *Deps,
) (*mcp.CallToolResult, SessionInteractOutput, error) {
	entry, ok := deps.Store.GetSession(input.SessionID)
	if !ok {
		return nil, SessionInteractOutput{}, fmt.Errorf("session not found: %s", input.SessionID)
	}

	conn := deps.Store.GetConnection(entry.ConnIndex)
	eng := conn.Engine

	timeout := time.Duration(input.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	interactCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := eng.Send(interactCtx, input.Input+"\n"); err != nil {
		return nil, SessionInteractOutput{}, fmt.Errorf("send: %w", err)
	}

	var output []byte
	matched := false

	if input.Expect != "" {
		re, err := regexp.Compile(input.Expect)
		if err != nil {
			return nil, SessionInteractOutput{}, fmt.Errorf("invalid pattern: %w", err)
		}
		pat := expect.Pattern{Kind: expect.PatternRegex, Regex: re}
		output, err = eng.Expect(interactCtx, pat)
		if err == nil {
			matched = true
		}
	} else {
		// No pattern specified: wait briefly and return whatever is in the buffer
		time.Sleep(500 * time.Millisecond)
		output = eng.Buffer().Bytes()
	}

	text := string(output)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, SessionInteractOutput{Output: text, Matched: matched}, nil
}

// HandleSessionChain starts a sub-session within an existing session.
func HandleSessionChain(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SessionChainInput,
	deps *Deps,
) (*mcp.CallToolResult, SessionChainOutput, error) {
	entry, ok := deps.Store.GetSession(input.SessionID)
	if !ok {
		return nil, SessionChainOutput{}, fmt.Errorf("session not found: %s", input.SessionID)
	}

	conn := deps.Store.GetConnection(entry.ConnIndex)
	eng := conn.Engine

	chainCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := eng.Send(chainCtx, input.Command+"\n"); err != nil {
		return nil, SessionChainOutput{}, fmt.Errorf("send chain command: %w", err)
	}

	// Wait for initial output from the chained device
	time.Sleep(1 * time.Second)
	initialOutput := eng.Buffer().Bytes()

	childID := input.Name
	if childID == "" {
		childID = deps.nextID("chain")
	}

	profile := input.Profile
	if profile == "" {
		profile = entry.ProfileKey
	}

	// Extract host from command (e.g., "telnet 10.1.31.251" → "10.1.31.251")
	host := input.Command
	if parts := strings.Fields(input.Command); len(parts) >= 2 {
		host = parts[len(parts)-1]
	}

	deps.Store.PushContext(entry.ConnIndex, session.SessionEntry{
		ID:         childID,
		Depth:      entry.Depth + 1,
		Host:       host,
		ProfileKey: profile,
	})

	text := string(initialOutput)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Chained session %s opened\n%s", childID, text)},
		},
	}, SessionChainOutput{SessionID: childID, Output: text}, nil
}

// HandleSessionList lists all active sessions.
func HandleSessionList(
	_ context.Context,
	_ *mcp.CallToolRequest,
	_ SessionListInput,
	deps *Deps,
) (*mcp.CallToolResult, SessionListOutput, error) {
	ids := deps.Store.ListSessions()
	sessions := make([]SessionInfo, 0, len(ids))

	for _, id := range ids {
		entry, ok := deps.Store.GetSession(id)
		if !ok {
			continue
		}
		sessions = append(sessions, SessionInfo{
			ID:       entry.ID,
			Host:     entry.Host,
			Profile:  entry.ProfileKey,
			Depth:    entry.Depth,
			LastUsed: entry.LastUsed,
		})
	}

	var sb strings.Builder
	for _, s := range sessions {
		fmt.Fprintf(&sb, "- %s (%s) depth=%d host=%s\n", s.ID, s.Profile, s.Depth, s.Host)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, SessionListOutput{Sessions: sessions}, nil
}

// HandleSessionClose closes a session.
func HandleSessionClose(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SessionCloseInput,
	deps *Deps,
) (*mcp.CallToolResult, SessionCloseOutput, error) {
	entry, ok := deps.Store.GetSession(input.SessionID)
	if !ok {
		return nil, SessionCloseOutput{}, fmt.Errorf("session not found: %s", input.SessionID)
	}

	conn := deps.Store.GetConnection(entry.ConnIndex)

	if entry.Depth > 0 {
		// Chained session: send exit, wait for parent prompt, pop context
		closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		conn.Engine.Buffer().Clear()
		conn.Engine.Send(closeCtx, "exit\n")

		// Find parent session's profile to wait for its prompt
		parentDepth := entry.Depth - 1
		for _, sid := range deps.Store.ListSessions() {
			if se, ok := deps.Store.GetSession(sid); ok && se.ConnIndex == entry.ConnIndex && se.Depth == parentDepth {
				if prof, ok := deps.Profiles.Lookup(se.ProfileKey); ok && len(prof.PromptPatterns) > 0 {
					conn.Engine.Expect(closeCtx, prof.PromptPatterns[0])
				}
				break
			}
		}
		deps.Store.PopContext(entry.ConnIndex)
	} else {
		// Root session: CloseConnection handles Engine + Transport cleanup
		deps.Store.CloseConnection(entry.ConnIndex)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Session %s closed", input.SessionID)},
		},
	}, SessionCloseOutput{SessionID: input.SessionID, Closed: true}, nil
}
