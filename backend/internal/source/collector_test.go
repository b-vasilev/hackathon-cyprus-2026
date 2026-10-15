package source

import (
	"os"
	"os/exec"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandEnvStripsConnectionMasterKey(t *testing.T) {
	t.Setenv("CONNECTION_MASTER_KEY", "test-master-key-must-not-reach-packager")
	collector := &Collector{}
	for _, entry := range collector.commandEnv("/work") {
		if strings.HasPrefix(entry, "CONNECTION_MASTER_KEY=") {
			t.Fatal("connection master key passed to source packager")
		}
	}
}

func TestCollectionPreservesOriginalGitBytes(t *testing.T) {
	binary, resolveErr := filepath.Abs("../../../frontend/node_modules/.bin/repomix")
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	if _, checkErr := os.Stat(binary); checkErr != nil {
		t.Skip("install frontend dependencies to verify the Repomix integration")
	}
	dir := t.TempDir()
	// Production workspaces live under backend/data, which the host repository ignores.
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("selected/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	noGitPath := filepath.Join(dir, "bin")
	if checkErr := os.Mkdir(noGitPath, 0o700); checkErr != nil {
		t.Fatal(checkErr)
	}
	if checkErr := os.Symlink(node, filepath.Join(noGitPath, "node")); checkErr != nil {
		t.Fatal(checkErr)
	}
	t.Setenv("PATH", noGitPath)
	repo := filepath.Join(dir, "repo")
	if checkErr := os.Mkdir(repo, 0o700); checkErr != nil {
		t.Fatal(checkErr)
	}
	content := "\n\n# Product\n\nA source-backed product example.\n\n"
	if checkErr := os.WriteFile(filepath.Join(repo, "README.md"), []byte(content), 0o600); checkErr != nil {
		t.Fatal(checkErr)
	}
	collector := Collector{WorkDir: dir, RepomixBin: binary, MaxBytes: 100000, MaxFiles: 10}
	repository, err := git.PlainInit(repo, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := worktree.Add("README.md"); checkErr != nil {
		t.Fatal(checkErr)
	}
	commit, err := worktree.Commit("source fixture", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Unix(1, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	// Read the original commit even after the branch advances.
	if checkErr := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed"), 0o600); checkErr != nil {
		t.Fatal(checkErr)
	}
	if _, checkErr := worktree.Add("README.md"); checkErr != nil {
		t.Fatal(checkErr)
	}
	if _, checkErr := worktree.Commit("advance branch", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Unix(2, 0)}}); checkErr != nil {
		t.Fatal(checkErr)
	}
	packed, err := collector.packRepository(t.Context(), dir, repository, commit.String(), []string{"README.md"}, collector.commandEnv(dir), filepath.Join(dir, "source.json"), filepath.Join(dir, "repomix.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if packed["README.md"] != content {
		t.Fatalf("original source bytes changed: %q", packed["README.md"])
	}
}

func TestSourceBoundaries(t *testing.T) {
	t.Parallel()
	for _, input := range []Input{{Kind: "github", Locator: "https://evil.example/repo/project"}, {Kind: "github", Locator: "https://github.com/owner/repo/tree/main"}, {Kind: "text", Text: ""}} {
		if checkErr := Validate([]Input{input}); checkErr == nil {
			t.Fatal("invalid source accepted")
		}
	}
	for _, file := range []treeFile{{Path: ".env", Type: "blob", Mode: "100644", Size: 10}, {Path: "src/link", Type: "blob", Mode: "120000", Size: 10}, {Path: "vendor/dependency.go", Type: "blob", Mode: "100644", Size: 10}, {Path: "credentials.json", Type: "blob", Mode: "100644", Size: 10}} {
		if eligibleFile(file) {
			t.Fatal("unsafe source candidate accepted", file.Path)
		}
	}
	collector := Collector{MaxBytes: 30, MaxFiles: 2}
	collection, err := collector.Collect(t.Context(), []Input{{Kind: "text", Text: "Purpose\nSecond line"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Documents) != 1 || collection.Documents[0].LineCount != 2 || !strings.Contains(Context(collection), "2: Second line") {
		t.Fatal("source lines are not stable")
	}
	if _, checkErr := collector.Collect(t.Context(), []Input{{Kind: "text", Text: strings.Repeat("x", 31)}}); checkErr == nil {
		t.Fatal("source budget not enforced")
	}
}
