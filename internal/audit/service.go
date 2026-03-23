package audit

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

type AuditService struct {
	db *sql.DB
}

func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{db: db}
}

// LogAction async logs an operation to the audit_logs table
func (s *AuditService) LogAction(tenantID, userID uuid.UUID, action, entityType string, entityID uuid.UUID, metadata any) {
	// Execute in a goroutine so auditing doesn't block the caller
	go func() {
		var metaBytes []byte
		if metadata != nil {
			metaBytes, _ = json.Marshal(metadata)
		}

		_, _ = s.db.Exec(`
			INSERT INTO audit_logs (id, tenant_id, user_id, action, entity_type, entity_id, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, uuid.New(), tenantID, userID, action, entityType, entityID, metaBytes)
	}()
}
