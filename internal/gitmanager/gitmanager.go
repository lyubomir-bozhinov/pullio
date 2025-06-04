package gitmanager

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lyubomir-bozhinov/pullio/internal/logger"
)

var ExecCommand = exec.Command

type RepoResult struct {
	Path         string
	Branch       string
	Success      bool
	ErrorMessage string
}

func runGitCommand(dir string, args ...string) (string, error) {
	cmd := ExecCommand("git", args...)
	cmd.Dir = dir
	
	logger.Debug("Running git %s in %s", strings.Join(args, " "), dir)
	
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	
	if err != nil {
		return outputStr, fmt.Errorf("git command failed: %v: %s", err, outputStr)
	}
	
	return outputStr, nil
}

func IsGitRepo(dir string) bool {
	_, err := runGitCommand(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func HasOriginRemote(dir string) bool {
	_, err := runGitCommand(dir, "remote", "get-url", "origin")
	return err == nil
}

func DetectDefaultBranch(dir string, fallbacks []string) (string, error) {
	// Method 1: Check symbolic ref for origin/HEAD
	output, err := runGitCommand(dir, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err == nil {
		branch := strings.TrimPrefix(output, "refs/remotes/origin/")
		logger.Debug("Found default branch via symbolic-ref: %s", branch)
		return branch, nil
	}
	
	// Method 2: Use git remote show origin
	output, err = runGitCommand(dir, "remote", "show", "origin")
	if err == nil {
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "HEAD branch:") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					branch := parts[len(parts)-1]
					logger.Debug("Found default branch via remote show: %s", branch)
					return branch, nil
				}
			}
		}
	}
	
	// Method 3: Check for common branch names
	for _, branch := range fallbacks {
		_, err := runGitCommand(dir, "show-ref", "--quiet", "refs/heads/"+branch)
		if err == nil {
			logger.Debug("Found default branch via fallback: %s", branch)
			return branch, nil
		}
	}
	
	return "", fmt.Errorf("could not detect default branch")
}

func CheckoutBranch(dir, branch string) error {
	_, err := runGitCommand(dir, "checkout", "-q", branch)
	return err
}

func Pull(dir string) error {
	_, err := runGitCommand(dir, "pull", "-q")
	return err
}

func ProcessRepository(repoPath string, defaultBranches []string) RepoResult {
	logger.RepoHeader(repoPath)
	
	result := RepoResult{
		Path:    repoPath,
		Success: false,
	}
	
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		result.ErrorMessage = "Directory does not exist"
		logger.Error("Directory does not exist: %s", repoPath)
		return result
	}
	
	if !IsGitRepo(repoPath) {
		result.ErrorMessage = "Not a Git repository"
		logger.Warning("Not a Git repository")
		return result
	}
	
	if !HasOriginRemote(repoPath) {
		result.ErrorMessage = "No origin remote"
		logger.Warning("No origin remote")
		return result
	}
	
	branch, err := DetectDefaultBranch(repoPath, defaultBranches)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to detect default branch: %v", err)
		logger.Error("Failed to detect default branch: %v", err)
		return result
	}
	result.Branch = branch
	
	startTime := time.Now()
	if err := CheckoutBranch(repoPath, branch); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to checkout branch %s: %v", branch, err)
		logger.Error("Failed to checkout branch %s: %v", branch, err)
		return result
	}
	logger.Debug("Checked out branch %s in %v", branch, time.Since(startTime))
	
	pullStart := time.Now()
	if err := Pull(repoPath); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to pull: %v", err)
		logger.Error("Failed to pull: %v", err)
		return result
	}
	
	logger.Success("Pulled %s in %v", branch, time.Since(pullStart))
	result.Success = true
	return result
}
