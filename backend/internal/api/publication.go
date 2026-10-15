package api

import (
	"database/sql"
	"net/http"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
)

// Leads inspect any version; learners see evidence only for the published version.
func (f *Factory) publishedEvidenceProduct(r *http.Request) (string, error) {
	id := r.PathValue("id")
	productID, err := f.Service.RevisionProduct(r.Context(), id)
	if err != nil {
		return "", err
	}
	role, err := f.Access.ProductRole(r.Context(), currentUser(r).ID, productID)
	if err != nil || role == access.RoleLead {
		return productID, err
	}
	p, err := f.Service.Products.Get(r.Context(), productID)
	if err != nil {
		return "", err
	}
	if p.PublishedRevisionID != id {
		return "", sql.ErrNoRows
	}
	return productID, nil
}
