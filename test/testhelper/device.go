package testhelper

import (
	"fmt"
	"io"
	"strings"
)

// CommandHandler maps a command prefix to a response string.
type CommandHandler struct {
	Prefix   string
	Response string
}

// DeviceConfig defines a simulated device's CLI behavior.
type DeviceConfig struct {
	Hostname     string
	Username     string
	Password     string
	EnablePass   string
	Prompt       string // e.g., "Switch>"
	EnablePrompt string // e.g., "Switch#"
	Banner       string
	Commands     []CommandHandler
}

// DeviceState tracks the current state of a simulated device session.
type DeviceState struct {
	Config          DeviceConfig
	Authenticated   bool
	Enabled         bool
	WaitingForEnable bool
}

// HandleLine processes a single line of input and writes the response.
func HandleLine(state *DeviceState, line string, w io.Writer) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		writePrompt(state, w)
		return true
	}

	if line == "exit" || line == "quit" {
		fmt.Fprintf(w, "\r\nConnection closed.\r\n")
		return false
	}

	if line == "enable" && !state.Enabled {
		state.WaitingForEnable = true
		fmt.Fprintf(w, "Password: ")
		return true
	}

	if state.WaitingForEnable {
		state.WaitingForEnable = false
		if line == state.Config.EnablePass {
			state.Enabled = true
			fmt.Fprintf(w, "\r\n")
			writePrompt(state, w)
			return true
		}
		fmt.Fprintf(w, "\r\n%% Access denied\r\n")
		writePrompt(state, w)
		return true
	}

	// Look up command
	for _, cmd := range state.Config.Commands {
		if strings.HasPrefix(line, cmd.Prefix) {
			fmt.Fprintf(w, "\r\n%s\r\n", cmd.Response)
			writePrompt(state, w)
			return true
		}
	}

	fmt.Fprintf(w, "\r\n%% Invalid input detected\r\n")
	writePrompt(state, w)
	return true
}

func writePrompt(state *DeviceState, w io.Writer) {
	if state.Enabled {
		fmt.Fprintf(w, "%s", state.Config.EnablePrompt)
	} else {
		fmt.Fprintf(w, "%s", state.Config.Prompt)
	}
}

// CiscoSwitchConfig returns a DeviceConfig for a simulated Cisco switch.
func CiscoSwitchConfig() DeviceConfig {
	return DeviceConfig{
		Hostname:     "Switch",
		Username:     "admin",
		Password:     "admin123",
		EnablePass:   "enable123",
		Prompt:       "Switch>",
		EnablePrompt: "Switch#",
		Banner:       "Cisco IOS Software, C3560CX Software",
		Commands: []CommandHandler{
			{Prefix: "show version", Response: "Cisco IOS Software, C3560CX Software\nVersion 15.2(7)E2"},
			{Prefix: "show running-config", Response: "Building configuration...\nhostname Switch"},
			{Prefix: "show interfaces status", Response: "Port    Name    Status    Vlan\nGi0/1           connected 1"},
			{Prefix: "terminal length 0", Response: ""},
		},
	}
}

// ASAConfig returns a DeviceConfig for a simulated Cisco ASA.
func ASAConfig() DeviceConfig {
	return DeviceConfig{
		Hostname:     "ciscoasa",
		Username:     "admin",
		Password:     "admin123",
		EnablePass:   "enable123",
		Prompt:       "ciscoasa>",
		EnablePrompt: "ciscoasa#",
		Banner:       "Cisco Adaptive Security Appliance Software Version 9.8",
		Commands: []CommandHandler{
			{Prefix: "show version", Response: "Cisco Adaptive Security Appliance Software Version 9.8"},
			{Prefix: "show running-config", Response: "ASA Version 9.8\nhostname ciscoasa"},
		},
	}
}

// YamahaRTXConfig returns a DeviceConfig for a simulated Yamaha RTX router.
func YamahaRTXConfig() DeviceConfig {
	return DeviceConfig{
		Hostname:     "RTX1210",
		Username:     "admin",
		Password:     "admin123",
		EnablePass:   "admin123",
		Prompt:       "RTX1210>",
		EnablePrompt: "RTX1210#",
		Banner:       "RTX1210 Rev.14.01.40",
		Commands: []CommandHandler{
			{Prefix: "show config", Response: "ip route default gateway dhcp lan2"},
			{Prefix: "show status", Response: "RTX1210 Rev.14.01.40\nUptime: 1 day"},
		},
	}
}
