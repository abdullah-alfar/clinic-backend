package search

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type SearchService interface {
	GlobalSearch(ctx context.Context, tenantID uuid.UUID, query string) (SearchData, error)
}

type searchService struct {
	repo SearchRepository
}

func NewSearchService(repo SearchRepository) SearchService {
	return &searchService{repo: repo}
}

func (s *searchService) GlobalSearch(ctx context.Context, tenantID uuid.UUID, query string) (SearchData, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return SearchData{
			Patients: []PatientSearchResult{},
			Doctors:  []any{},
			Reports:  []any{},
		}, nil
	}

	limitPerType := 20

	patients, err := s.repo.SearchPatients(ctx, tenantID, query, limitPerType)
	if err != nil {
		return SearchData{}, err
	}

	return SearchData{
		Patients: patients,
		Doctors:  []any{}, // Future: s.repo.SearchDoctors(...)
		Reports:  []any{}, // Future: s.repo.SearchReports(...)
	}, nil
}
