package testhelper

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// TelnetServer is an in-process telnet server for integration testing.
// Uses raw TCP — no telnet protocol negotiation (sufficient for CLI simulation).
type TelnetServer struct {
	listener net.Listener
	device   DeviceConfig
	done     chan struct{}
}

// NewTelnetServer creates a test telnet server with the given device config.
func NewTelnetServer(device DeviceConfig) (*TelnetServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	s := &TelnetServer{
		listener: listener,
		device:   device,
		done:     make(chan struct{}),
	}
	go s.acceptLoop()
	return s, nil
}

// Addr returns the server's listen address.
func (s *TelnetServer) Addr() string {
	return s.listener.Addr().String()
}

// Close stops the server.
func (s *TelnetServer) Close() error {
	close(s.done)
	return s.listener.Close()
}

func (s *TelnetServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *TelnetServer) handleConn(conn net.Conn) {
	defer conn.Close()

	fmt.Fprintf(conn, "%s\r\n", s.device.Banner)

	// Login sequence
	fmt.Fprintf(conn, "Username: ")
	scanner := bufio.NewScanner(conn)

	if !scanner.Scan() {
		return
	}
	username := strings.TrimSpace(scanner.Text())

	fmt.Fprintf(conn, "Password: ")
	if !scanner.Scan() {
		return
	}
	password := strings.TrimSpace(scanner.Text())

	if username != s.device.Username || password != s.device.Password {
		fmt.Fprintf(conn, "\r\n%% Login failed\r\n")
		return
	}

	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", s.device.Prompt)

	state := &DeviceState{
		Config:        s.device,
		Authenticated: true,
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !HandleLine(state, line, conn) {
			return
		}
	}
}
