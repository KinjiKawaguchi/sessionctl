package expect

import (
	"context"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestEngine_Send(t *testing.T) {
	var buf strings.Builder
	eng := NewEngine(strings.NewReader(""), &buf, 1024)
	defer eng.Close()

	ctx := context.Background()
	err := eng.Send(ctx, "show version\n")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if buf.String() != "show version\n" {
		t.Fatalf("writer got %q, want %q", buf.String(), "show version\n")
	}
}

func TestEngine_ExpectFindsPatternInBuffer(t *testing.T) {
	deviceOutput := "Welcome to Switch\nSwitch>"
	reader := strings.NewReader(deviceOutput)

	eng := NewEngine(reader, io.Discard, 1024)
	defer eng.Close()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pattern := Pattern{Kind: PatternExact, Text: "Switch>"}
	output, err := eng.Expect(ctx, pattern)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}

	if !strings.Contains(string(output), "Switch>") {
		t.Fatalf("output %q does not contain 'Switch>'", string(output))
	}
}

func TestEngine_ExpectTimeout(t *testing.T) {
	// Pipe never closes — no EOF, purely tests context timeout
	reader, _ := io.Pipe()

	eng := NewEngine(reader, io.Discard, 1024)
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	pattern := Pattern{Kind: PatternExact, Text: "NeverGoingToMatch"}
	_, err := eng.Expect(ctx, pattern)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline error, got: %v", err)
	}
}

func TestEngine_ExpectMultiplePatterns(t *testing.T) {
	deviceOutput := "Enter Password: "
	reader := strings.NewReader(deviceOutput)

	eng := NewEngine(reader, io.Discard, 1024)
	defer eng.Close()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	patterns := []Pattern{
		{Kind: PatternExact, Text: "Username:"},
		{Kind: PatternExact, Text: "Password:"},
	}

	output, idx, err := eng.ExpectAny(ctx, patterns)
	if err != nil {
		t.Fatalf("ExpectAny failed: %v", err)
	}
	if idx != 1 {
		t.Fatalf("matched pattern index = %d, want 1", idx)
	}
	if !strings.Contains(string(output), "Password:") {
		t.Fatalf("output %q does not contain 'Password:'", string(output))
	}
}

func TestEngine_SendExpectChain(t *testing.T) {
	engineReader, deviceWriter := io.Pipe()
	deviceReader, engineWriter := io.Pipe()

	eng := NewEngine(engineReader, engineWriter, 1024)
	defer eng.Close()

	go func() {
		buf := make([]byte, 1024)
		deviceWriter.Write([]byte("Switch>"))

		n, _ := deviceReader.Read(buf)
		cmd := string(buf[:n])
		if strings.Contains(cmd, "enable") {
			deviceWriter.Write([]byte("Password:"))

			n, _ = deviceReader.Read(buf)
			_ = string(buf[:n])
			deviceWriter.Write([]byte("\nSwitch#"))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prompt := Pattern{Kind: PatternRegex, Regex: regexp.MustCompile(`Switch[#>]`)}

	_, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("Step 1 (initial prompt): %v", err)
	}

	err = eng.Send(ctx, "enable\n")
	if err != nil {
		t.Fatalf("Step 2 (send enable): %v", err)
	}

	pwPrompt := Pattern{Kind: PatternExact, Text: "Password:"}
	_, err = eng.Expect(ctx, pwPrompt)
	if err != nil {
		t.Fatalf("Step 3 (password prompt): %v", err)
	}

	err = eng.Send(ctx, "secret\n")
	if err != nil {
		t.Fatalf("Step 4 (send password): %v", err)
	}

	_, err = eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("Step 5 (privileged prompt): %v", err)
	}
}

func TestEngine_ClearBeforeExpect_PreventsStaleMatch(t *testing.T) {
	engineReader, deviceWriter := io.Pipe()
	_, engineWriter := io.Pipe()

	eng := NewEngine(engineReader, engineWriter, 1024)
	defer eng.Close()

	// Device sends initial prompt
	deviceWriter.Write([]byte("Switch>"))
	time.Sleep(50 * time.Millisecond)

	// Consume the initial prompt (simulating session_open)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	prompt := Pattern{Kind: PatternExact, Text: "Switch>"}
	eng.Expect(ctx, prompt)

	// Now device sends more output that includes the prompt again
	// (simulating leftover data or echo)
	deviceWriter.Write([]byte("Switch>"))
	time.Sleep(50 * time.Millisecond)

	// Clear the buffer before sending a command
	eng.Buffer().Clear()

	// Device will send command output after a delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		deviceWriter.Write([]byte("show version\nCisco IOS v15.2\nSwitch>"))
	}()

	// Expect should NOT match the stale "Switch>" that was cleared,
	// but SHOULD match the one after "Cisco IOS v15.2"
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}

	text := string(output)
	if !strings.Contains(text, "Cisco IOS") {
		t.Fatalf("output %q should contain command output, not be empty", text)
	}
}

