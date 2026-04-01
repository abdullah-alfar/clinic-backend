package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type patientProvider struct {
	db *sql.DB
}

func NewPatientProvider(db *sql.DB) SearchProvider {
	return &patientProvider{db: db}
}

func (p *patientProvider) GetEntityType() EntityType {
	return EntityPatient
}

func (p *patientProvider) GetEntityLabel() string {
	return "Patients"
}

func (p *patientProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT id, first_name, last_name, phone, email 
		FROM patients
		WHERE tenant_id = $1 
		  AND (
		      first_name ILIKE $2 OR 
		      last_name ILIKE $2 OR 
		      phone ILIKE $2 OR 
		      email ILIKE $2
		  )
		ORDER BY last_name ASC, first_name ASC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, fName, lName string
		var phone, email *string
		if err := rows.Scan(&id, &fName, &lName, &phone, &email); err != nil {
			return nil, err
		}

		subtitle := ""
		if phone != nil && *phone != "" {
			subtitle += *phone
		}
		if email != nil && *email != "" {
			if subtitle != "" {
				subtitle += " • "
			}
			subtitle += *email
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       fName + " " + lName,
			Subtitle:    subtitle,
			Description: "Patient",
			URL:         fmt.Sprintf("/patients/%s", id),
			Score:       0, // Ranker will handle
			Metadata:    map[string]any{},
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
