package procedurecatalog

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	CreateProcedure(ctx context.Context, tenantID uuid.UUID, req CreateProcedureReq) (*ProcedureCatalog, error)
	GetProcedure(ctx context.Context, tenantID, procID uuid.UUID) (*ProcedureCatalog, error)
	ListProcedures(ctx context.Context, tenantID uuid.UUID) ([]ProcedureCatalog, error)
	UpdateProcedure(ctx context.Context, tenantID, procID uuid.UUID, req UpdateProcedureReq) (*ProcedureCatalog, error)
}

type procedureService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &procedureService{repo: repo}
}

func (s *procedureService) CreateProcedure(ctx context.Context, tenantID uuid.UUID, req CreateProcedureReq) (*ProcedureCatalog, error) {
	proc := &ProcedureCatalog{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
	}

	if err := s.repo.CreateProcedure(ctx, proc); err != nil {
		return nil, err
	}

	if len(req.Items) > 0 {
		if err := s.repo.SetProcedureItems(ctx, proc.ID, req.Items); err != nil {
			return nil, err
		}
	}

	return s.repo.GetProcedureByID(ctx, tenantID, proc.ID)
}

func (s *procedureService) GetProcedure(ctx context.Context, tenantID, procID uuid.UUID) (*ProcedureCatalog, error) {
	return s.repo.GetProcedureByID(ctx, tenantID, procID)
}

func (s *procedureService) ListProcedures(ctx context.Context, tenantID uuid.UUID) ([]ProcedureCatalog, error) {
	return s.repo.ListProcedures(ctx, tenantID)
}

func (s *procedureService) UpdateProcedure(ctx context.Context, tenantID, procID uuid.UUID, req UpdateProcedureReq) (*ProcedureCatalog, error) {
	proc, err := s.repo.GetProcedureByID(ctx, tenantID, procID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		proc.Name = *req.Name
	}
	if req.Description != nil {
		proc.Description = req.Description
	}
	if req.IsActive != nil {
		proc.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateProcedure(ctx, proc); err != nil {
		return nil, err
	}

	if req.Items != nil {
		if err := s.repo.SetProcedureItems(ctx, proc.ID, req.Items); err != nil {
			return nil, err
		}
	}

	return s.repo.GetProcedureByID(ctx, tenantID, proc.ID)
}