func TestEngine_ExecPattern_SkipsStalePrompt(t *testing.T) {
	engineReader, deviceWriter := io.Pipe()
	deviceReader, engineWriter := io.Pipe()

	eng := NewEngine(engineReader, engineWriter, 1024)
	defer eng.Close()

	prompt := Pattern{Kind: PatternExact, Text: "Switch>"}

	// Device sends initial prompt
	deviceWriter.Write([]byte("Welcome\nSwitch>"))
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Consume initial prompt
	eng.Expect(ctx, prompt)

	// Simulate: stale prompt left in buffer
	deviceWriter.Write([]byte("Switch>"))
	time.Sleep(50 * time.Millisecond)

	// Clear buffer, send command, expect fresh output
	eng.Buffer().Clear()

	go func() {
		buf := make([]byte, 1024)
		n, _ := deviceReader.Read(buf)
		cmd := string(buf[:n])
		if strings.Contains(cmd, "show version") {
			deviceWriter.Write([]byte("show version\nVersion 15.2\nSwitch>"))
		}
	}()

	eng.Send(ctx, "show version\n")
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}

	text := string(output)
	if !strings.Contains(text, "Version 15.2") {
		t.Fatalf("expected command output, got %q", text)
	}
}

func TestEngine_DisconnectedError(t *testing.T) {
	engineReader, deviceWriter := io.Pipe()
	eng := NewEngine(engineReader, io.Discard, 1024)
	defer eng.Close()

	// Close the device side — simulates connection drop
	deviceWriter.Close()
	time.Sleep(100 * time.Millisecond)

	// Expect should return a disconnected error immediately, not timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := eng.Expect(ctx, Pattern{Kind: PatternExact, Text: "Switch>"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "disconnected") {
		t.Fatalf("expected disconnected error, got: %v", err)
	}
}

func TestEngine_SendAfterDisconnect(t *testing.T) {
	engineReader, deviceWriter := io.Pipe()
	var buf strings.Builder
	eng := NewEngine(engineReader, &buf, 1024)
	defer eng.Close()

	deviceWriter.Close()
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	err := eng.Send(ctx, "show version\n")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "disconnected") {
		t.Fatalf("expected disconnected error, got: %v", err)
	}
}

func TestEngine_StripANSI(t *testing.T) {
	// Device sends prompt with ANSI color codes
	deviceOutput := "\x1b[32mSwitch\x1b[0m# "
	reader := strings.NewReader(deviceOutput)

	eng := NewEngine(reader, io.Discard, 1024, WithStripANSI(true))
	defer eng.Close()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Pattern matches clean text — should work because ANSI was stripped at write time
	prompt := Pattern{Kind: PatternExact, Text: "Switch#"}
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}
	if strings.Contains(string(output), "\x1b") {
		t.Fatalf("output still contains ANSI: %q", string(output))
	}
}

func TestEngine_StripANSI_Disabled(t *testing.T) {
	deviceOutput := "\x1b[32mSwitch\x1b[0m# "
	reader := strings.NewReader(deviceOutput)

	eng := NewEngine(reader, io.Discard, 1024, WithStripANSI(false))
	defer eng.Close()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Match against raw data including ANSI
	prompt := Pattern{Kind: PatternExact, Text: "\x1b[0m# "}
	output, err := eng.Expect(ctx, prompt)
	if err != nil {
		t.Fatalf("Expect failed: %v", err)
	}
	if !strings.Contains(string(output), "\x1b") {
		t.Fatalf("expected ANSI in output: %q", string(output))
	}
}

func TestEngine_ExpectAny_ReturnsPromptIndex(t *testing.T) {
	// Simulates: command output followed by a Password: prompt (not the device prompt)
	engineReader, deviceWriter := io.Pipe()
	eng := NewEngine(engineReader, io.Discard, 1024)
	defer eng.Close()

	go func() {
		deviceWriter.Write([]byte("sudo: password required\nPassword: "))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Watch for both device prompt AND interactive prompts
	patterns := []Pattern{
		{Kind: PatternExact, Text: "Switch>"},       // index 0: device prompt
		{Kind: PatternExact, Text: "Password: "},    // index 1: interactive prompt
	}

	output, idx, err := eng.ExpectAny(ctx, patterns)
	if err != nil {
		t.Fatalf("ExpectAny failed: %v", err)
	}
	if idx != 1 {
		t.Fatalf("expected pattern index 1 (Password:), got %d", idx)
	}
	if !strings.Contains(string(output), "password required") {
		t.Fatalf("output %q missing preceding text", string(output))
	}
}
