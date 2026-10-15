package source

import (
	"strings"
	"testing"
)

func TestBrandAssetBoundaries(t *testing.T) {
	collector := Collector{}
	pinned := Snapshot{Locator: "https://github.com/example/product", Revision: strings.Repeat("a", 40)}
	for _, file := range []string{"../other/logo.svg", "/logo.svg", "images/../../logo.svg", "credentials.json", ".env"} {
		if _, err := collector.ReadBrandAsset(t.Context(), pinned, file); err == nil {
			t.Fatal("unsafe brand asset path accepted", file)
		}
	}
	for _, snapshot := range []Snapshot{
		{Locator: "https://127.0.0.1/example/product", Revision: pinned.Revision},
		{Locator: pinned.Locator, Revision: "main"},
	} {
		if _, err := collector.ReadBrandAsset(t.Context(), snapshot, "logo.svg"); err == nil {
			t.Fatal("unpinned or foreign asset source accepted")
		}
	}
	if !eligibleFile(treeFile{Path: "public/product-logo.svg", Type: "blob", Mode: "100644", Size: 100}) || eligibleFile(treeFile{Path: "icons/delete.svg", Type: "blob", Mode: "100644", Size: 100}) {
		t.Fatal("source selection must admit product logos without collecting arbitrary icons")
	}
}
