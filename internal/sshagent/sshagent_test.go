package sshagent

import (
	"io"
	"net"
	"os"
	"os/exec"
	"testing"

	"golang.org/x/crypto/ssh/agent"
)

// MockNetDial mocks the net.Dial function
type mockConn struct {
	net.Conn
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestAddSSHKey(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()
	
	// Test successful key addition
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		// Mock a successful ssh-add command
		return exec.Command("echo", "Identity added")
	}
	
	err := addSSHKey("/fake/id_ed25519")
	if err != nil {
		t.Errorf("addSSHKey should not return error for successful key addition: %v", err)
	}
	
	// Test failed key addition
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		// Create a command that will fail
		cmd := exec.Command("false")
		return cmd
	}
	
	err = addSSHKey("/fake/id_ed25519")
	if err == nil {
		t.Error("addSSHKey should return error for failed key addition")
	}
}

// TestStartSSHAgent tests the startSSHAgent function
func TestStartSSHAgent(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()
	
	// Save original environment variables
	origAuthSock := os.Getenv("SSH_AUTH_SOCK")
	origAgentPid := os.Getenv("SSH_AGENT_PID")
	defer func() {
		os.Setenv("SSH_AUTH_SOCK", origAuthSock)
		os.Setenv("SSH_AGENT_PID", origAgentPid)
	}()
	
	// Clear environment variables before test
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	
	// Mock successful ssh-agent start
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		// Mock ssh-agent output
		mockOutput := `SSH_AUTH_SOCK=/tmp/ssh-1234/agent.5678; export SSH_AUTH_SOCK;
SSH_AGENT_PID=1234; export SSH_AGENT_PID;
echo Agent pid 1234;`
		
		// Create a command that will output this
		cmd := exec.Command("echo", mockOutput)
		return cmd
	}
	
	err := startSSHAgent()
	if err != nil {
		t.Errorf("startSSHAgent should not return error: %v", err)
	}
	
	// Check that environment variables were set
	if os.Getenv("SSH_AUTH_SOCK") != "/tmp/ssh-1234/agent.5678" {
		t.Errorf("SSH_AUTH_SOCK not set correctly, got: %s", os.Getenv("SSH_AUTH_SOCK"))
	}
	
	if os.Getenv("SSH_AGENT_PID") != "1234" {
		t.Errorf("SSH_AGENT_PID not set correctly, got: %s", os.Getenv("SSH_AGENT_PID"))
	}
	
	// Test failed ssh-agent start
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	
	err = startSSHAgent()
	if err == nil {
		t.Error("startSSHAgent should return error for failed start")
	}
}

// MockAgentClient implements a mock SSH agent for testing
type MockAgentClient struct {
	keys []*agent.Key
}

func (m *MockAgentClient) List() ([]*agent.Key, error) {
	return m.keys, nil
}

func (m *MockAgentClient) Sign(key agent.PublicKey, data []byte) (*agent.Signature, error) {
	return nil, nil
}

func (m *MockAgentClient) Add(key agent.AddedKey) error {
	return nil
}

func (m *MockAgentClient) Remove(key agent.PublicKey) error {
	return nil
}

func (m *MockAgentClient) RemoveAll() error {
	return nil
}

func (m *MockAgentClient) Lock(passphrase []byte) error {
	return nil
}

func (m *MockAgentClient) Unlock(passphrase []byte) error {
	return nil
}

func (m *MockAgentClient) Signers() ([]agent.Signer, error) {
	return nil, nil
}

func (m *MockAgentClient) Extension(extensionType string, contents []byte) ([]byte, error) {
	return nil, nil
}