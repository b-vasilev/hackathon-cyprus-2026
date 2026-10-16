package api

import (
	"errors"
	"net/http"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learningpack"
)

// Leads generate, check and publish a passport version's learning pack; members take the published one.
func (f *Factory) registerLearningPacks(mux *http.ServeMux) {
	revisionProduct := func(r *http.Request) (string, error) {
		return f.Service.RevisionProduct(r.Context(), r.PathValue("id"))
	}
	reviewer := func(next func(*http.Request) (any, error)) http.HandlerFunc {
		return f.product(true, revisionProduct, func(w http.ResponseWriter, r *http.Request) {
			value, err := next(r)
			respond(w, r, value, err)
		})
	}
	mux.HandleFunc("GET /api/passport-revisions/{id}/learning-pack", reviewer(func(r *http.Request) (any, error) {
		return f.Packs.Get(r.Context(), r.PathValue("id"))
	}))
	mux.HandleFunc("POST /api/passport-revisions/{id}/learning-pack", reviewer(func(r *http.Request) (any, error) {
		return f.Packs.Start(r.Context(), r.PathValue("id"), currentUser(r).ID)
	}))
	mux.HandleFunc("POST /api/passport-revisions/{id}/learning-pack/publish", f.product(true, revisionProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Packs.Publish(r.Context(), r.PathValue("id"), currentUser(r).ID)
		if errors.Is(err, learningpack.ErrNotReady) {
			problem(w, http.StatusConflict, "learning_pack_not_ready", err.Error())
			return
		}
		respond(w, r, value, err)
	}))
	mux.HandleFunc("POST /api/passport-revisions/{id}/learning-pack/unpublish", reviewer(func(r *http.Request) (any, error) {
		return f.Packs.Unpublish(r.Context(), r.PathValue("id"))
	}))
	mux.HandleFunc("GET /api/products/{id}/learning-pack", f.product(false, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Packs.Published(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
}
