package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type SearchRepository interface {
	SearchPatients(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]PatientSearchResult, error)
}

type postgresSearchRepository struct {
	db *sql.DB
}

func NewPostgresSearchRepository(db *sql.DB) SearchRepository {
	return &postgresSearchRepository{db: db}
}

func (r *postgresSearchRepository) SearchPatients(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]PatientSearchResult, error) {
	// If query is empty, we return empty instead of doing heavy table scan
	if query == "" {
		return []PatientSearchResult{}, nil
	}

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

	rows, err := r.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PatientSearchResult
	for rows.Next() {
		var p PatientSearchResult
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Phone, &p.Email); err != nil {
			return nil, err
		}
		results = append(results, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if results == nil {
		results = []PatientSearchResult{} // ensure not nil for JSON response
	}

	return results, nil
}
