package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
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
