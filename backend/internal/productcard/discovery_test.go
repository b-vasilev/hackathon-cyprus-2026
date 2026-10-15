package productcard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

const testLogo = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><path fill="#123456" d="M0 0H40V40Z"/></svg>`

func brandCollection() source.Collection {
	return source.Collection{
		Snapshots: []source.Snapshot{{ID: "repo", Locator: "https://github.com/example/product", Revision: strings.Repeat("a", 40)}},
		Documents: []source.Document{
			{SnapshotID: "repo", Path: "README.md", Content: "Product [brand guide](https://example.com/brand/guide)."},
			{SnapshotID: "repo", Path: "src/theme.css", Content: ":root { --primary: #123456; --surface: #fff; }"},
			{SnapshotID: "repo", Path: "public/logo.svg", Content: testLogo},
		},
	}
}

func TestDiscoverProductBrand(t *testing.T) {
	var fetched []string
	fetch := func(_ context.Context, location string, _ int64) ([]byte, string, error) {
		fetched = append(fetched, location)
		switch location {
		case "https://example.com/brand/guide":
			return []byte(`<p>Product primary #123456</p><img src="/brand/logo.svg"><a href="/brand/another">Another guide</a>`), "text/html", nil
		case "https://example.com/brand/logo.svg":
			return []byte(testLogo), "image/svg+xml", nil
		default:
			return nil, "", errors.New("unexpected resource")
		}
	}
	d := discover(t.Context(), brandCollection(), nil, fetch)
	if len(fetched) != 2 || len(d.References) != 4 || len(d.Logos) != 2 {
		t.Fatalf("unexpected discovery: %d fetches, %d references, %d logos", len(fetched), len(d.References), len(d.Logos))
	}
	if !strings.Contains(d.Logos[0].URL, "/blob/"+strings.Repeat("a", 40)+"/public/logo.svg") {
		t.Fatal("repository logo was not pinned to the analyzed commit")
	}
	if strings.Contains(d.Context(), "data:image/") || !strings.Contains(d.Context(), "logo_candidates") {
		t.Fatal("model context must contain selectable original URLs, not image bytes")
	}
	theme := d.Theme(&passport.ProductCardTheme{AccentColor: "#123456", BackgroundColor: "#FFFFFF", LogoURL: d.Logos[0].URL})
	if theme == nil || !strings.HasPrefix(theme.LogoURL, "data:image/svg+xml;base64,") || theme.BackgroundColor != "#ffffff" || len(theme.Sources) == 0 {
		t.Fatal("verified palette, original logo or provenance was lost")
	}
	if d.Theme(theme).LogoURL != theme.LogoURL {
		t.Fatal("repair discarded the frozen logo")
	}
}

func TestBrandFallbackAndManualOverride(t *testing.T) {
	fetch := func(context.Context, string, int64) ([]byte, string, error) {
		return nil, "", errors.New("unavailable")
	}
	d := discover(t.Context(), brandCollection(), nil, fetch)
	if len(d.Notes) != 1 || d.Theme(&passport.ProductCardTheme{AccentColor: "#BADBAD", BackgroundColor: "#000000", LogoURL: "https://invented.example/logo.svg"}) != nil {
		t.Fatal("unavailable sources or invented branding were not handled")
	}
	manual := discover(t.Context(), brandCollection(), &Brandkit{FileName: "override.md", Text: "Primary #654321"}, fetch)
	if manual.Theme(&passport.ProductCardTheme{AccentColor: "#123456"}) != nil {
		t.Fatal("repository palette overrode the explicit brand kit")
	}
	theme := manual.Theme(&passport.ProductCardTheme{AccentColor: "#654321", BackgroundColor: "#BADBAD"})
	if theme == nil || theme.BackgroundColor != "#FDFCFD" || len(theme.Sources) != 1 || theme.Sources[0].Label != "override.md" {
		t.Fatal("manual kit or neutral background fallback was lost")
	}
	withLogo := manual.Theme(&passport.ProductCardTheme{AccentColor: "#654321", LogoURL: manual.Logos[0].URL})
	if withLogo == nil || withLogo.LogoURL == "" || withLogo.AccentColor != "#654321" {
		t.Fatal("discovered logo did not fill the manual kit's missing logo")
	}
	if (Discovery{}).Theme(&passport.ProductCardTheme{AccentColor: "#123456"}) != nil {
		t.Fatal("a product without branding acquired a theme")
	}
}

