package ai_core

import (
	"context"
	"fmt"
	"time"

	"clinic-backend/internal/ai"
	"clinic-backend/internal/settings"
)

// AIService is the public entrypoint for all intelligence orchestration across the SaaS.
type AIService interface {
	Process(ctx context.Context, req AIRequest) (AIResponse, error)
}

type aiServiceImpl struct {
	settingsRepo settings.Repository
	systemTools  *SystemTools
	memory       MemoryManager
}

// NewAIService returns the fully configured intelligent core.
func NewAIService(st *SystemTools, mem MemoryManager, sRepo settings.Repository) AIService {
	return &aiServiceImpl{
		systemTools:  st,
		memory:       mem,
		settingsRepo: sRepo,
	}
}

func (s *aiServiceImpl) Process(ctx context.Context, req AIRequest) (AIResponse, error) {
	// 1. Fetch Tenant AI Configuration
	cfg, err := s.settingsRepo.GetByTenantID(req.TenantID)
	if err != nil {
		if err == settings.ErrNotFound {
			return AIResponse{}, fmt.Errorf("ai processing disabled: no configuration found for tenant")
		}
		return AIResponse{}, fmt.Errorf("failed to fetch AI config: %w", err)
	}

	if !cfg.AIEnabled {
		return AIResponse{Message: "AI intelligence is currently disabled in your system control panel."}, nil
	}

	// 2. Initialize Provider dynamically based on Tenant settings
	apiKey, err := settings.Decrypt(cfg.AIAPIKey)
	if err != nil {
		return AIResponse{}, fmt.Errorf("failed to decrypt AI API key: %w", err)
	}
	
	provider, err := ai.NewProvider(cfg.AIProvider, apiKey)
	if err != nil {
		return AIResponse{}, fmt.Errorf("failed to instantiate AI provider: %w", err)
	}

	// 3. Setup Orchestrator for this Request
	promptBuilder := NewPromptBuilder(s.systemTools)
	orchestrator := NewOrchestrator(provider, s.memory, s.systemTools, promptBuilder)

	// Contextual fallback timeout (override if ctx has none)
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// 4. Run the ReAct loop
	return orchestrator.Orchestrate(ctx, req)
}
