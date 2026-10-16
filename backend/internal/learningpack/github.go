package learningpack

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	githubAPI      = "https://api.github.com"
	pullsInspected = 15
	candidateLimit = 6
	bodyLimit      = 1500
)

// "Fixes #12" and similar closing keywords link a pull request to the issue it resolved.
var closingReference = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)`)

// Candidate is a closed issue offered to the model, with the context it needs to choose.
type Candidate struct {
	Issue
	Body         string   `json:"body"`
	Labels       []string `json:"labels"`
	PullBody     string   `json:"pull_body"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	ChangedFiles int      `json:"changed_files"`
}

type githubClient struct {
	http  *http.Client
	token string
	base  string
}

func (g githubClient) get(ctx context.Context, path string, value any) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base+path, http.NoBody)
	if err != nil {
		return fmt.Errorf("build GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return fmt.Errorf("call GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned %s for %s", resp.Status, path)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(value); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}
	return nil
}

type githubPull struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	HTMLURL      string `json:"html_url"`
	MergedAt     string `json:"merged_at"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	ChangedFiles int    `json:"changed_files"`
}

type githubIssue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	State       string `json:"state"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// candidates finds recent closed issues resolved by small merged pull requests. Small
// changes are easier to study; the model still picks the one that teaches the most.
func (g githubClient) candidates(ctx context.Context, repository string) ([]Candidate, error) {
	var pulls []githubPull
	if err := g.get(ctx, "/repos/"+repository+"/pulls?state=closed&sort=updated&direction=desc&per_page=100", &pulls); err != nil {
		return nil, err
	}
	type linked struct {
		pull  githubPull
		issue int
	}
	var found []linked
	for _, pull := range pulls {
		if pull.MergedAt == "" {
			continue
		}
		match := closingReference.FindStringSubmatch(pull.Body)
		if match == nil {
			continue
		}
		issue, _ := strconv.Atoi(match[1])
		found = append(found, linked{pull: pull, issue: issue})
		if len(found) == pullsInspected {
			break
		}
	}
	// The list endpoint omits diff sizes, so read them for the inspected pull requests.
	for i := range found {
		if err := g.get(ctx, fmt.Sprintf("/repos/%s/pulls/%d", repository, found[i].pull.Number), &found[i].pull); err != nil {
			return nil, err
		}
	}
	slices.SortStableFunc(found, func(a, b linked) int {
		return cmp.Compare(a.pull.Additions+a.pull.Deletions, b.pull.Additions+b.pull.Deletions)
	})
	var result []Candidate
	for _, item := range found {
		var issue githubIssue
		if err := g.get(ctx, fmt.Sprintf("/repos/%s/issues/%d", repository, item.issue), &issue); err != nil || issue.PullRequest != nil || issue.State != "closed" {
			continue
		}
		var files []struct {
			Filename string `json:"filename"`
		}
		if err := g.get(ctx, fmt.Sprintf("/repos/%s/pulls/%d/files?per_page=30", repository, item.pull.Number), &files); err != nil {
			return nil, err
		}
		candidate := Candidate{
			Repository: repository, Number: issue.Number, Title: issue.Title, URL: issue.HTMLURL,
			PullNumber: item.pull.Number, PullTitle: item.pull.Title, PullURL: item.pull.HTMLURL,
			Body: clip(issue.Body, bodyLimit), PullBody: clip(item.pull.Body, bodyLimit),
			Additions: item.pull.Additions, Deletions: item.pull.Deletions, ChangedFiles: item.pull.ChangedFiles,
		}
		for _, file := range files {
			candidate.Files = append(candidate.Files, file.Filename)
		}
		for _, label := range issue.Labels {
			candidate.Labels = append(candidate.Labels, label.Name)
		}
		result = append(result, candidate)
		if len(result) == candidateLimit {
			break
		}
	}
	return result, nil
}

// repositoryName turns https://github.com/owner/repo(.git) into owner/repo.
func repositoryName(locator string) (string, bool) {
	rest, ok := strings.CutPrefix(locator, "https://github.com/")
	if !ok {
		return "", false
	}
	parts := strings.Split(strings.TrimSuffix(strings.Trim(rest, "/"), ".git"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

func clip(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return strings.ToValidUTF8(text[:limit], "") + " …"
}
