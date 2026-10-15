package source

import (
	"path/filepath"
	"strings"
)

// IsBrandFile identifies likely identity sources, not arbitrary product images.
func IsBrandFile(path string) bool {
	path = strings.ToLower(path)
	name := filepath.Base(path)
	if strings.Contains(name, "_test.") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
		return false
	}
	//nolint:misspell // Source file names can use British spelling.
	for _, part := range []string{"brand", "logo", "wordmark", "palette", "design-token", "theme", "colors.", "colours."} {
		if strings.Contains(path, part) {
			return true
		}
	}
	return name == "readme.md" || name == "readme" || name == "readme.mdx" || name == "style.css" || name == "styles.css" || name == "globals.css" || strings.HasPrefix(name, "tailwind.config.")
}
