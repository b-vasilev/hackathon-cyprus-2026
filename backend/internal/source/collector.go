// Package source collects bounded, immutable source material for passport generation.
package source

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// Input is either a GitHub repository or supplemental product context.
type Input struct {
	Kind    string `json:"kind"`
	Locator string `json:"locator"`
	Ref     string `json:"ref"`
	Include string `json:"include"`
	Text    string `json:"text"`
}

// Snapshot records exactly which revision was analyzed and what was collected.
type Snapshot struct {
	ID       string   `json:"id"`
	Locator  string   `json:"locator"`
	Revision string   `json:"revision"`
	Coverage Coverage `json:"coverage"`
}

// Coverage describes collection limits without claiming complete repository analysis.
type Coverage struct {
	FileCount int    `json:"file_count"`
	Bytes     int    `json:"bytes"`
	Selection string `json:"selection"`
	Note      string `json:"note"`
}

// Document preserves the original collected text and its stable line numbering.
type Document struct {
	ID          string `json:"id"`
	SnapshotID  string `json:"snapshot_id"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	Content     string `json:"content"`
	LineCount   int    `json:"line_count"`
}

// Collection is a bounded set of source evidence.
type Collection struct {
	Snapshots []Snapshot
	Documents []Document
}

// FileSelection is a bounded LLM selection from a verified repository file catalog.
type FileSelection struct {
	Paths []string `json:"paths" jsonschema:"File IDs (such as f12) or exact paths from the supplied catalog, ordered by importance."`
}

// SelectFiles chooses useful files; only verified catalog entries can be packaged.
type SelectFiles func(context.Context, string) ([]string, error)

// Collector invokes a pinned Repomix executable; repository code is never executed.
type Collector struct {
	WorkDir, RepomixBin, GitHubToken string
	MaxBytes, MaxFiles               int
	CacheScope                       string
	cacheMu                          sync.Mutex
	cache                            map[string]map[string]string
}

var commitPattern = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

// Validate checks source input before queueing any work.
func Validate(inputs []Input) error {
	if len(inputs) == 0 || len(inputs) > 5 {
		return errors.New("provide 1–5 sources")
	}
	for _, s := range inputs {
		switch s.Kind {
		case "github":
			u, err := url.Parse(s.Locator)
			if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(strings.Split(strings.Trim(u.Path, "/"), "/")) != 2 {
				return errors.New("repository sources must be https://github.com/owner/repository URLs")
			}
			if len(s.Ref) > 200 || strings.ContainsAny(s.Ref, "\r\n") || len(s.Include) > 2000 {
				return errors.New("invalid source ref or file selection")
			}
		case "text":
			if strings.TrimSpace(s.Text) == "" || len(s.Text) > 100000 {
				return errors.New("context sources must contain 1–100000 characters")
			}
		default:
			return errors.New("source kind must be github or text")
		}
	}
	return nil
}

// Collect resolves every repository to a SHA before asking Repomix to package it.
func (c *Collector) Collect(ctx context.Context, inputs []Input) (Collection, error) {
	return c.CollectSelected(ctx, inputs, nil)
}

// CollectSelected adds automatic file selection while retaining deterministic source anchors.
func (c *Collector) CollectSelected(ctx context.Context, inputs []Input, choose SelectFiles) (Collection, error) {
	if checkErr := Validate(inputs); checkErr != nil {
		return Collection{}, checkErr
	}
	result := Collection{Snapshots: make([]Snapshot, 0), Documents: make([]Document, 0)}
	bytes := 0
	for _, input := range inputs {
		snapshot := Snapshot{ID: rand.Text(), Locator: input.Locator}
		files := map[string]string{}
		cached := false
		if input.Kind == "text" {
			snapshot.Locator = "Supplemental context"
			hash := sha256.Sum256([]byte(input.Text))
			snapshot.Revision = hex.EncodeToString(hash[:])
			files["product-context.txt"] = input.Text
		} else {
			sha, err := c.resolve(ctx, input)
			if err != nil {
				return Collection{}, err
			}
			snapshot.Revision = sha
			budget := max(1, c.MaxBytes/len(inputs)*8/10)
			files, cached, err = c.repositoryFiles(ctx, input, sha, choose, budget)
			if err != nil {
				return Collection{}, err
			}
		}
		paths := make([]string, 0, len(files))
		for path := range files {
			paths = append(paths, path)
		}
		slices.Sort(paths)
		for _, path := range paths {
			content := files[path]
			if content == "" {
				continue
			}
			if len(content) > 128*1024 {
				return Collection{}, errors.New("a selected source file exceeds 128 KiB; narrow the source include patterns")
			}
			bytes += len(content)
			if bytes > c.MaxBytes || len(result.Documents) >= c.MaxFiles {
				return Collection{}, errors.New("source context exceeds the configured budget; narrow the source include patterns")
			}
			hash := sha256.Sum256([]byte(content))
			result.Documents = append(result.Documents, Document{ID: rand.Text(), SnapshotID: snapshot.ID, Path: path, ContentHash: hex.EncodeToString(hash[:]), Content: content, LineCount: len(strings.Split(content, "\n"))})
			snapshot.Coverage.FileCount++
			snapshot.Coverage.Bytes += len(content)
		}
		snapshot.Coverage.Selection = input.Include
		snapshot.Coverage.Note = "Selected text files only. Build outputs, dependencies, binary files and secret-like files are excluded. Missing material is not evidence of absent behavior."
		if cached {
			snapshot.Coverage.Note += " Reused the safe source package at the unchanged commit SHA."
		}
		result.Snapshots = append(result.Snapshots, snapshot)
	}
	if len(result.Documents) == 0 {
		return Collection{}, errors.New("no usable source documents were collected")
	}
	return result, nil
}

func (c *Collector) repositoryFiles(ctx context.Context, input Input, sha string, choose SelectFiles, budget int) (map[string]string, bool, error) {
	key := fmt.Sprintf("%s|%s|%s|%s|%d|%d", c.CacheScope, input.Locator, sha, input.Include, budget, c.MaxFiles)
	c.cacheMu.Lock()
	files, cached := c.cache[key]
	c.cacheMu.Unlock()
	if !cached {
		selected, selectErr := c.selectPaths(ctx, input, sha, choose, budget)
		if selectErr != nil {
			return nil, false, selectErr
		}
		packed, packErr := c.pack(ctx, input, sha, selected)
		files = packed
		if packErr != nil {
			return nil, false, packErr
		}
		c.cacheMu.Lock()
		if len(c.cache) >= 8 {
			clear(c.cache)
		}
		if c.cache == nil {
			c.cache = make(map[string]map[string]string)
		}
		c.cache[key] = files
		c.cacheMu.Unlock()
	}
	return files, cached, nil
}

func (c *Collector) resolve(ctx context.Context, input Input) (string, error) {
	if commitPattern.MatchString(input.Ref) {
		return strings.ToLower(input.Ref), nil
	}
	u, _ := url.Parse(input.Locator)
	repo := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	ref := input.Ref
	if ref == "" {
		ref = "HEAD"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/commits/"+url.PathEscape(ref), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build GitHub request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if c.GitHubToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.GitHubToken)
	}
	client := http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("could not reach GitHub to resolve the source revision")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub could not resolve the repository/ref (HTTP %d); check source access and GITHUB_TOKEN", response.StatusCode)
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if checkErr := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&commit); checkErr != nil || !commitPattern.MatchString(commit.SHA) {
		return "", errors.New("GitHub returned an invalid commit revision")
	}
	return commit.SHA, nil
}

//nolint:gocyclo // Keep collection, safe staging and original-byte preservation in one bounded operation.
func (c *Collector) pack(ctx context.Context, input Input, sha string, selected []string) (map[string]string, error) {
	if checkErr := os.MkdirAll(c.WorkDir, 0o700); checkErr != nil {
		return nil, fmt.Errorf("create collection workspace: %w", checkErr)
	}
	dir, workspaceErr := os.MkdirTemp(c.WorkDir, "source-")
	if workspaceErr != nil {
		return nil, fmt.Errorf("create source workspace: %w", workspaceErr)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	output := filepath.Join(dir, "source.json")
	configPath := filepath.Join(dir, "repomix.config.json")
	env := c.commandEnv(dir)
	repository := filepath.Join(dir, "git")
	if checkErr := os.Mkdir(repository, 0o700); checkErr != nil {
		return nil, fmt.Errorf("create Git source workspace: %w", checkErr)
	}
	for _, args := range [][]string{
		{"init", "--quiet", repository},
		{"-C", repository, "remote", "add", "origin", input.Locator},
		{"-C", repository, "config", "remote.origin.promisor", "true"},
		{"-C", repository, "config", "remote.origin.partialclonefilter", "blob:none"},
		{"-C", repository, "fetch", "--quiet", "--filter=blob:none", "--depth", "1", "--recurse-submodules=no", "origin", sha},
	} {
		//nolint:gosec // Fixed executable and argument list; no shell or repository code is executed.
		command := exec.CommandContext(ctx, "git", args...)
		command.Env = env
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if checkErr := command.Run(); checkErr != nil {
			return nil, errors.New("could not fetch the pinned source revision; check repository access")
		}
	}
	selectedDir := filepath.Join(dir, "selected")
	if checkErr := os.Mkdir(selectedDir, 0o700); checkErr != nil {
		return nil, fmt.Errorf("create selected source workspace: %w", checkErr)
	}
	originals := make(map[string]string, len(selected))
	for _, path := range selected {
		if !filepath.IsLocal(path) {
			return nil, errors.New("source catalog contains an invalid path")
		}
		//nolint:gosec // SHA and exact relative path come from the verified GitHub catalog.
		command := exec.CommandContext(ctx, "git", "-C", repository, "show", sha+":"+path)
		command.Env = env
		content, err := command.Output()
		if err != nil {
			return nil, errors.New("could not read an original file at the pinned revision")
		}
		if len(content) > 128*1024 {
			return nil, errors.New("selected source file exceeds the file budget")
		}
		target := filepath.Join(selectedDir, path)
		if checkErr := os.MkdirAll(filepath.Dir(target), 0o700); checkErr != nil {
			return nil, fmt.Errorf("create source directory: %w", checkErr)
		}
		if checkErr := os.WriteFile(target, content, 0o600); checkErr != nil {
			return nil, fmt.Errorf("write source evidence: %w", checkErr)
		}
		originals[path] = string(content)
	}
	config := map[string]any{"output": map[string]any{"filePath": output, "style": "json", "removeComments": false, "removeEmptyLines": false, "showLineNumbers": false, "compress": false, "fileSummary": false}, "security": map[string]any{"enableSecurityCheck": true}, "ignore": map[string]any{"useGitignore": true, "useDefaultPatterns": true, "customPatterns": []string{"**/.env*", "**/*.pem", "**/*.key", "**/credentials*", "**/package-lock.json", "**/go.sum", "**/pnpm-lock.yaml", "**/yarn.lock", "**/*.min.js", "**/*.svg", "**/*.lock"}}}
	patterns := make([]string, 0, len(selected))
	escape := strings.NewReplacer("\\", "\\\\", "*", "\\*", "?", "\\?", "[", "\\[", "]", "\\]", "{", "\\{", "}", "\\}", "!", "\\!")
	for _, path := range selected {
		patterns = append(patterns, escape.Replace(path))
	}
	config["include"] = patterns
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("encode collection settings: %w", err)
	}
	if checkErr := os.WriteFile(configPath, encoded, 0o600); checkErr != nil {
		return nil, fmt.Errorf("write collection settings: %w", checkErr)
	}
	args := []string{selectedDir, "--config", configPath, "--output", output, "--style", "json", "--quiet", "--no-git-sort-by-changes", "--token-budget", "100000"}
	//nolint:gosec // Executable is operator-configured; arguments are fixed or app-owned local paths.
	command := exec.CommandContext(ctx, c.RepomixBin, args...)
	command.Dir = dir
	command.Env = env
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if checkErr := command.Run(); checkErr != nil {
		return nil, errors.New("repomix could not package this revision. Check repository access, the Repomix executable and source size limits")
	}
	//nolint:gosec // Output is an app-generated file within a fresh private workspace.
	file, err := os.Open(output)
	if err != nil {
		return nil, errors.New("repomix did not produce a source package")
	}
	defer file.Close()
	var packed struct {
		Files map[string]string `json:"files"`
	}
	if checkErr := json.NewDecoder(io.LimitReader(file, int64(c.MaxBytes)+(1<<20))).Decode(&packed); checkErr != nil {
		return nil, errors.New("repomix source package is invalid or exceeds the collection budget")
	}
	// Repomix trims surrounding whitespace. Use its safe-file selection, but retain
	// the original Git bytes so citations and content hashes never shift lines.
	safe := make(map[string]string, len(packed.Files))
	for path := range packed.Files {
		if content, ok := originals[path]; ok {
			safe[path] = content
		}
	}
	return safe, nil
}

func (c *Collector) commandEnv(dir string) []string {
	env := make([]string, 0)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "GIT_") || slices.Contains([]string{"TMPDIR", "LLM_MASTER_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GITHUB_TOKEN"}, name) {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "TMPDIR="+dir, "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=core.hooksPath", "GIT_CONFIG_VALUE_0=/dev/null")
	if c.GitHubToken != "" {
		env[len(env)-3] = "GIT_CONFIG_COUNT=2"
		env = append(env, "GIT_CONFIG_KEY_1=http.https://github.com/.extraheader", "GIT_CONFIG_VALUE_1=Authorization: Basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:"+c.GitHubToken)))
	}
	return env
}

type treeFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Mode string `json:"mode"`
	Size int    `json:"size"`
}

//nolint:gocyclo // File eligibility, user selection and model selection share one catalog boundary.
func (c *Collector) selectPaths(ctx context.Context, input Input, sha string, choose SelectFiles, budget int) ([]string, error) {
	u, _ := url.Parse(input.Locator)
	repo := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/git/trees/"+sha+"?recursive=1", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build source catalog request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if c.GitHubToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.GitHubToken)
	}
	client := http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("could not retrieve the repository file catalog")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub file catalog request failed (HTTP %d)", response.StatusCode)
	}
	var tree struct {
		Tree      []treeFile `json:"tree"`
		Truncated bool       `json:"truncated"`
	}
	if checkErr := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&tree); checkErr != nil {
		return nil, errors.New("repository catalog is invalid or too large")
	}
	if tree.Truncated {
		return nil, errors.New("GitHub returned an incomplete file catalog; use a smaller repository source")
	}
	eligible := make(map[string]int)
	aliases := make(map[string]string)
	lastDirectory := ""
	var catalog strings.Builder
	for _, file := range tree.Tree {
		if !eligibleFile(file) {
			continue
		}
		matches := input.Include == ""
		for pattern := range strings.SplitSeq(input.Include, ",") {
			if pattern == "" {
				continue
			}
			match, matchErr := doublestar.Match(strings.TrimSpace(pattern), file.Path)
			if matchErr != nil {
				return nil, errors.New("invalid source include pattern")
			}
			matches = matches || match
		}
		if !matches {
			continue
		}
		if len(eligible) >= 5000 {
			break
		}
		eligible[file.Path] = file.Size
		id := fmt.Sprintf("f%d", len(eligible))
		aliases[id] = file.Path
		directory := filepath.Dir(file.Path)
		if directory != lastDirectory {
			fmt.Fprintf(&catalog, "\nDirectory: %s\n", directory)
			lastDirectory = directory
		}
		fmt.Fprintf(&catalog, "%s %s %dB\n", id, filepath.Base(file.Path), file.Size)
	}
	if len(eligible) == 0 {
		return nil, errors.New("no eligible text/source files match this repository selection")
	}
	if choose == nil {
		return nil, errors.New("automatic source selection requires a configured model")
	}
	chosen, err := choose(ctx, fmt.Sprintf("Repository: %s\nCommit: %s\nFile budget: at most %d files and %d bytes. Catalog limit: 5000 eligible entries.\nCATALOG:\n%s", input.Locator, sha, min(c.MaxFiles, 80), budget, catalog.String()))
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	seen := make(map[string]bool)
	bytes := 0
	for _, path := range chosen {
		if resolved, ok := aliases[path]; ok {
			path = resolved
		}
		size, ok := eligible[path]
		if !ok {
			return nil, errors.New("the model selected a file outside the verified source catalog")
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		if len(paths) >= min(c.MaxFiles, 80) || bytes+size > budget {
			continue
		}
		paths = append(paths, path)
		bytes += size
	}
	if len(paths) == 0 {
		return nil, errors.New("the model did not select any source files within the context budget")
	}
	return paths, nil
}

func eligibleFile(file treeFile) bool {
	if file.Type != "blob" || file.Mode == "120000" || file.Size < 1 || file.Size > 128*1024 {
		return false
	}
	path := strings.ToLower(file.Path)
	name := filepath.Base(path)
	if strings.HasPrefix(name, ".env") || strings.HasPrefix(name, "credentials") || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".key") || strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".min.js") {
		return false
	}
	if slices.Contains([]string{"go.sum", "package-lock.json", "pnpm-lock.yaml", "yarn.lock"}, name) {
		return false
	}
	for part := range strings.SplitSeq(path, "/") {
		if slices.Contains([]string{"node_modules", "vendor", ".git", "dist", "build", "bin", "obj", "coverage", "__pycache__", "artifacts", "testdata", "fixtures", "__snapshots__"}, part) {
			return false
		}
	}
	return !slices.Contains([]string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".zip", ".pdf", ".woff", ".woff2", ".ttf", ".mp4", ".parquet", ".sqlite", ".db"}, filepath.Ext(path))
}

// Context formats source text with stable IDs and line numbers; it is always untrusted data.
func Context(collection Collection) string {
	var b strings.Builder
	for _, snapshot := range collection.Snapshots {
		fmt.Fprintf(&b, "SOURCE %s | %s | revision %s\nCoverage: %s\n", snapshot.ID, snapshot.Locator, snapshot.Revision, snapshot.Coverage.Note)
	}
	for _, doc := range collection.Documents {
		fmt.Fprintf(&b, "\n<source_document id=%q path=%q>\n", doc.ID, doc.Path)
		for i, line := range strings.Split(doc.Content, "\n") {
			fmt.Fprintf(&b, "%d: %s\n", i+1, line)
		}
		b.WriteString("</source_document>\n")
	}
	return b.String()
}
