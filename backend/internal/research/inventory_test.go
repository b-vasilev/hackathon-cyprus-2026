package research

import (
	"reflect"
	"slices"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestInventoryIsIndependentDeterministicAndRevisionBound(t *testing.T) {
	collection := source.Collection{Snapshots: []source.Snapshot{{ID: "api", Locator: "api-repo", Revision: "sha-api"}, {ID: "engine", Locator: "engine-repo", Revision: "sha-engine"}}, Documents: []source.Document{
		{SnapshotID: "engine", Path: "go.mod", Content: "module example.org/engine\n"},
		{SnapshotID: "api", Path: "package.json", Content: `{"name":"api-product"}`},
		{SnapshotID: "api", Path: "cmd/api/main.go", Content: "package main\nfunc main() {\n RegisterHandlers()\n}"},
		{SnapshotID: "api", Path: "internal/client/engine.go", Content: "package client\nimport \"example.org/engine\""},
		{SnapshotID: "engine", Path: "internal/convert/registry.go", Content: "func RegisterAdapters() {}"},
		{SnapshotID: "engine", Path: ".github/workflows/build.yml", Content: "name: build"},
	}}
	got := buildInventory(collection)
	slices.Reverse(collection.Documents)
	slices.Reverse(collection.Snapshots)
	if !reflect.DeepEqual(got, buildInventory(collection)) {
		t.Fatal("inventory depends on map/input order")
	}
	for _, want := range []struct {
		repo, path, kind, target string
		line                     int
	}{
		{"api", "api/internal/client", "source_area", "", 0},
		{"api", "api/cmd/api/main.go", "entrypoint", "", 2},
		{"api", "api/internal/client/engine.go", "repository_reference", "engine", 2},
		{"engine", "engine/internal/convert/registry.go", "registry", "", 1},
		{"engine", "engine/.github/workflows/build.yml", "configuration", "", 1},
		{"engine", "engine/go.mod", "manifest", "", 1},
	} {
		if !slices.ContainsFunc(got.Candidates, func(c Candidate) bool {
			return c.Repository == want.repo && c.Path == want.path && c.Kind == want.kind && c.Line == want.line && c.TargetRepository == want.target
		}) {
			t.Errorf("missing candidate %+v", want)
		}
	}
	if got.Repositories[0].Revision != "sha-api" || got.Repositories[1].Revision != "sha-engine" {
		t.Fatal("pinned revisions lost")
	}
	single := buildInventory(source.Collection{Snapshots: collection.Snapshots[:1], Documents: []source.Document{{SnapshotID: "engine", Path: "main.go", Content: "package main\nfunc main() {}"}}})
	if slices.ContainsFunc(single.Candidates, func(c Candidate) bool { return c.Kind == "repository_reference" }) {
		t.Fatal("single-repository inventory invented a distributed boundary")
	}
}

func TestInventoryGroupsNestedVariantsUnderSubsystem(t *testing.T) {
	inv := buildInventory(source.Collection{
		Snapshots: []source.Snapshot{{ID: "engine", Revision: "pinned"}},
		Documents: []source.Document{
			{SnapshotID: "engine", Path: "internal/convert/targets/sql/helpers/render.go"},
			{SnapshotID: "engine", Path: "internal/convert/targets/dax/temporal/project.go"},
		},
	})
	areas := 0
	for _, c := range inv.Candidates {
		if c.Kind == "source_area" {
			areas++
			if c.Path != "engine/internal/convert" {
				t.Fatalf("nested variant became an independent discovery task: %s", c.Path)
			}
		}
	}
	if areas != 1 || inv.Repositories[0].Files != 2 {
		t.Fatal("subsystem grouping lost source inventory")
	}
}
