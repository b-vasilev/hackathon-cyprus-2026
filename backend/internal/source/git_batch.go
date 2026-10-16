package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// fetchGitRepository stores only the pinned revision, without a worktree or hooks.
func fetchGitRepository(ctx context.Context, directory, locator, sha, token string) (*git.Repository, error) {
	u, err := url.Parse(locator)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(strings.Split(strings.Trim(u.Path, "/"), "/")) != 2 || !commitPattern.MatchString(sha) {
		return nil, errors.New("invalid GitHub repository or source revision")
	}
	repository, err := git.PlainInit(directory, true)
	if err != nil {
		return nil, fmt.Errorf("create Git object store: %w", err)
	}
	if _, err := repository.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{locator}}); err != nil {
		return nil, fmt.Errorf("configure source remote: %w", err)
	}
	options := &git.FetchOptions{RemoteName: "origin", RefSpecs: []config.RefSpec{config.RefSpec("+" + sha + ":refs/source/pinned")}, Depth: 1, Tags: git.NoTags}
	if token != "" {
		options.Auth = &githttp.BasicAuth{Username: "x-access-token", Password: token}
	}
	if err := repository.FetchContext(ctx, options); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, errors.New("could not fetch the pinned source revision; check repository access")
	}
	if _, err := repository.CommitObject(plumbing.NewHash(sha)); err != nil {
		return nil, errors.New("the fetched repository does not contain the pinned commit")
	}
	return repository, nil
}

// readGitFiles reads original blobs directly; repository files are never executed.
func readGitFiles(ctx context.Context, repository *git.Repository, sha string, paths []string, budget int) (map[string]string, error) {
	if !commitPattern.MatchString(sha) {
		return nil, errors.New("invalid source revision")
	}
	commit, err := repository.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("read pinned source commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("read source tree: %w", err)
	}
	files := make(map[string]string, len(paths))
	total := int64(0)
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("read source files: %w", err)
		}
		if !filepath.IsLocal(path) || strings.ContainsAny(path, "\r\n\\") {
			return nil, errors.New("invalid source path")
		}
		file, err := tree.File(path)
		if err != nil {
			return nil, fmt.Errorf("read selected source file: %w", err)
		}
		if file.Mode != filemode.Regular && file.Mode != filemode.Executable {
			return nil, errors.New("source file is not a regular blob")
		}
		if file.Size < 0 || file.Size > 8<<20 {
			return nil, errors.New("source blob exceeds safety ceiling")
		}
		total += file.Size
		if total > int64(budget) {
			return nil, errors.New("repository exceeds the collection safety ceiling; no partial package was used")
		}
		reader, err := file.Reader()
		if err != nil {
			return nil, fmt.Errorf("open source blob: %w", err)
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, file.Size+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || int64(len(content)) != file.Size {
			return nil, errors.New("could not read the complete source blob")
		}
		if bytes.IndexByte(content, 0) < 0 {
			files[path] = string(content)
		}
	}
	return files, nil
}

type githubTokenKey struct{}

func (c *Collector) githubToken(ctx context.Context) string {
	if token, ok := ctx.Value(githubTokenKey{}).(string); ok {
		return token
	}
	if connectionowner.UserID(ctx) != "" {
		return ""
	}
	return c.GitHubToken
}
