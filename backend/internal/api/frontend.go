package api

import (
	"net/http"
	"os"
	"path"
	"strings"
)

// frontend serves built assets and the known client-side routes from one origin.
func frontend(directory string) http.Handler {
	files := http.FileServer(http.Dir(directory))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		spa := name == "" || name == "." || name == "products" || name == "connections" ||
			name == "login" || name == "register" || name == "access" ||
			(strings.HasPrefix(name, "products/") && strings.Count(name, "/") == 1) ||
			name == "foundations" || (strings.HasPrefix(name, "foundations/") && strings.Count(name, "/") == 1)
		if spa {
			name = "index.html"
		}
		root := os.DirFS(directory)
		file, err := root.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		info, err := file.Stat()
		_ = file.Close()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; worker-src 'self' blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if spa {
			w.Header().Set("Cache-Control", "no-cache")
			cloned := r.Clone(r.Context())
			cloned.URL.Path = "/"
			files.ServeHTTP(w, cloned)
			return
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}
