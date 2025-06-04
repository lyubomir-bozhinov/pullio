package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lyubomir-bozhinov/pullio/internal/gitmanager"
	"github.com/lyubomir-bozhinov/pullio/internal/logger"
	"github.com/lyubomir-bozhinov/pullio/internal/sshagent"
	"github.com/lyubomir-bozhinov/pullio/internal/utils"
)

var (
	sshKeyFlag     string
	branchesFlag   string
	concurrentFlag int
	verboseFlag    bool
	startPath      string
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}

	defaultSSHKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519")
	
	flag.StringVar(&sshKeyFlag, "key", defaultSSHKeyPath, "Path to the SSH private key")
	flag.StringVar(&branchesFlag, "branches", "main,master", "Comma-separated list of default branch names to try")
	flag.IntVar(&concurrentFlag, "concurrent", 4, "Number of repositories to process concurrently")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output")
	flag.StringVar(&startPath, "path", ".", "Starting path to search for repositories")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nUpdates all Git repositories under the specified path\n\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	
	logger.SetVerbose(verboseFlag)
	defaultBranches := strings.Split(branchesFlag, ",")
	
	logger.Info("Initializing SSH agent...")
	if err := sshagent.EnsureAgentAndKey(sshKeyFlag); err != nil {
		logger.Fatal("SSH Agent setup failed: %v", err)
	}
	
	logger.Info("Finding Git repositories from %s...", startPath)
	startTime := time.Now()
	gitDirs, err := utils.FindGitDirs(startPath)
	if err != nil {
		logger.Fatal("Failed to find Git directories: %v", err)
	}
	logger.Success("Found %d Git repositories in %v", len(gitDirs), time.Since(startTime))
	
	if len(gitDirs) == 0 {
		logger.Info("No Git repositories found. Exiting.")
		return
	}
	
	// Process repositories concurrently
	resultChan := make(chan gitmanager.RepoResult, len(gitDirs))
	sem := make(chan struct{}, concurrentFlag)
	
	var wg sync.WaitGroup
	for _, gitDir := range gitDirs {
		wg.Add(1)
		sem <- struct{}{}
		
		go func(dir string) {
			defer wg.Done()
			defer func() { <-sem }()
			
			repoPath := filepath.Dir(dir)
			result := gitmanager.ProcessRepository(repoPath, defaultBranches)
			resultChan <- result
		}(gitDir)
	}
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	var succeeded, failed []gitmanager.RepoResult
	for result := range resultChan {
		if result.Success {
			succeeded = append(succeeded, result)
		} else {
			failed = append(failed, result)
		}
	}
	
	// Print summary
	fmt.Printf("\n📦 Done. %d updated, %d failed.\n", len(succeeded), len(failed))
	
	if len(succeeded) > 0 {
		fmt.Println("\nSuccessfully updated repositories:")
		for _, r := range succeeded {
			fmt.Printf("✅ %s (branch: %s)\n", r.Path, r.Branch)
		}
	}
	
	if len(failed) > 0 {
		fmt.Println("\nFailed repositories:")
		for _, r := range failed {
			fmt.Printf("❌ %s (reason: %s)\n", r.Path, r.ErrorMessage)
		}
	}
}
