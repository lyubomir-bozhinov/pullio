package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lyubomir-bozhinov/pullio/internal/logger"
)

// FileSystem interface allows mocking filesystem operations for testing
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// RealFileSystem implements FileSystem with real OS operations
type RealFileSystem struct{}

func (RealFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (RealFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// Default filesystem implementation
var filesystem FileSystem = RealFileSystem{}

// SetFileSystem allows setting a mock filesystem for testing
func SetFileSystem(fs FileSystem) {
	filesystem = fs
}

// FindGitDirs finds all .git directories starting from the given root path
func FindGitDirs(root string) ([]string, error) {
	// Standardize the root path
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", root, err)
	}
	
	logger.Debug("Searching for Git repositories in %s", root)
	
	var gitDirs []string
	var mu sync.Mutex // Mutex to protect concurrent access to gitDirs
	var searchErr error
	
	// Check if the provided path is a Git repository itself
	gitDir := filepath.Join(root, ".git")
	info, err := filesystem.Stat(gitDir)
	if err == nil && info.IsDir() {
		logger.Debug("Found root directory is a Git repository: %s", root)
		return []string{gitDir}, nil
	}
	
	// Walk the directory tree to find .git directories
	err = filesystem.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Check for errors accessing the path
		if err != nil {
			logger.Debug("Error accessing path %s: %v", path, err)
			return filepath.SkipDir
		}
		
		// Skip directories that typically don't contain Git repositories
		if d.IsDir() {
			name := d.Name()
			
			// Skip common directories that don't contain Git repositories
			if name == "node_modules" || name == ".git" || 
			   strings.HasPrefix(name, ".") || 
			   name == "vendor" || name == "dist" || 
			   name == "build" || name == "target" {
				return filepath.SkipDir
			}
			
			// Check if current directory is a Git repository
			gitPath := filepath.Join(path, ".git")
			info, err := filesystem.Stat(gitPath)
			if err == nil && info.IsDir() {
				mu.Lock()
				gitDirs = append(gitDirs, gitPath)
				mu.Unlock()
				logger.Debug("Found Git repository: %s", path)
				
				// Skip scanning inside this directory as it's a Git repository
				return filepath.SkipDir
			}
		}
		
		return nil
	})
	
	if err != nil {
		searchErr = fmt.Errorf("error walking directory %s: %w", root, err)
	}
	
	if searchErr != nil {
		return nil, searchErr
	}
	
	return gitDirs, nil
}
