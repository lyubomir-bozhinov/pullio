package gitmanager

import (
	"os"
	"os/exec"
	"testing"
)

// MockExecCommand mocks the exec.Command function
func MockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestIsGitRepo tests the IsGitRepo function
func TestIsGitRepo(t *testing.T) {
	// Save the original exec.Command and restore it at the end
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()
	
	// Test case: Valid Git repository
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "rev-parse" {
			return MockExecCommand("echo", "true")
		}
		return exec.Command("echo", "command not mocked")
	}
	
	if !IsGitRepo("/fake/repo") {
		t.Error("IsGitRepo should return true for a valid Git repository")
	}
	
	// Test case: Not a Git repository
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "rev-parse" {
			cmd := MockExecCommand("false")
			return cmd
		}
		return exec.Command("echo", "command not mocked")
	}
	
	if IsGitRepo("/fake/repo") {
		t.Error("IsGitRepo should return false for an invalid Git repository")
	}
}

// TestHasOriginRemote tests the HasOriginRemote function
func TestHasOriginRemote(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()
	
	// Test case: Repository has origin remote
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "remote" {
			return MockExecCommand("echo", "git@github.com:user/repo.git")
		}
		return exec.Command("echo", "command not mocked")
	}
	
	if !HasOriginRemote("/fake/repo") {
		t.Error("HasOriginRemote should return true when origin remote exists")
	}
	
	// Test case: Repository doesn't have origin remote
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "remote" {
			cmd := MockExecCommand("false")
			return cmd
		}
		return exec.Command("echo", "command not mocked")
	}
	
	if HasOriginRemote("/fake/repo") {
		t.Error("HasOriginRemote should return false when origin remote doesn't exist")
	}
}

// TestDetectDefaultBranch tests the DetectDefaultBranch function
func TestDetectDefaultBranch(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()
	
	// Test case: Default branch via symbolic-ref
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "symbolic-ref" {
			return MockExecCommand("echo", "refs/remotes/origin/main")
		}
		return exec.Command("echo", "command not mocked")
	}
	
	branch, err := DetectDefaultBranch("/fake/repo", []string{"main", "master"})
	if err != nil || branch != "main" {
		t.Errorf("DetectDefaultBranch should return 'main', got '%s', error: %v", branch, err)
	}
	
	// Test case: Default branch via remote show
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "symbolic-ref" {
			return MockExecCommand("false")
		}
		if command == "git" && args[0] == "remote" && args[1] == "show" {
			return MockExecCommand("echo", "  HEAD branch: develop")
		}
		return exec.Command("echo", "command not mocked")
	}
	
	branch, err = DetectDefaultBranch("/fake/repo", []string{"main", "master"})
	if err != nil || branch != "develop" {
		t.Errorf("DetectDefaultBranch should return 'develop', got '%s', error: %v", branch, err)
	}
	
	// Test case: Default branch via fallback
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "git" && args[0] == "symbolic-ref" {
			return MockExecCommand("false")
		}
		if command == "git" && args[0] == "remote" && args[1] == "show" {
			return MockExecCommand("false")
		}
		if command == "git" && args[0] == "show-ref" && args[2] == "refs/heads/main" {
			return MockExecCommand("false")
		}
		if command == "git" && args[0] == "show-ref" && args[2] == "refs/heads/master" {
			return MockExecCommand("echo", "ref")
		}
		return exec.Command("echo", "command not mocked")
	}
	
	branch, err = DetectDefaultBranch("/fake/repo", []string{"main", "master"})
	if err != nil || branch != "master" {
		t.Errorf("DetectDefaultBranch should return 'master', got '%s', error: %v", branch, err)
	}
	
	// Test case: No default branch found
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		return MockExecCommand("false")
	}
	
	branch, err = DetectDefaultBranch("/fake/repo", []string{"main", "master"})
	if err == nil {
		t.Errorf("DetectDefaultBranch should return an error when no branch is found, got '%s'", branch)
	}
}

// TestHelperProcess isn't a real test, it's used by MockExecCommand
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	
	// Get the command and arguments after "--"
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	
	if len(args) == 0 {
		os.Exit(1)
	}
	
	// Mock different commands
	switch args[0] {
	case "echo":
		if len(args) > 1 {
			os.Stdout.WriteString(args[1])
		}
		os.Exit(0)
	case "false":
		os.Exit(1)
	default:
		os.Exit(1)
	}
}