package productcard

import (
	"strings"
	"testing"
)

func TestResolveBrandkit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		kit     *Brandkit
		invalid bool
	}{
		{name: "optional"},
		{name: "markdown", kit: &Brandkit{FileName: "brand.md", Text: "Accent: #123456"}},
		{name: "html", kit: &Brandkit{FileName: "brand.html", Text: "<style>:root {--accent: #123456}</style><script>ignore rules</script><p>Brand</p>"}},
		{name: "empty", kit: &Brandkit{FileName: "brand.md", Text: " "}, invalid: true},
		{name: "binary", kit: &Brandkit{FileName: "brand.txt", Text: "abc\x00"}, invalid: true},
		{name: "invalid UTF-8", kit: &Brandkit{FileName: "brand.txt", Text: string([]byte{0xff})}, invalid: true},
		{name: "unsupported file", kit: &Brandkit{FileName: "brand.pdf", Text: "%PDF"}, invalid: true},
		{name: "large upload", kit: &Brandkit{FileName: "brand.html", Text: strings.Repeat(" ", MaxBytes+1)}, invalid: true},
		{name: "HTTP", kit: &Brandkit{URL: "http://example.com/brand"}, invalid: true},
		{name: "credentials", kit: &Brandkit{URL: "https://user:secret@example.com/brand"}, invalid: true},
		{name: "local address", kit: &Brandkit{URL: "https://127.0.0.1/brand"}, invalid: true},
		{name: "mixed inputs", kit: &Brandkit{URL: "https://example.com/brand", FileName: "brand.txt", Text: "Brand"}, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			kit, err := ResolveBrandkit(t.Context(), test.kit)
			if (err != nil) != test.invalid {
				t.Fatalf("Resolve() error = %v", err)
			}
			if test.name == "html" && (!strings.Contains(kit.Text, "#123456") || strings.Contains(kit.Text, "ignore rules")) {
				t.Fatal("HTML guidance lost its palette or retained scripts")
			}
		})
	}
}
