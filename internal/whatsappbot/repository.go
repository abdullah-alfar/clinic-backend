package whatsappbot

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// BotRepository stores bot sessions and logs all messages.
type BotRepository interface {
	GetSession(ctx context.Context, tenantID uuid.UUID, phone string) (*BotSession, error)
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*BotSession, error)
	UpsertSession(ctx context.Context, s *BotSession) error
	LogMessage(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, phone, direction, msgType, content, providerID string) error
	FindPatientByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*uuid.UUID, error)
}

type postgresBotRepository struct {
	db *sql.DB
}

func NewPostgresBotRepository(db *sql.DB) BotRepository {
	return &postgresBotRepository{db: db}
}

func (r *postgresBotRepository) GetSession(ctx context.Context, tenantID uuid.UUID, phone string) (*BotSession, error) {
	var s BotSession
	var stateBytes []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, patient_id, phone_number, current_flow, current_step, state, last_message_at, created_at, updated_at
		FROM whatsapp_bot_sessions
		WHERE tenant_id = $1 AND phone_number = $2
	`, tenantID, phone).Scan(
		&s.ID, &s.TenantID, &s.PatientID, &s.PhoneNumber, &s.CurrentFlow, &s.CurrentStep, &stateBytes, &s.LastMessageAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.State = make(StateData)
	json.Unmarshal(stateBytes, &s.State)
	return &s, nil
}

func (r *postgresBotRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*BotSession, error) {
	var s BotSession
	var stateBytes []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, patient_id, phone_number, current_flow, current_step, state, last_message_at, created_at, updated_at
		FROM whatsapp_bot_sessions
		WHERE id = $1
	`, sessionID).Scan(
		&s.ID, &s.TenantID, &s.PatientID, &s.PhoneNumber, &s.CurrentFlow, &s.CurrentStep, &stateBytes, &s.LastMessageAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.State = make(StateData)
	json.Unmarshal(stateBytes, &s.State)
	return &s, nil
}

func (r *postgresBotRepository) UpsertSession(ctx context.Context, s *BotSession) error {
	stateBytes, _ := json.Marshal(s.State)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO whatsapp_bot_sessions (id, tenant_id, patient_id, phone_number, current_flow, current_step, state, last_message_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW())
		ON CONFLICT (tenant_id, phone_number) DO UPDATE SET
			patient_id = EXCLUDED.patient_id,
			current_flow = EXCLUDED.current_flow,
			current_step = EXCLUDED.current_step,
			state = EXCLUDED.state,
			last_message_at = NOW(),
			updated_at = NOW()
	`, s.ID, s.TenantID, s.PatientID, s.PhoneNumber, s.CurrentFlow, s.CurrentStep, stateBytes)
	return err
}

func (r *postgresBotRepository) LogMessage(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, phone, direction, msgType, content, providerID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO whatsapp_messages (id, tenant_id, patient_id, direction, phone_number, message_type, content, provider_message_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, uuid.New(), tenantID, patientID, direction, phone, msgType, content, providerID)
	return err
}

func (r *postgresBotRepository) FindPatientByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM patients WHERE tenant_id = $1 AND phone = $2 LIMIT 1
	`, tenantID, phone).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil // Not found, but no system error
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
