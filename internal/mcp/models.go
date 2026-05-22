package mcptools

import "time"

// SessionOpenInput is the input for session_open.
type SessionOpenInput struct {
	Name     string `json:"name,omitempty" mcp:"human-readable session name (auto-generated if omitted)"`
	Host     string `json:"host" mcp:"host to connect to"`
	Port     int    `json:"port,omitempty" mcp:"port number (default: 22 for SSH, 23 for telnet)"`
	Protocol string `json:"protocol,omitempty" mcp:"ssh (default) or telnet"`
	Username string `json:"username" mcp:"authentication username"`
	Password string `json:"password" mcp:"authentication password"`
	Profile   string `json:"profile,omitempty" mcp:"device profile: cisco_ios, yamaha_rtx, unix"`
	StripANSI *bool  `json:"strip_ansi,omitempty" mcp:"strip ANSI escape sequences from output (default: true)"`
}

// SessionOpenOutput is the output for session_open.
type SessionOpenOutput struct {
	SessionID string `json:"session_id"`
	Host      string `json:"host"`
	Profile   string `json:"profile"`
	Output    string `json:"output"`
}

// SessionExecInput is the input for session_exec.
type SessionExecInput struct {
	SessionID string `json:"session_id" mcp:"session to execute command on"`
	Command   string `json:"command" mcp:"command to execute"`
	Timeout   int    `json:"timeout,omitempty" mcp:"timeout in seconds (default: 10)"`
}

// SessionExecOutput is the output for session_exec.
type SessionExecOutput struct {
	Output   string `json:"output"`
	Prompted bool   `json:"prompted"` // true if an interactive prompt (Password: etc) was detected
}

// SessionInteractInput is the input for session_interact.
type SessionInteractInput struct {
	SessionID string `json:"session_id" mcp:"session to interact with"`
	Input     string `json:"input" mcp:"text to send (e.g. password response)"`
	Expect    string `json:"expect,omitempty" mcp:"regex pattern to wait for after sending"`
	Timeout   int    `json:"timeout,omitempty" mcp:"timeout in seconds (default: 10)"`
}

// SessionInteractOutput is the output for session_interact.
type SessionInteractOutput struct {
	Output  string `json:"output"`
	Matched bool   `json:"matched"`
}

// SessionChainInput is the input for session_chain.
type SessionChainInput struct {
	SessionID string `json:"session_id" mcp:"parent session ID"`
	Command   string `json:"command" mcp:"command to start sub-session (e.g. telnet 10.1.31.251)"`
	Name      string `json:"name,omitempty" mcp:"name for the new session"`
	Profile   string `json:"profile,omitempty" mcp:"device profile for the child device"`
}

// SessionChainOutput is the output for session_chain.
type SessionChainOutput struct {
	SessionID string `json:"session_id"`
	Output    string `json:"output"`
}

// SessionListInput is the (empty) input for session_list.
type SessionListInput struct{}

// SessionListOutput is the output for session_list.
type SessionListOutput struct {
	Sessions []SessionInfo `json:"sessions"`
}

// SessionInfo describes one active session.
type SessionInfo struct {
	ID       string    `json:"id"`
	Host     string    `json:"host"`
	Profile  string    `json:"profile"`
	Depth    int       `json:"depth"`
	LastUsed time.Time `json:"last_used"`
}

// SessionCloseInput is the input for session_close.
type SessionCloseInput struct {
	SessionID string `json:"session_id" mcp:"session to close"`
}

// SessionCloseOutput is the output for session_close.
type SessionCloseOutput struct {
	SessionID string `json:"session_id"`
	Closed    bool   `json:"closed"`
}
