// Package product manages source configuration for reusable product passports.
package product

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/productcard"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// Product contains source configuration and the latest generation pointers.
type Product struct {
	ID                  string                     `json:"id"`
	Name                string                     `json:"name"`
	Sources             []source.Input             `json:"sources"`
	CardBrandkit        *productcard.Brandkit      `json:"card_brandkit,omitempty"`
	CardTheme           *passport.ProductCardTheme `json:"card_theme,omitempty"`
	IconURL             string                     `json:"icon_url" jsonschema:"Custom product icon as a data URI, or empty to use the passport logo."`
	CreatedAt           string                     `json:"created_at"`
	LatestRevisionID    string                     `json:"latest_revision_id"`
	PublishedRevisionID string                     `json:"published_revision_id"`
	LatestRunID         string                     `json:"latest_run_id"`
	LatestRunStatus     string                     `json:"latest_run_status"`
}

// Input registers a product without requiring a manually written passport.
type Input struct {
	Name         string                `json:"name"`
	Sources      []source.Input        `json:"sources"`
	CardBrandkit *productcard.Brandkit `json:"card_brandkit,omitempty"`
}

// IconInput uploads a custom product icon.
type IconInput struct {
	Data string `json:"data" jsonschema:"Base64-encoded SVG, PNG, JPEG or GIF image, up to 256 KB."`
}

// configuration extends the existing JSON field without changing the DB schema.
type configuration struct {
	Sources      []source.Input             `json:"sources"`
	CardBrandkit *productcard.Brandkit      `json:"card_brandkit,omitempty"`
	CardTheme    *passport.ProductCardTheme `json:"card_theme,omitempty"`
	Icon         string                     `json:"icon,omitempty"`
}

// ErrInvalid identifies invalid product input.
var ErrInvalid = errors.New("invalid product")

// Store uses the application's existing SQLite connection.
type Store struct{ DB *sql.DB }

// Create validates and persists the input sources.
func (s Store) Create(ctx context.Context, input Input) (Product, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 {
		return Product{}, fmt.Errorf("%w: product name must contain 1–100 characters", ErrInvalid)
	}
	if err := source.Validate(input.Sources); err != nil {
		return Product{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	kit, err := productcard.ResolveBrandkit(ctx, input.CardBrandkit)
	if err != nil {
		return Product{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	encoded, err := json.Marshal(configuration{Sources: input.Sources, CardBrandkit: kit})
	if err != nil {
		return Product{}, fmt.Errorf("encode sources: %w", err)
	}
	p := Product{ID: rand.Text(), Name: input.Name, Sources: input.Sources, CardBrandkit: kit, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO products(id,name,sources_json,created_at) VALUES(?,?,?,?)", p.ID, p.Name, string(encoded), p.CreatedAt); err != nil {
		return Product{}, fmt.Errorf("save product: %w", err)
	}
	return p, nil
}

const selectProduct = `SELECT p.id,p.name,p.sources_json,p.created_at,
 COALESCE((SELECT id FROM passport_revisions WHERE product_id=p.id ORDER BY version DESC LIMIT 1),''),
 COALESCE((SELECT id FROM generation_runs WHERE product_id=p.id ORDER BY created_at DESC LIMIT 1),''),
 COALESCE((SELECT status FROM generation_runs WHERE product_id=p.id ORDER BY created_at DESC LIMIT 1),''),
 COALESCE(p.published_revision_id,'') FROM products p`

type scanner interface{ Scan(...any) error }

func scanProduct(row scanner) (Product, error) {
	var p Product
	var sources string
	if err := row.Scan(&p.ID, &p.Name, &sources, &p.CreatedAt, &p.LatestRevisionID, &p.LatestRunID, &p.LatestRunStatus, &p.PublishedRevisionID); err != nil {
		return p, fmt.Errorf("read product: %w", err)
	}
	if strings.HasPrefix(strings.TrimSpace(sources), "[") {
		if err := json.Unmarshal([]byte(sources), &p.Sources); err != nil {
			return p, fmt.Errorf("decode product sources: %w", err)
		}
	} else {
		var config configuration
		if err := json.Unmarshal([]byte(sources), &config); err != nil {
			return p, fmt.Errorf("decode product configuration: %w", err)
		}
		p.Sources, p.CardBrandkit, p.CardTheme, p.IconURL = config.Sources, config.CardBrandkit, config.CardTheme, config.Icon
	}
	return p, nil
}

// Get returns one product and its latest run/revision metadata.
func (s Store) Get(ctx context.Context, id string) (Product, error) {
	return scanProduct(s.DB.QueryRowContext(ctx, selectProduct+" WHERE p.id=?", id))
}

// List returns products ordered by creation time.
func (s Store) List(ctx context.Context) ([]Product, error) {
	rows, err := s.DB.QueryContext(ctx, selectProduct+" ORDER BY p.created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	result := make([]Product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read products: %w", err)
	}
	return result, nil
}

// Older products store a source array; wrap it before adding configuration keys.
const (
	productConfiguration = "CASE WHEN json_type(sources_json)='array' THEN json_object('sources',json(sources_json)) ELSE sources_json END"
	setIcon              = "UPDATE products SET sources_json=json_set(" + productConfiguration + ",'$.icon',?) WHERE id=?"
	removeIcon           = "UPDATE products SET sources_json=json_remove(" + productConfiguration + ",'$.icon') WHERE id=?"
)

// SetIcon replaces the product's custom icon after validating the image.
func (s Store) SetIcon(ctx context.Context, id string, input IconInput) (Product, error) {
	data, err := base64.StdEncoding.DecodeString(input.Data)
	icon := ""
	if err == nil {
		icon = productcard.IconData(data)
	}
	if icon == "" {
		return Product{}, fmt.Errorf("%w: use a safe SVG, PNG, JPEG or GIF image up to 256 KB", ErrInvalid)
	}
	return s.updateConfiguration(ctx, id, setIcon, icon, id)
}

// RemoveIcon restores the passport logo or product initial.
func (s Store) RemoveIcon(ctx context.Context, id string) (Product, error) {
	return s.updateConfiguration(ctx, id, removeIcon, id)
}

func (s Store) updateConfiguration(ctx context.Context, id, query string, args ...any) (Product, error) {
	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return Product{}, fmt.Errorf("update product configuration: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil || changed == 0 {
		return Product{}, errors.Join(sql.ErrNoRows, err)
	}
	return s.Get(ctx, id)
}
