// Package source collects bounded, immutable source material for passport generation.
package source

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	git "github.com/go-git/go-git/v5"
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
	FileCount  int            `json:"file_count"`
	Bytes      int            `json:"bytes"`
	Selection  string         `json:"selection"`
	Note       string         `json:"note"`
	ReadFiles  int            `json:"read_files"`
	CitedFiles int            `json:"cited_files"`
	Files      []FileCoverage `json:"files"`
}

// LineRange records original source lines actually returned to the researcher.
type LineRange struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}

// FileCoverage distinguishes indexed files from files read and cited by the model.
type FileCoverage struct {
	ID         string      `json:"id"`
	Path       string      `json:"path"`
	Bytes      int         `json:"bytes"`
	ReadRanges []LineRange `json:"read_ranges"`
	Cited      bool        `json:"cited"`
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
	Token                            func(context.Context) (string, error)
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
	if c.Token != nil && slices.ContainsFunc(inputs, func(input Input) bool { return input.Kind == "github" }) {
		token, err := c.Token(ctx)
		if err != nil {
			return Collection{}, fmt.Errorf("load GitHub connection: %w", err)
		}
		if token == "" {
			token = c.GitHubToken
		}
		ctx = context.WithValue(ctx, githubTokenKey{}, token)
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
			if len(content) > 8*1024*1024 {
				return Collection{}, errors.New("a source file exceeds the 8 MiB safety ceiling")
			}
			bytes += len(content)
			if bytes > c.MaxBytes || len(result.Documents) >= c.MaxFiles {
				return Collection{}, errors.New("source context exceeds the configured budget; narrow the source include patterns")
			}
			hash := sha256.Sum256([]byte(content))
			result.Documents = append(result.Documents, Document{ID: rand.Text(), SnapshotID: snapshot.ID, Path: path, ContentHash: hex.EncodeToString(hash[:]), Content: content, LineCount: strings.Count(content, "\n") + 1})
			snapshot.Coverage.FileCount++
			snapshot.Coverage.Bytes += len(content)
		}
		snapshot.Coverage.Selection = input.Include
		snapshot.Coverage.Note = "Repository text indexed at the pinned SHA. Build outputs, dependencies, binary files and secret-like files are excluded. Available files are not automatically read by the model."
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
	key := fmt.Sprintf("product-brand-v1|%s|%s|%s|%s|%d|%d", c.CacheScope, input.Locator, sha, input.Include, budget, c.MaxFiles)
	c.cacheMu.Lock()
	files, cached := c.cache[key]
	c.cacheMu.Unlock()
	if !cached {
		files, cached = c.loadPackage(key)
	}
	if cached {
		return files, true, nil
	}
	selected, selectErr := c.selectPaths(ctx, input, sha, choose, budget)
	if selectErr != nil {
		return nil, false, selectErr
	}
	packed, packErr := c.pack(ctx, input, sha, selected)
	files = packed
	if packErr != nil {
		return nil, false, packErr
	}
	if err := c.savePackage(key, files); err != nil {
		return nil, false, err
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
	if token := c.githubToken(ctx); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
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
	repository, err := fetchGitRepository(ctx, filepath.Join(dir, "git"), input.Locator, sha, c.githubToken(ctx))
	if err != nil {
		return nil, err
	}
	return c.packRepository(ctx, dir, repository, sha, selected, env, output, configPath)
}

func (c *Collector) packRepository(ctx context.Context, dir string, repository *git.Repository, sha string, selected, env []string, output, configPath string) (map[string]string, error) {
	selectedDir := filepath.Join(dir, "selected")
	if checkErr := os.Mkdir(selectedDir, 0o700); checkErr != nil {
		return nil, fmt.Errorf("create selected source workspace: %w", checkErr)
	}
	originals, readErr := readGitFiles(ctx, repository, sha, selected, c.MaxBytes)
	if readErr != nil {
		return nil, readErr
	}
	for path, content := range originals {
		target := filepath.Join(selectedDir, path)
		if checkErr := os.MkdirAll(filepath.Dir(target), 0o700); checkErr != nil {
			return nil, fmt.Errorf("create source directory: %w", checkErr)
		}
		if checkErr := os.WriteFile(target, []byte(content), 0o600); checkErr != nil {
			return nil, fmt.Errorf("write source evidence: %w", checkErr)
		}
	}
	// Files were selected from the pinned Git tree. The host repository's ignore
	// rules must not hide this staging directory (for example, backend/data).
	config := map[string]any{"output": map[string]any{"filePath": output, "style": "json", "removeComments": false, "removeEmptyLines": false, "showLineNumbers": false, "compress": false, "fileSummary": false}, "security": map[string]any{"enableSecurityCheck": true}, "ignore": map[string]any{"useGitignore": false, "useDefaultPatterns": true, "customPatterns": []string{"**/.env*", "**/*.pem", "**/*.key", "**/credentials*", "**/package-lock.json", "**/go.sum", "**/pnpm-lock.yaml", "**/yarn.lock", "**/*.min.js", "**/*.lock"}}}
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
	args := []string{selectedDir, "--config", configPath, "--output", output, "--style", "json", "--quiet", "--no-git-sort-by-changes"}
	//nolint:gosec // Executable is operator-configured; arguments are fixed or app-owned local paths.
	command := exec.CommandContext(ctx, c.RepomixBin, args...)
	command.Dir = dir
	command.Env = env
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if checkErr := command.Run(); checkErr != nil {
		return nil, errors.New("could not prepare source files for this revision. Check repository access, source reader availability and source size limits")
	}
	//nolint:gosec // Output is an app-generated file within a fresh private workspace.
	file, err := os.Open(output)
	if err != nil {
		return nil, errors.New("the source reader did not produce a source package")
	}
	defer file.Close()
	var packed struct {
		Files map[string]string `json:"files"`
	}
	if checkErr := json.NewDecoder(io.LimitReader(file, int64(c.MaxBytes)*3+(1<<20))).Decode(&packed); checkErr != nil {
		return nil, errors.New("the source package is invalid or exceeds the collection budget")
	}
	// Repomix trims surrounding whitespace. Use its safe-file selection, but retain
	// the original Git bytes so citations and content hashes never shift lines.
	safe := make(map[string]string, len(packed.Files))
	for path := range packed.Files {
		if content, ok := originals[path]; ok {
			safe[path] = content
		}
	}
	if len(safe) == 0 {
		return nil, errors.New("no selected source files remained after source filtering")
	}
	return safe, nil
}

func (c *Collector) commandEnv(dir string) []string {
	env := make([]string, 0)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "GIT_") || slices.Contains([]string{"TMPDIR", "CONNECTION_MASTER_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GITHUB_TOKEN", "GITHUB_OAUTH_CLIENT_SECRET"}, name) {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "TMPDIR="+dir)
	return env
}

type treeFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Mode string `json:"mode"`
	Size int    `json:"size"`
}

func (c *Collector) repositoryTree(ctx context.Context, locator, sha string) ([]treeFile, error) {
	u, _ := url.Parse(locator)
	repo := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/git/trees/"+sha+"?recursive=1", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build source catalog request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if token := c.githubToken(ctx); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
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
	return tree.Tree, nil
}

func (c *Collector) selectPaths(ctx context.Context, input Input, sha string, choose SelectFiles, budget int) ([]string, error) {
	tree, err := c.repositoryTree(ctx, input.Locator, sha)
	if err != nil {
		return nil, err
	}
	catalog, err := c.buildSourceCatalog(input, tree)
	if err != nil {
		return nil, err
	}
	eligible, aliases := catalog.files, catalog.aliases
	if choose == nil {
		paths := make([]string, 0, len(eligible))
		for path := range eligible {
			paths = append(paths, path)
		}
		slices.Sort(paths)
		return paths, nil
	}
	chosen, err := choose(ctx, fmt.Sprintf("Repository: %s\nCommit: %s\nFile budget: at most %d files and %d bytes. Catalog limit: 5000 eligible entries.\nCATALOG:\n%s", input.Locator, sha, min(c.MaxFiles, 80), budget, catalog.text))
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	seen := make(map[string]bool)
	// Reserve a small part of the selection for identity files. Otherwise the
	// technical file selector can omit the brand guide and original SVG logo.
	brandPaths := make([]string, 0)
	for path, size := range eligible {
		if IsBrandFile(path) && size <= 100000 {
			brandPaths = append(brandPaths, path)
		}
	}
	slices.Sort(brandPaths)
	brandBytes := 0
	for _, path := range brandPaths {
		if len(paths) >= min(12, c.MaxFiles/4) || brandBytes+eligible[path] > budget/4 {
			continue
		}
		paths = append(paths, path)
		seen[path] = true
		brandBytes += eligible[path]
	}
	bytes := brandBytes
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
	if file.Type != "blob" || file.Mode == "120000" || file.Size < 1 || file.Size > 8<<20 {
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
		if slices.Contains([]string{"node_modules", "vendor", ".git", "dist", "build", "bin", "obj", "coverage", "__pycache__", "artifacts"}, part) {
			return false
		}
	}
	if filepath.Ext(path) == ".svg" {
		return IsBrandFile(path) && file.Size <= 100000
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
		n := 0
		for line := range strings.SplitSeq(doc.Content, "\n") {
			n++
			fmt.Fprintf(&b, "%d: %s\n", n, line)
		}
		b.WriteString("</source_document>\n")
	}
	return b.String()
}

type repositoryCatalog struct {
	files   map[string]int
	aliases map[string]string
	text    string
}

func (c *Collector) buildSourceCatalog(input Input, tree []treeFile) (repositoryCatalog, error) {
	eligible := make(map[string]int)
	aliases := make(map[string]string)
	lastDirectory := ""
	var catalog strings.Builder
	for _, file := range tree {
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
				return repositoryCatalog{}, errors.New("invalid source include pattern")
			}
			matches = matches || match
		}
		if !matches {
			continue
		}
		if len(eligible) >= c.MaxFiles {
			return repositoryCatalog{}, errors.New("repository exceeds the configured file safety ceiling; no partial catalog was used")
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
		return repositoryCatalog{}, errors.New("no eligible text/source files match this repository selection")
	}
	return repositoryCatalog{files: eligible, aliases: aliases, text: catalog.String()}, nil
}
