package whatsappbot

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BotRepository stores bot sessions and logs all messages.
type BotRepository interface {
	GetSession(ctx context.Context, tenantID uuid.UUID, phone string) (*BotSession, error)
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*BotSession, error)
	UpsertSession(ctx context.Context, s *BotSession) error
	LogMessage(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, phone, direction, msgType, content, providerID string) error
	FindPatientByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*uuid.UUID, error)
	GetPatientMessages(ctx context.Context, tenantID, patientID uuid.UUID) ([]whatsapp_message_model, error)
	GetBotStatus(ctx context.Context, tenantID, patientID uuid.UUID) (*BotStatusDTO, error)
}

type whatsapp_message_model struct {
	ID                uuid.UUID
	Direction         string
	PhoneNumber       string
	MessageType       string
	Content           string
	ProviderMessageID sql.NullString
	CreatedAt         time.Time
}

type BotStatusDTO struct {
	IsReady         bool
	PhoneNumber     sql.NullString
	LastInteraction sql.NullTime
	OptInStatus     bool
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
func (r *postgresBotRepository) GetPatientMessages(ctx context.Context, tenantID, patientID uuid.UUID) ([]whatsapp_message_model, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, direction, phone_number, message_type, content, provider_message_id, created_at
		FROM whatsapp_messages
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
		LIMIT 50
	`, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []whatsapp_message_model
	for rows.Next() {
		var m whatsapp_message_model
		if err := rows.Scan(&m.ID, &m.Direction, &m.PhoneNumber, &m.MessageType, &m.Content, &m.ProviderMessageID, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (r *postgresBotRepository) GetBotStatus(ctx context.Context, tenantID, patientID uuid.UUID) (*BotStatusDTO, error) {
	var status BotStatusDTO
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			p.phone IS NOT NULL as is_ready,
			p.phone,
			s.last_message_at,
			COALESCE(pref.whatsapp_enabled, false) as opt_in_status
		FROM patients p
		LEFT JOIN whatsapp_bot_sessions s ON s.patient_id = p.id AND s.tenant_id = p.tenant_id
		LEFT JOIN patient_notification_preferences pref ON pref.patient_id = p.id AND pref.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.id = $2
	`, tenantID, patientID).Scan(&status.IsReady, &status.PhoneNumber, &status.LastInteraction, &status.OptInStatus)
	
	if err != nil {
		return nil, err
	}
	return &status, nil
}
