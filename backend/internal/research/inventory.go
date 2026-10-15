package research

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// Inventory is a deterministic map of candidates in the collected snapshot, not
// an architecture or a claim that the collector included every repository file.
type Inventory struct {
	Repositories []InventoryRepository `json:"repositories"`
	Candidates   []Candidate           `json:"candidates"`
	Note         string                `json:"note"`
}

type InventoryRepository struct {
	ID       string `json:"id"`
	Locator  string `json:"locator"`
	Revision string `json:"revision"`
	Files    int    `json:"indexed_files"`
}

type Candidate struct {
	ID               string `json:"id"`
	Repository       string `json:"repository"`
	Path             string `json:"path"`
	Kind             string `json:"kind"`
	Reason           string `json:"reason"`
	Line             int    `json:"line,omitempty"`
	TargetRepository string `json:"target_repository,omitempty"`
}

var discoveryPatterns = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"entrypoint", regexp.MustCompile(`^\s*(func main\(|(?:public\s+)?(?:static\s+)?(?:async\s+)?(?:void|Task|int)\s+Main\(|if __name__\s*==|(?:export\s+)?(?:async\s+)?function main\(|(?:const|var)\s+app\s*=\s*(?:express|FastAPI)\()`)},
	{"registry", regexp.MustCompile(`(?i)(Register(?:Routes|Handlers|Services|Tools|Adapters)|Map(?:Get|Post|Put|Delete|Controllers)|AddScoped|AddSingleton|createRouter|(?:adapter|plugin|provider|handler|tool)Registry|switch\s*\([^)]*(?:source|target|provider|adapter))`)},
}

func buildInventory(collection source.Collection) Inventory {
	inv := Inventory{Note: "Research major runtime source areas and repository responsibilities. Individual manifests, entrypoints, registries, configurations, text references and support areas are optional navigation hints, not separate coverage obligations. Read representative flows; promote a hint only for a concrete architectural question. Name/content heuristics and range checks do not prove semantic completeness."}
	counts := map[string]int{}
	identities := map[string][]string{}
	for _, doc := range collection.Documents {
		counts[doc.SnapshotID]++
		identities[doc.SnapshotID] = append(identities[doc.SnapshotID], manifestIdentities(doc)...)
	}
	for _, snap := range collection.Snapshots {
		inv.Repositories = append(inv.Repositories, InventoryRepository{snap.ID, snap.Locator, snap.Revision, counts[snap.ID]})
	}
	slices.SortFunc(inv.Repositories, func(a, b InventoryRepository) int { return strings.Compare(a.ID, b.ID) })
	seen := map[string]bool{}
	add := func(repo, file, kind, reason string, line int, target string) {
		key := repo + "\x00" + file + "\x00" + kind + "\x00" + target
		if seen[key] {
			return
		}
		seen[key] = true
		sum := sha256.Sum256([]byte(key))
		inv.Candidates = append(inv.Candidates, Candidate{fmt.Sprintf("candidate-%x", sum[:10]), repo, file, kind, reason, line, target})
	}
	for _, snap := range collection.Snapshots {
		add(snap.ID, snap.ID+"/.", "repository", "Supplied repository at pinned revision; account for missing or non-product sources explicitly.", 0, "")
	}
	for _, doc := range collection.Documents {
		file := doc.SnapshotID + "/" + doc.Path
		area := sourceArea(doc.Path)
		add(doc.SnapshotID, doc.SnapshotID+"/"+area, "source_area", "Indexed directory group; inspect and classify its responsibility or exclusion.", 0, "")
		// Nested helpers and variants remain searchable without each creating
		// a separate discovery obligation.
		base := strings.ToLower(path.Base(doc.Path))
		if isManifest(base) {
			add(doc.SnapshotID, file, "manifest", "Build/package manifest filename.", 1, "")
		}
		if isConfig(base, doc.Path) {
			add(doc.SnapshotID, file, "configuration", "Build, deployment, CI or application configuration filename.", 1, "")
		}
		lineNumber := 0
		for line := range strings.SplitSeq(doc.Content, "\n") {
			lineNumber++
			for _, pattern := range discoveryPatterns {
				if pattern.re.MatchString(line) {
					add(doc.SnapshotID, file, pattern.kind, "Static source pattern; verify behavior and variants by reading.", lineNumber, "")
				}
			}
			for target, names := range identities {
				if target == doc.SnapshotID {
					continue
				}
				for _, name := range names {
					if strings.Contains(line, name) {
						add(doc.SnapshotID, file, "repository_reference", "Text mentions another supplied repository's manifest identity; verify both endpoints or explain the false lead.", lineNumber, target)
					}
				}
			}
		}
	}
	slices.SortFunc(inv.Candidates, func(a, b Candidate) int {
		return strings.Compare(a.Repository+"/"+a.Path+"/"+a.Kind+"/"+a.TargetRepository, b.Repository+"/"+b.Path+"/"+b.Kind+"/"+b.TargetRepository)
	})
	return inv
}

