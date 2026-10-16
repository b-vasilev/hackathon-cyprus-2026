package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

// Leads inspect any version; learners see evidence only for the published version.
func (f *Factory) publishedEvidenceProduct(r *http.Request) (string, error) {
	id := r.PathValue("id")
	productID, err := f.Service.RevisionProduct(r.Context(), id)
	if err != nil {
		return "", fmt.Errorf("find revision product: %w", err)
	}
	role, err := f.Access.ProductRole(r.Context(), currentUser(r).ID, productID)
	if err != nil {
		return productID, fmt.Errorf("read product role: %w", err)
	}
	if role == access.RoleLead {
		return productID, nil
	}
	p, err := f.Service.Products.Get(r.Context(), productID)
	if err != nil {
		return "", fmt.Errorf("read published product: %w", err)
	}
	if p.PublishedRevisionID != id {
		return "", sql.ErrNoRows
	}
	return productID, nil
}

// registerReview exposes the lead/admin review workflow for passport versions.
func (f *Factory) registerReview(mux *http.ServeMux) {
	revisionProduct := func(r *http.Request) (string, error) {
		return f.Service.RevisionProduct(r.Context(), r.PathValue("id"))
	}
	reviewer := func(next func(*http.Request) (any, error)) http.HandlerFunc {
		return f.product(true, revisionProduct, func(w http.ResponseWriter, r *http.Request) {
			value, err := next(r)
			respond(w, r, value, err)
		})
	}
	mux.HandleFunc("POST /api/passport-revisions/{id}/publish", reviewer(func(r *http.Request) (any, error) {
		return f.Service.Publish(r.Context(), r.PathValue("id"), currentUser(r).ID)
	}))
	mux.HandleFunc("POST /api/passport-revisions/{id}/unpublish", reviewer(func(r *http.Request) (any, error) {
		return f.Service.Unpublish(r.Context(), r.PathValue("id"), currentUser(r).ID)
	}))
	mux.HandleFunc("GET /api/passport-revisions/{id}/review", reviewer(func(r *http.Request) (any, error) {
		return f.Service.Review(r.Context(), r.PathValue("id"))
	}))
	mux.HandleFunc("DELETE /api/passport-revisions/{id}/flag", reviewer(func(r *http.Request) (any, error) {
		return f.Service.ResolveFlag(r.Context(), r.PathValue("id"), currentUser(r).ID)
	}))
	mux.HandleFunc("PUT /api/passport-revisions/{id}/flag", withInput(f, revisionProduct, func(r *http.Request, input generation.FlagInput) (any, error) {
		return f.Service.Flag(r.Context(), r.PathValue("id"), currentUser(r).ID, input)
	}))
	mux.HandleFunc("POST /api/passport-revisions/{id}/comments", withInput(f, revisionProduct, func(r *http.Request, input generation.CommentInput) (any, error) {
		return f.Service.Comment(r.Context(), r.PathValue("id"), currentUser(r).ID, input)
	}))
	mux.HandleFunc("PUT /api/passport-revisions/{id}/chapters/{chapterId}", withInput(f, revisionProduct, func(r *http.Request, input passport.ChapterEdit) (any, error) {
		return f.Service.EditChapter(r.Context(), r.PathValue("id"), r.PathValue("chapterId"), currentUser(r).ID, input)
	}))
}

// withInput decodes a JSON body for a lead/admin revision action.
func withInput[T any](f *Factory, resolve productResolver, next func(*http.Request, T) (any, error)) http.HandlerFunc {
	return f.product(true, resolve, func(w http.ResponseWriter, r *http.Request) {
		input, ok := decode[T](w, r)
		if !ok {
			return
		}
		value, err := next(r, input)
		respond(w, r, value, err)
	})
}
