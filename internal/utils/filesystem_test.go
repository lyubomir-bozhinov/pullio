package utils

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"
)

// MockFileInfo implements os.FileInfo for testing
type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return m.size }
func (m MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m MockFileInfo) ModTime() time.Time { return m.modTime }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() interface{}   { return m.sys }

// MockDirEntry implements fs.DirEntry for testing
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m MockDirEntry) Name() string               { return m.name }
func (m MockDirEntry) IsDir() bool                { return m.isDir }
func (m MockDirEntry) Type() os.FileMode          { return os.ModeDir }
func (m MockDirEntry) Info() (os.FileInfo, error) { return MockFileInfo{name: m.name, isDir: m.isDir}, nil }

// MockFileSystem implements FileSystem for testing
type MockFileSystem struct {
	files map[string]MockFileInfo
	dirs  map[string][]MockDirEntry
}

func (m MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if info, ok := m.files[name]; ok {
		return info, nil
	}
	return nil, os.ErrNotExist
}

func (m MockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	entries, ok := m.dirs[root]
	if !ok {
		return errors.New("directory not found")
	}
	
	// First, call the function for the root itself
	rootEntry := MockDirEntry{name: ".", isDir: true}
	if err := fn(root, rootEntry, nil); err != nil {
		return err
	}
	
	// Then for each entry
	for _, entry := range entries {
		path := root + "/" + entry.Name()
		if err := fn(path, entry, nil); err != nil {
			if err == filepath.SkipDir {
				continue
			}
			return err
		}
		
		// If it's a directory, recurse
		if entry.IsDir() {
			if err := m.WalkDir(path, fn); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// TestFindGitDirs tests the FindGitDirs function
func TestFindGitDirs(t *testing.T) {
	// Setup mock filesystem
	mockFS := MockFileSystem{
		files: map[string]MockFileInfo{
			"/root/.git":                MockFileInfo{name: ".git", isDir: true},
			"/root/repo1/.git":          MockFileInfo{name: ".git", isDir: true},
			"/root/repo2/.git":          MockFileInfo{name: ".git", isDir: true},
			"/root/empty/file.txt":      MockFileInfo{name: "file.txt", isDir: false},
			"/root/node_modules/.git":   MockFileInfo{name: ".git", isDir: true},
			"/root/repo3/nested/.git":   MockFileInfo{name: ".git", isDir: true},
		},
		dirs: map[string][]MockDirEntry{
			"/root": {
				MockDirEntry{name: "repo1", isDir: true},
				MockDirEntry{name: "repo2", isDir: true},
				MockDirEntry{name: "empty", isDir: true},
				MockDirEntry{name: "node_modules", isDir: true},
				MockDirEntry{name: "repo3", isDir: true},
				MockDirEntry{name: ".git", isDir: true},
			},
			"/root/repo1": {
				MockDirEntry{name: ".git", isDir: true},
				MockDirEntry{name: "src", isDir: true},
			},
			"/root/repo2": {
				MockDirEntry{name: ".git", isDir: true},
			},
			"/root/empty": {
				MockDirEntry{name: "file.txt", isDir: false},
			},
			"/root/node_modules": {
				MockDirEntry{name: ".git", isDir: true},
			},
			"/root/repo3": {
				MockDirEntry{name: "nested", isDir: true},
			},
			"/root/repo3/nested": {
				MockDirEntry{name: ".git", isDir: true},
			},
		},
	}
	
	// Save original filesystem and restore it at the end
	origFS := filesystem
	defer func() { filesystem = origFS }()
	
	// Set mock filesystem
	filesystem = mockFS
	
	// Test FindGitDirs with mock filesystem
	gitDirs, err := FindGitDirs("/root")
	
	if err != nil {
		t.Fatalf("FindGitDirs returned error: %v", err)
	}
	
	// We expect to find the root's .git directory
	if len(gitDirs) != 1 {
		t.Fatalf("Expected 1 Git directory, got %d", len(gitDirs))
	}
	
	// The root directory itself is a Git repository, so only that should be returned
	if gitDirs[0] != "/root/.git" {
		t.Errorf("Expected to find /root/.git, got %s", gitDirs[0])
	}
	
	// Test with a non-Git root directory
	mockFS.files["/root/.git"] = MockFileInfo{name: ".git", isDir: false} // Make root not a Git repo
	
	gitDirs, err = FindGitDirs("/root")
	
	if err != nil {
		t.Fatalf("FindGitDirs returned error: %v", err)
	}
	
	// Now we expect to find other Git repositories
	expectedRepos := map[string]bool{
		"/root/repo1/.git": true,
		"/root/repo2/.git": true,
		"/root/repo3/nested/.git": true,
	}
	
	// node_modules should be skipped
	for _, dir := range gitDirs {
		if strings.Contains(dir, "node_modules") {
			t.Errorf("node_modules should be skipped, but found %s", dir)
		}
		
		if !expectedRepos[dir] {
			t.Errorf("Unexpected Git directory found: %s", dir)
		}
		delete(expectedRepos, dir)
	}
	
	if len(expectedRepos) > 0 {
		t.Errorf("Some expected Git directories were not found: %v", expectedRepos)
	}
}