// Expand structural wrappers once; preserve meaningful package directories.
func sourceArea(file string) string {
	parts := strings.Split(path.Dir(file), "/")
	if parts[0] == "." {
		return "."
	}
	end := 1
	for end < len(parts) && slices.Contains([]string{"src", "source", "internal", "pkg", "cmd", "apps", "packages", "backend", "frontend", "lib", "libs", "tests"}, strings.ToLower(parts[end-1])) {
		end++
	}
	return strings.Join(parts[:end], "/")
}

func isManifest(base string) bool {
	return slices.Contains([]string{"go.mod", "package.json", "pyproject.toml", "cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts", "requirements.txt", "gemfile", "composer.json", "pnpm-workspace.yaml", "directory.build.props", "global.json"}, base) || strings.HasSuffix(base, ".csproj") || strings.HasSuffix(base, ".sln")
}

func isConfig(base, file string) bool {
	return slices.Contains([]string{"dockerfile", "compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml", "makefile", "justfile", "settings.py", "manage.py", "program.cs", "startup.cs"}, base) || strings.HasPrefix(base, "appsettings") || strings.HasPrefix(base, "vite.config.") || strings.HasPrefix(base, "next.config.") || strings.HasPrefix(file, ".github/workflows/") || strings.HasSuffix(base, ".tf")
}

func (w *workspace) inventoryTool() fantasy.AgentTool {
	return fantasy.NewAgentTool("list_research_inventory", "Inspect architecture source areas and optional file hints. Only major runtime source areas and repositories need coverage by default; do not classify or read every hinted file. Promote a file hint with an explicit deferred disposition only when a concrete product flow or external contract requires it. offset=0, limit=60; use exact candidate IDs.", func(_ context.Context, input listInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
		if err := w.tick("Reviewing independent repository candidates."); err != nil {
			return fantasy.ToolResponse{}, err
		}
		candidates := w.inventory.Candidates
		if w.task != nil {
			candidates = w.task.Candidates
		}
		start := min(max(input.Offset, 0), len(candidates))
		end := min(start+min(max(input.Limit, 1), 100), len(candidates))
		return toolJSON(map[string]any{"repositories": w.inventory.Repositories, "note": w.inventory.Note, "candidates": candidates[start:end], "next_offset": end, "has_more": end < len(candidates), "total": len(candidates)})
	})
}

func manifestIdentities(doc source.Document) []string {
	var result []string
	if path.Base(doc.Path) == "go.mod" {
		for line := range strings.SplitSeq(doc.Content, "\n") {
			if module, ok := strings.CutPrefix(line, "module "); ok {
				result = append(result, strings.TrimSpace(module))
			}
		}
	}
	if path.Base(doc.Path) == "package.json" {
		var manifest struct {
			Name string `json:"name"`
		}
		if json.Unmarshal([]byte(doc.Content), &manifest) == nil && manifest.Name != "" {
			result = append(result, manifest.Name)
		}
	}
	if strings.EqualFold(path.Ext(doc.Path), ".csproj") {
		result = append(result, strings.TrimSuffix(path.Base(doc.Path), path.Ext(doc.Path))+".csproj")
	}
	return result
}
