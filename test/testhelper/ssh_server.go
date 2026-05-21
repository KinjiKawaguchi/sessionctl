package testhelper

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHServer is an in-process SSH server for integration testing.
type SSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
	device   DeviceConfig
	done     chan struct{}
	// TelnetTargets maps "host:port" strings typed in the CLI
	// to actual addresses (for testing session chaining).
	TelnetTargets map[string]string
}

// NewSSHServer creates a test SSH server with the given device config.
func NewSSHServer(device DeviceConfig) (*SSHServer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}

	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == device.Username && string(pass) == device.Password {
				return nil, nil
			}
			return nil, fmt.Errorf("auth failed")
		},
	}
	sshConfig.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	s := &SSHServer{
		listener:      listener,
		config:        sshConfig,
		device:        device,
		done:          make(chan struct{}),
		TelnetTargets: make(map[string]string),
	}
	go s.acceptLoop()
	return s, nil
}

// Addr returns the server's listen address.
func (s *SSHServer) Addr() string {
	return s.listener.Addr().String()
}

// Close stops the server.
func (s *SSHServer) Close() error {
	close(s.done)
	return s.listener.Close()
}

func (s *SSHServer) acceptLoop() {
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

func (s *SSHServer) handleConn(netConn net.Conn) {
	defer netConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}

		channel, requests, err := newChan.Accept()
		if err != nil {
			continue
		}

		go s.handleSession(channel, requests)
	}
}

func (s *SSHServer) handleSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()

	// Handle pty-req and shell requests
	for req := range reqs {
		switch req.Type {
		case "pty-req":
			req.Reply(true, nil)
		case "shell":
			req.Reply(true, nil)
			s.runCLI(ch)
			return
		default:
			req.Reply(false, nil)
		}
	}
}

func (s *SSHServer) runCLI(ch io.ReadWriter) {
	state := &DeviceState{
		Config:        s.device,
		Authenticated: true, // already authed via SSH
	}

	fmt.Fprintf(ch, "%s\r\n", s.device.Banner)
	fmt.Fprintf(ch, "%s", s.device.Prompt)

	s.readLoop(ch, state)
}

func (s *SSHServer) readLoop(ch io.ReadWriter, state *DeviceState) {
	scanner := bufio.NewScanner(ch)

	var proxyConn net.Conn
	var proxyEnded chan struct{} // closed when proxy cleanup is complete

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check if proxy ended while we were waiting for input
		if proxyEnded != nil {
			select {
			case <-proxyEnded:
				proxyConn = nil
				proxyEnded = nil
			default:
			}
		}

		// Proxy mode: forward input to telnet
		if proxyConn != nil {
			proxyConn.Write([]byte(line + "\n"))
			continue
		}

		// Start telnet proxy
		if strings.HasPrefix(line, "telnet ") {
			target := strings.TrimPrefix(line, "telnet ")
			actual, ok := s.TelnetTargets[target]
			if !ok {
				fmt.Fprintf(ch, "\r\n%% Connection refused\r\n%s", s.device.Prompt)
				continue
			}

			conn, err := net.Dial("tcp", actual)
			if err != nil {
				fmt.Fprintf(ch, "\r\nConnection refused\r\n%s", s.device.Prompt)
				continue
			}

			proxyConn = conn
			proxyEnded = make(chan struct{})
			prompt := s.device.Prompt
			fmt.Fprintf(ch, "\r\nTrying %s...\r\n", target)

			go func() {
				io.Copy(ch, conn)
				conn.Close()
				// Print prompt from this goroutine so the client
				// unblocks and sends the next command, which lets
				// scanner.Scan() return in the main loop.
				fmt.Fprintf(ch, "\r\n%s", prompt)
				close(proxyEnded)
			}()
			continue
		}

		if !HandleLine(state, line, ch) {
			return
		}
	}
}
