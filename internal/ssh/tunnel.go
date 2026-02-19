package ssh

import (
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents an SSH tunnel
type Tunnel struct {
	ID          string
	LocalAddr   string
	RemoteAddr  string
	SSHAddr     string
	SSHConfig   *ssh.ClientConfig
	client      *ssh.Client
	listener    net.Listener
	active      bool
	stopChan    chan struct{}
}

// NewTunnel creates a new SSH tunnel
func NewTunnel(id, localAddr, remoteAddr, sshAddr string, config *ssh.ClientConfig) *Tunnel {
	return &Tunnel{
		ID:         id,
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		SSHAddr:    sshAddr,
		SSHConfig:  config,
		stopChan:   make(chan struct{}),
	}
}

// Start starts the SSH tunnel
func (t *Tunnel) Start() error {
	// Connect to SSH server
	client, err := ssh.Dial("tcp", t.SSHAddr, t.SSHConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	t.client = client

	// Start local listener
	listener, err := net.Listen("tcp", t.LocalAddr)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to listen on local address: %w", err)
	}
	t.listener = listener
	t.active = true

	// Start accepting connections
	go t.acceptConnections()

	return nil
}

// acceptConnections accepts and forwards connections
func (t *Tunnel) acceptConnections() {
	for {
		select {
		case <-t.stopChan:
			return
		default:
			// Set deadline for accept
			t.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			
			localConn, err := t.listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return
			}

			// Forward connection
			go t.forwardConnection(localConn)
		}
	}
}

// forwardConnection forwards a connection through the SSH tunnel
func (t *Tunnel) forwardConnection(localConn net.Conn) {
	defer localConn.Close()

	// Dial remote address through SSH
	remoteConn, err := t.client.Dial("tcp", t.RemoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Copy data bidirectionally
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	<-done
}

// Stop stops the tunnel
func (t *Tunnel) Stop() error {
	t.active = false
	close(t.stopChan)

	if t.listener != nil {
		t.listener.Close()
	}

	if t.client != nil {
		return t.client.Close()
	}

	return nil
}

// IsActive returns whether the tunnel is active
func (t *Tunnel) IsActive() bool {
	return t.active
}

// CreateSSHConfig creates SSH client configuration
func CreateSSHConfig(user string, authMethods ...ssh.AuthMethod) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, verify host keys
		Timeout:         30 * time.Second,
	}
}

// CreateAuthMethodPassword creates password authentication
func CreateAuthMethodPassword(password string) ssh.AuthMethod {
	return ssh.Password(password)
}

// CreateAuthMethodKey creates key-based authentication
func CreateAuthMethodKey(privateKey []byte) (ssh.AuthMethod, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

