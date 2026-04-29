package ai_agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"clinic-backend/internal/ai"
	"clinic-backend/internal/settings"
)

type AgentService interface {
	ProcessChat(ctx context.Context, req ChatRequest, tenantID string) (ChatResponse, error)
	ProcessConfirmation(ctx context.Context, req ConfirmRequest, tenantID string) (ConfirmResponse, error)
}

type agentServiceImpl struct {
	settingsRepo settings.Repository
	systemTools  *SystemTools
	memory       MemoryManager
	confirmStore ConfirmationStore
}

func NewAgentService(st *SystemTools, mem MemoryManager, sRepo settings.Repository, cs ConfirmationStore) AgentService {
	return &agentServiceImpl{
		systemTools:  st,
		memory:       mem,
		settingsRepo: sRepo,
		confirmStore: cs,
	}
}

func (s *agentServiceImpl) ProcessChat(ctx context.Context, req ChatRequest, tenantID string) (ChatResponse, error) {
	tID, err := uuid.Parse(tenantID)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("invalid tenant UUID: %w", err)
	}

	cfg, err := s.settingsRepo.GetByTenantID(tID)
	if err != nil {
		if err == settings.ErrNotFound {
			return ChatResponse{}, fmt.Errorf("ai processing disabled: no configuration found for tenant")
		}
		return ChatResponse{}, fmt.Errorf("failed to fetch AI config: %w", err)
	}

	if !cfg.AIEnabled {
		return ChatResponse{Message: "AI intelligence is currently disabled in your system control panel."}, nil
	}

	apiKey, err := settings.Decrypt(cfg.AIAPIKey)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to decrypt AI API key: %w", err)
	}

	provider, err := ai.NewProvider(cfg.AIProvider, apiKey)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to instantiate AI provider: %w", err)
	}

	promptBuilder := NewPromptBuilder(s.systemTools)
	orchestrator := NewOrchestrator(provider, s.memory, s.systemTools, promptBuilder, s.confirmStore)

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	aiReq := AIRequest{
		TenantID:  tenantID,
		SessionID: req.SessionID,
		Input:     req.Message,
		Context:   req.Context,
	}

	if pid, ok := req.Context["patient_id"].(string); ok {
		aiReq.PatientID = pid
	}

	resp, err := orchestrator.Orchestrate(ctx, aiReq)
	if err != nil {
		return ChatResponse{}, err
	}

	return ChatResponse{
		Message:              resp.Message,
		RequiresConfirmation: resp.RequiresConfirmation,
		Action:               resp.Action,
	}, nil
}

func (s *agentServiceImpl) ProcessConfirmation(ctx context.Context, req ConfirmRequest, tenantID string) (ConfirmResponse, error) {
	action, err := s.confirmStore.Get(req.Token)
	if err != nil {
		return ConfirmResponse{}, err
	}

	if action.Confirmed {
		return ConfirmResponse{}, ErrActionAlreadyDone
	}

	tool, exists := s.systemTools.GetTool(action.Type)
	if !exists {
		return ConfirmResponse{}, fmt.Errorf("tool %s not found", action.Type)
	}

	resultMsg, err := tool.ExecuteConfirmed(ctx, tenantID, action.Payload)
	if err != nil {
		return ConfirmResponse{}, fmt.Errorf("tool execution failed: %w", err)
	}

	if err := s.confirmStore.MarkConfirmed(req.Token); err != nil {
		return ConfirmResponse{}, err
	}

	return ConfirmResponse{
		Message: resultMsg,
	}, nil
}