func TestUnsafeLogosAreRejected(t *testing.T) {
	for _, input := range []string{
		`<svg onload="alert(1)"></svg>`,
		`<svg><script>alert(1)</script></svg>`,
		`<svg><foreignObject><div>HTML</div></foreignObject></svg>`,
		`<svg><use href="https://example.com/logo.svg"/></svg>`,
		`<svg><path style="fill:url(https://example.com/paint)"/></svg>`,
		`<svg><style>@import "https://example.com/style.css";</style></svg>`,
		`<svg/><svg/>`,
		`<html>not an image</html>`,
	} {
		if logoData([]byte(input)) != "" {
			t.Fatal("unsafe logo accepted", input)
		}
	}
	if logoData([]byte(testLogo)) == "" {
		t.Fatal("original static SVG rejected")
	}
	if logoData([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><style>.logo{fill:#123456}</style><defs><path id="mark" d="M0 0H40V40Z"/></defs><use href="#mark" class="logo"/></svg>`)) == "" {
		t.Fatal("static logo with local styles and shapes rejected")
	}
}

func TestRepositoryLogoLinksKeepTheirCommit(t *testing.T) {
	base := repositoryURL(brandCollection().Snapshots[0], "docs/README.md")
	for _, input := range []struct{ text, suffix string }{
		{"![Product logo](../public/logo.svg)", "/public/logo.svg"},
		{"![Product logo](/public/logo.svg)", "/public/logo.svg"},
	} {
		found := links(Reference{URL: base, Text: input.text})
		if len(found) != 1 || !strings.Contains(found[0], "/blob/"+strings.Repeat("a", 40)+input.suffix) {
			t.Fatal("relative logo escaped the pinned repository", found)
		}
	}
}

func TestLogoPathsInBrandGuide(t *testing.T) {
	for _, test := range []struct {
		name, location, text, want string
	}{
		{"URL guide", "https://example.com/docs/brand.md", "Asset base URL: `https://example.com/identity/`\n`mark/product-blue.svg`", "https://example.com/identity/mark/product-blue.svg"},
		{"uploaded guide", "", "Asset base URL: `https://example.com/identity`\n`mark/product-blue.svg`", "https://example.com/identity/mark/product-blue.svg"},
		{"Markdown asset", "https://example.com/docs/brand.md", "Asset base URL: https://example.com/identity/\n![Product logo](mark/product-blue.svg)", "https://example.com/identity/mark/product-blue.svg"},
		{"document relative", "https://example.com/brand/guide.md", "`logo.svg`", "https://example.com/brand/logo.svg"},
		{"repository relative", repositoryURL(brandCollection().Snapshots[0], "docs/brand.md"), "`../public/logo.svg`", repositoryURL(brandCollection().Snapshots[0], "public/logo.svg")},
		{"no base", "", "`mark/product-blue.svg`", ""},
		{"unsafe base", "", "Asset base URL: `https://user:password@example.com/brand/`\n`mark/product-blue.svg`", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			found := logoLinks(Reference{URL: test.location, Text: test.text})
			if test.want == "" {
				if len(found) != 0 {
					t.Fatal("unexpected logo", found)
				}
			} else if len(found) != 1 || found[0] != test.want {
				t.Fatal("logo path was not resolved", found)
			}
		})
	}
}

func TestDiscoverBrandGuideAssetPaths(t *testing.T) {
	// Naming examples precede the actual asset paths, as in the Antares guide.
	kit := &Brandkit{URL: "https://example.com/brand/agents.md", Text: "Asset base URL: `https://example.com/brand/`\nPrimary #123456\n" +
		"SVG naming example: `product-mark-blue.svg`\n" +
		"| Logo, light surface | `mark/product-blue.svg` |\n" +
		"| Logo, light surface | `mark/product-blue.svg` |"}
	var images []string
	fetch := func(_ context.Context, location string, _ int64) ([]byte, string, error) {
		if strings.HasSuffix(location, ".svg") {
			images = append(images, location)
		}
		if location == "https://example.com/brand/mark/product-blue.svg" {
			return []byte(testLogo), "image/svg+xml", nil
		}
		return nil, "", errors.New("unavailable")
	}
	d := discover(t.Context(), source.Collection{}, kit, fetch)
	if len(images) != 2 || images[0] != "https://example.com/brand/mark/product-blue.svg" || len(d.Logos) != 1 {
		t.Fatalf("expected actual paths before naming examples, without duplicates: %v; logos: %d", images, len(d.Logos))
	}
	theme := d.Theme(&passport.ProductCardTheme{AccentColor: "#123456", LogoURL: d.Logos[0].URL})
	if theme == nil || theme.AccentColor != "#123456" || !strings.HasPrefix(theme.LogoURL, "data:image/svg+xml;base64,") {
		t.Fatal("brand guide palette or original logo was lost")
	}
}

func TestPrivateRepositoryLogoUsesSourceAccess(t *testing.T) {
	collection := brandCollection()
	collection.Documents = []source.Document{{SnapshotID: "repo", Path: "README.md", Content: "![Product logo](public/logo.svg)"}}
	called := false
	read := func(_ context.Context, snapshot source.Snapshot, file string) ([]byte, error) {
		called = true
		if snapshot.Revision != strings.Repeat("a", 40) || file != "public/logo.svg" {
			t.Fatal("logo request was not confined to the analyzed source revision", snapshot.Revision, file)
		}
		return []byte(testLogo), nil
	}
	d := Discover(t.Context(), collection, nil, read)
	if !called || len(d.Logos) != 1 || !strings.HasPrefix(d.Logos[0].DataURL, "data:image/svg+xml;") {
		t.Fatal("private repository logo did not use the authenticated source reader")
	}
}
