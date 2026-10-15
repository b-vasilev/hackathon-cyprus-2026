package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectionPreservesOriginalGitBytes(t *testing.T) {
	t.Parallel()
	binary, resolveErr := filepath.Abs("../../../frontend/node_modules/.bin/repomix")
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skip("install frontend dependencies to verify the Repomix integration")
	}
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "\n\n# Product\n\nA source-backed product example.\n\n"
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	collector := Collector{WorkDir: dir, RepomixBin: binary, MaxBytes: 100000, MaxFiles: 10}
	git := func(args ...string) string {
		t.Helper()
		//nolint:gosec // Test-owned arguments build an isolated fixture repository.
		command := exec.CommandContext(t.Context(), "git", args...)
		command.Dir = repo
		command.Env = collector.commandEnv(dir)
		result, runErr := command.CombinedOutput()
		if runErr != nil {
			t.Fatalf("fixture git failed: %v %s", runErr, result)
		}
		return strings.TrimSpace(string(result))
	}
	git("init", "--quiet")
	git("add", "README.md")
	git("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "source fixture")
	sha := git("rev-parse", "HEAD")
	packed, err := collector.pack(t.Context(), Input{Kind: "github", Locator: repo}, sha, []string{"README.md"})
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
		if err := Validate([]Input{input}); err == nil {
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
	if _, err := collector.Collect(t.Context(), []Input{{Kind: "text", Text: strings.Repeat("x", 31)}}); err == nil {
		t.Fatal("source budget not enforced")
	}
}
