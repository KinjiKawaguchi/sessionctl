package transport

import (
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

// TransportKind identifies the connection protocol.
type TransportKind uint8

const (
	TransportSSH TransportKind = iota
	TransportTelnet
)

// Transport is a tagged union holding either an SSH or Telnet connection.
// No interface — Kind field determines which fields are valid.
type Transport struct {
	Kind       TransportKind
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	TelnetConn net.Conn
	Stdin      io.WriteCloser
	Stdout     io.Reader
}

// SSHConfig holds SSH connection parameters.
type SSHConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	HostKeyCheck   ssh.HostKeyCallback
}

// DialSSH establishes an SSH connection and returns a Transport.
func DialSSH(cfg SSHConfig) (Transport, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.HostKeyCheck == nil {
		cfg.HostKeyCheck = ssh.InsecureIgnoreHostKey()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	sshConfig := &ssh.ClientConfig{
		User: cfg.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
			ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = cfg.Password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: cfg.HostKeyCheck,
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return Transport{}, fmt.Errorf("ssh dial: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return Transport{}, fmt.Errorf("ssh session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		session.Close()
		client.Close()
		return Transport{}, fmt.Errorf("pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return Transport{}, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return Transport{}, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return Transport{}, fmt.Errorf("shell: %w", err)
	}

	return Transport{
		Kind:       TransportSSH,
		SSHClient:  client,
		SSHSession: session,
		Stdin:      stdin,
		Stdout:     stdout,
	}, nil
}

// TelnetConfig holds Telnet connection parameters.
type TelnetConfig struct {
	Host string
	Port int
}

// DialTelnet establishes a raw TCP connection and returns a Transport.
func DialTelnet(cfg TelnetConfig) (Transport, error) {
	if cfg.Port == 0 {
		cfg.Port = 23
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return Transport{}, fmt.Errorf("telnet dial: %w", err)
	}

	return Transport{
		Kind:       TransportTelnet,
		TelnetConn: conn,
		Stdin:      conn.(io.WriteCloser),
		Stdout:     conn,
	}, nil
}

// Close releases all resources based on Kind.
func (t *Transport) Close() error {
	switch t.Kind {
	case TransportSSH:
		if t.SSHSession != nil {
			t.SSHSession.Close()
		}
		if t.SSHClient != nil {
			return t.SSHClient.Close()
		}
	case TransportTelnet:
		if t.TelnetConn != nil {
			return t.TelnetConn.Close()
		}
	}
	return nil
}
