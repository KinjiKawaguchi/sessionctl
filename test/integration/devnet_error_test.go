//go:build devnet

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
)

func TestDevNet_InvalidCommand(t *testing.T) {
	eng := connectDevNet(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eng.Expect(ctx, iosPrompt)
	eng.Send(ctx, "terminal length 0\n")
	eng.Expect(ctx, iosPrompt)

	// Send a command that doesn't exist
	eng.Send(ctx, "show nonexistent\n")

	// Expect error pattern OR prompt
	errorPattern := expect.Pattern{Kind: expect.PatternExact, Text: "% Invalid"}
	output, idx, err := eng.ExpectAny(ctx, []expect.Pattern{errorPattern, iosPrompt})
	if err != nil {
		t.Fatalf("expect after invalid command: %v", err)
	}

	text := string(output)
	t.Logf("invalid command output:\n%s", text)

	// Should contain error marker
	if idx == 0 {
		t.Logf("error pattern matched correctly")
		// Still need to consume until prompt
		eng.Expect(ctx, iosPrompt)
	} else if strings.Contains(text, "% Invalid") || strings.Contains(text, "% Ambiguous") {
		t.Logf("error text found in output before prompt")
	} else {
		t.Errorf("expected error pattern in output, got: %s", text)
	}
}

func TestDevNet_EmptyCommand(t *testing.T) {
	eng := connectDevNet(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eng.Expect(ctx, iosPrompt)

	// Send empty line - should just get another prompt
	eng.Send(ctx, "\n")
	_, err := eng.Expect(ctx, iosPrompt)
	if err != nil {
		t.Fatalf("expect after empty command: %v", err)
	}
}

func TestDevNet_LongOutput(t *testing.T) {
	eng := connectDevNet(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	eng.Expect(ctx, iosPrompt)
	eng.Send(ctx, "terminal length 0\n")
	eng.Expect(ctx, iosPrompt)

	// show running-config produces long output
	eng.Send(ctx, "show running-config\n")
	output, err := eng.Expect(ctx, iosPrompt)
	if err != nil {
		t.Fatalf("expect after show run: %v", err)
	}

	text := string(output)
	lines := strings.Split(text, "\n")
	t.Logf("show running-config: %d lines", len(lines))

	if len(lines) < 10 {
		t.Errorf("expected substantial output, got %d lines", len(lines))
	}
	// Long output should contain common config sections
	if !strings.Contains(text, "interface") && !strings.Contains(text, "version") && !strings.Contains(text, "line") {
		t.Errorf("output missing expected config content")
	}
}
