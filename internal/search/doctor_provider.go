package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type doctorProvider struct{ db *sql.DB }

// NewDoctorProvider creates a SearchProvider that searches the doctors table.
func NewDoctorProvider(db *sql.DB) SearchProvider { return &doctorProvider{db: db} }

func (p *doctorProvider) Type() EntityType { return EntityDoctor }
func (p *doctorProvider) Label() string    { return "Doctors" }

func (p *doctorProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	q := `
		SELECT id, full_name, specialty, license_number
		FROM doctors
		WHERE tenant_id = $1
		  AND (
		        full_name      ILIKE $2 OR
		        specialty      ILIKE $2 OR
		        license_number ILIKE $2
		      )
		ORDER BY full_name ASC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, req.TenantID, pattern, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("doctors: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, fullName string
		var specialty, licenseNum sql.NullString
		if err := rows.Scan(&id, &fullName, &specialty, &licenseNum); err != nil {
			return nil, fmt.Errorf("doctors scan: %w", err)
		}

		var subs []string
		if specialty.Valid && specialty.String != "" {
			subs = append(subs, specialty.String)
		}
		if licenseNum.Valid && licenseNum.String != "" {
			subs = append(subs, "#"+licenseNum.String)
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       "Dr. " + fullName,
			Subtitle:    strings.Join(subs, " • "),
			Description: "Doctor",
			URL:         "/doctors/" + id,
			Metadata:    map[string]any{},
		})
	}
	return results, rows.Err()
}
