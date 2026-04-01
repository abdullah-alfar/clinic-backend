package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type invoiceProvider struct {
	db *sql.DB
}

func NewInvoiceProvider(db *sql.DB) SearchProvider {
	return &invoiceProvider{db: db}
}

func (p *invoiceProvider) GetEntityType() EntityType {
	return EntityInvoice
}

func (p *invoiceProvider) GetEntityLabel() string {
	return "Invoices"
}

func (p *invoiceProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			i.id, 
			i.status, 
			i.amount, 
			pt.first_name, 
			pt.last_name
		FROM invoices i
		JOIN patients pt ON i.patient_id = pt.id
		WHERE i.tenant_id = $1 
		  AND (
		      i.status ILIKE $2 OR 
		      pt.first_name ILIKE $2 OR 
		      pt.last_name ILIKE $2 OR
			  CAST(i.amount AS TEXT) ILIKE $2
		  )
		ORDER BY i.created_at DESC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id string
		var status string
		var amount float64
		var fName, lName string
		if err := rows.Scan(&id, &status, &amount, &fName, &lName); err != nil {
			return nil, err
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       fmt.Sprintf("Invoice for %s %s", fName, lName),
			Subtitle:    fmt.Sprintf("Amount: $%.2f • Status: %s", amount, status),
			Description: "Billing",
			URL:         fmt.Sprintf("/patients/%s?tab=billing", id), // Assuming it links to patient's billing tab
			Score:       0,
			Metadata: map[string]any{
				"status": status,
				"amount": amount,
			},
		})
	}

	return results, nil
}
