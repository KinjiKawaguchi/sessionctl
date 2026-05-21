package testhelper

import (
	"fmt"
	"net"
)

// NewSSHServerOnPort creates a test SSH server on a specific port.
func NewSSHServerOnPort(device DeviceConfig, port int) (*SSHServer, error) {
	srv, err := NewSSHServer(device)
	if err != nil {
		return nil, err
	}
	// Close the random-port listener and re-listen on fixed port
	srv.listener.Close()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}
	srv.listener = listener
	srv.done = make(chan struct{})
	go srv.acceptLoop()
	return srv, nil
}

// NewTelnetServerOnPort creates a test telnet server on a specific port.
func NewTelnetServerOnPort(device DeviceConfig, port int) (*TelnetServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}

	s := &TelnetServer{
		listener: listener,
		device:   device,
		done:     make(chan struct{}),
	}
	go s.acceptLoop()
	return s, nil
}
