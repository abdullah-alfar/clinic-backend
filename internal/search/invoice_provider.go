package search

import (
	"context"
	"database/sql"
	"fmt"
)

type invoiceProvider struct{ db *sql.DB }

// NewInvoiceProvider creates a SearchProvider that searches invoices.
func NewInvoiceProvider(db *sql.DB) SearchProvider { return &invoiceProvider{db: db} }

func (p *invoiceProvider) Type() EntityType { return EntityInvoice }
func (p *invoiceProvider) Label() string    { return "Invoices" }

func (p *invoiceProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

	if req.Status != "" {
		args = append(args, req.Status)
		extra += fmt.Sprintf(" AND i.status = $%d", len(args))
	}
	if req.PatientID != nil {
		args = append(args, *req.PatientID)
		extra += fmt.Sprintf(" AND i.patient_id = $%d", len(args))
	}
	if req.DateFrom != nil {
		args = append(args, *req.DateFrom)
		extra += fmt.Sprintf(" AND i.created_at >= $%d", len(args))
	}
	if req.DateTo != nil {
		args = append(args, *req.DateTo)
		extra += fmt.Sprintf(" AND i.created_at <= $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT
			i.id,
			i.status,
			i.amount,
			pt.first_name,
			pt.last_name,
			i.patient_id
		FROM invoices i
		JOIN patients pt ON i.patient_id = pt.id
		WHERE i.tenant_id = $1
		  AND (
		        i.status ILIKE $2 OR
		        pt.first_name ILIKE $2 OR
		        pt.last_name  ILIKE $2 OR
		        (pt.first_name || ' ' || pt.last_name) ILIKE $2 OR
		        CAST(i.amount AS TEXT) ILIKE $2
		      )
		%s
		ORDER BY i.created_at DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("invoices: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, status, patientID string
		var amount float64
		var fName, lName string
		if err := rows.Scan(&id, &status, &amount, &fName, &lName, &patientID); err != nil {
			return nil, fmt.Errorf("invoices scan: %w", err)
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       fmt.Sprintf("Invoice for %s %s", fName, lName),
			Subtitle:    fmt.Sprintf("$%.2f • %s", amount, status),
			Description: "Billing",
			URL:         fmt.Sprintf("/patients/%s?tab=billing", patientID),
			Metadata: map[string]any{
				"status": status,
				"amount": amount,
			},
		})
	}
	return results, rows.Err()
}
