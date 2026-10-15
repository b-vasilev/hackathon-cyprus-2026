package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGitBlobBoundaries(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	repository, err := git.PlainInit(directory, false)
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{"README.md": "source\n", "binary.bin": "a\x00b"} {
		if checkErr := os.WriteFile(filepath.Join(directory, path), []byte(content), 0o600); checkErr != nil {
			t.Fatal(checkErr)
		}
	}
	if checkErr := os.Symlink("README.md", filepath.Join(directory, "link")); checkErr != nil {
		t.Fatal(checkErr)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if checkErr := worktree.AddWithOptions(&git.AddOptions{All: true}); checkErr != nil {
		t.Fatal(checkErr)
	}
	commit, err := worktree.Commit("source", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Unix(1, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		paths   []string
		budget  int
		invalid bool
	}{
		{"regular and binary", []string{"README.md", "binary.bin"}, 20, false},
		{"budget", []string{"README.md"}, 1, true},
		{"symlink", []string{"link"}, 20, true},
		{"traversal", []string{"../README.md"}, 20, true},
		{"missing", []string{"missing"}, 20, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files, readErr := readGitFiles(t.Context(), repository, commit.String(), test.paths, test.budget)
			if (readErr != nil) != test.invalid {
				t.Fatalf("unexpected result: %v", readErr)
			}
			if !test.invalid && (files["README.md"] != "source\n" || len(files) != 1) {
				t.Fatal("binary was collected or text changed")
			}
		})
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, readErr := readGitFiles(canceled, repository, commit.String(), []string{"README.md"}, 20); readErr == nil {
		t.Fatal("canceled read completed")
	}
}

func TestFetchRejectsLocalGitTransports(t *testing.T) {
	t.Parallel()
	for _, locator := range []string{"/local/repo", "file:///local/repo", "ssh://git@github.com/owner/repo", "https://github.com/owner/repo?token=secret", "https://evil.example/owner/repo"} {
		if _, err := fetchGitRepository(t.Context(), t.TempDir(), locator, "1234567890123456789012345678901234567890", "credential"); err == nil {
			t.Fatal("unsafe transport accepted", locator)
		}
	}
}
