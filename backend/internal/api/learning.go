package api

import (
	"net/http"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learning"
)

// Each person reads and replaces only their own learning progress.
func (f *Factory) registerLearning(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/me/learning", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		value, err := f.Learning.Get(r.Context(), currentUser(r).ID)
		respond(w, r, value, err)
	}))
	mux.HandleFunc("PUT /api/me/learning", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		input, ok := decode[learning.State](w, r)
		if !ok {
			return
		}
		value, err := f.Learning.Put(r.Context(), currentUser(r).ID, input)
		respond(w, r, value, err)
	}))
}
