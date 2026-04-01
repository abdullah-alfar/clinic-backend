package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type doctorProvider struct {
	db *sql.DB
}

func NewDoctorProvider(db *sql.DB) SearchProvider {
	return &doctorProvider{db: db}
}

func (p *doctorProvider) GetEntityType() EntityType {
	return EntityDoctor
}

func (p *doctorProvider) GetEntityLabel() string {
	return "Doctors"
}

func (p *doctorProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT id, full_name, specialty, license_number
		FROM doctors
		WHERE tenant_id = $1 
		  AND (
		      full_name ILIKE $2 OR 
		      specialty ILIKE $2 OR 
		      license_number ILIKE $2
		  )
		ORDER BY full_name ASC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, fullName string
		var specialty, licenseNum sql.NullString
		if err := rows.Scan(&id, &fullName, &specialty, &licenseNum); err != nil {
			return nil, err
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
			URL:         fmt.Sprintf("/doctors/%s", id),
			Score:       0,
			Metadata:    map[string]any{},
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
