package sshagent

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lyubomir-bozhinov/pullio/internal/logger"
	"golang.org/x/crypto/ssh/agent"
)

// ExecCommand is a variable that holds the exec.Command function.
// It can be overridden in tests to mock exec.Command.
var ExecCommand = exec.Command

// NetDial is a variable that holds the net.Dial function.
// It can be overridden in tests to mock net.Dial.
var NetDial = net.Dial

// EnsureAgentAndKey checks if ssh-agent is running and if the key is loaded.
// If not, it attempts to start the agent and add the key.
func EnsureAgentAndKey(sshKeyPath string) error {
	// Expand ~ to home directory if present
	if strings.HasPrefix(sshKeyPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		sshKeyPath = filepath.Join(homeDir, sshKeyPath[1:])
	}
	
	// Check if the key file exists
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key does not exist: %s", sshKeyPath)
	}
	
	// Get SSH_AUTH_SOCK
	authSock := os.Getenv("SSH_AUTH_SOCK")
	
	// If SSH_AUTH_SOCK is not set, try to start ssh-agent
	if authSock == "" {
		logger.Debug("SSH_AUTH_SOCK not set, attempting to start ssh-agent")
		if err := startSSHAgent(); err != nil {
			return fmt.Errorf("failed to start ssh-agent: %w", err)
		}
		authSock = os.Getenv("SSH_AUTH_SOCK")
		if authSock == "" {
			return errors.New("SSH_AUTH_SOCK is still empty after starting ssh-agent")
		}
	}
	
	// Connect to the SSH agent
	conn, err := NetDial("unix", authSock)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH agent socket at %s: %w", authSock, err)
	}
	defer conn.Close()
	
	// Check if the key is already loaded
	ag := agent.NewClient(conn)
	keys, err := ag.List()
	if err != nil {
		return fmt.Errorf("failed to list keys from SSH agent: %w", err)
	}
	
	keyFilename := filepath.Base(sshKeyPath)
	keyLoaded := false
	
	for _, key := range keys {
		// Key comments often contain the filename
		if strings.Contains(key.Comment, keyFilename) {
			logger.Debug("SSH key %s is already loaded in agent", keyFilename)
			keyLoaded = true
			break
		}
	}
	
	// Add the key if it's not loaded
	if !keyLoaded {
		logger.Info("Adding SSH key: %s", sshKeyPath)
		if err := addSSHKey(sshKeyPath); err != nil {
			return fmt.Errorf("failed to add SSH key %s to agent: %w", sshKeyPath, err)
		}
		logger.Success("SSH key added successfully")
	}
	
	return nil
}

// startSSHAgent starts the ssh-agent process
func startSSHAgent() error {
	logger.Info("Starting ssh-agent...")
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, the approach is different
		cmd = ExecCommand("powershell", "-Command", "Start-Service ssh-agent")
	} else {
		// Unix-like systems
		cmd = ExecCommand("ssh-agent", "-s")
	}
	
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-agent command failed: %w", err)
	}
	
	// Parse the output to get environment variables
	outputStr := string(output)
	logger.Debug("ssh-agent output: %s", outputStr)
	
	// Windows has a different approach to set environment variables
	if runtime.GOOS == "windows" {
		// Just check if the service is running
		return nil
	}
	
	// Parse output like:
	// SSH_AUTH_SOCK=/tmp/ssh-XXXXXX/agent.YYYY; export SSH_AUTH_SOCK;
	// SSH_AGENT_PID=ZZZZZ; export SSH_AGENT_PID;
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "SSH_AUTH_SOCK=") {
			parts := strings.Split(line, ";")
			if len(parts) > 0 {
				sockParts := strings.Split(parts[0], "=")
				if len(sockParts) > 1 {
					os.Setenv("SSH_AUTH_SOCK", sockParts[1])
				}
			}
		} else if strings.Contains(line, "SSH_AGENT_PID=") {
			parts := strings.Split(line, ";")
			if len(parts) > 0 {
				pidParts := strings.Split(parts[0], "=")
				if len(pidParts) > 1 {
					os.Setenv("SSH_AGENT_PID", pidParts[1])
				}
			}
		}
	}
	
	// Give the agent a moment to fully initialize
	time.Sleep(100 * time.Millisecond)
	
	// Verify that SSH_AUTH_SOCK is now set
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return errors.New("failed to set SSH_AUTH_SOCK environment variable")
	}
	
	return nil
}

// addSSHKey adds the specified SSH key to the agent
func addSSHKey(sshKeyPath string) error {
	// Check for different platforms
	var cmd *exec.Cmd
	
	if runtime.GOOS == "windows" {
		cmd = ExecCommand("powershell", "-Command", fmt.Sprintf("ssh-add %s", sshKeyPath))
	} else {
		cmd = ExecCommand("ssh-add", sshKeyPath)
	}
	
	// Allow user to enter passphrase if needed
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-add command failed: %w", err)
	}
	
	return nil
}
