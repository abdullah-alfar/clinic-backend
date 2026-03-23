package tenant

import (
	"database/sql"

	"github.com/google/uuid"
)

type Tenant struct {
	ID             uuid.UUID `json:"id"`
	Subdomain      string    `json:"subdomain"`
	Name           string    `json:"name"`
	LogoURL        *string   `json:"logo_url"`
	PrimaryColor   *string   `json:"primary_color"`
	SecondaryColor *string   `json:"secondary_color"`
	BorderRadius   *string   `json:"border_radius"`
	FontFamily     *string   `json:"font_family"`
}

type TenantService struct {
	db *sql.DB
}

func NewTenantService(db *sql.DB) *TenantService {
	return &TenantService{db: db}
}

func (s *TenantService) GetTenantBySubdomain(subdomain string) (*Tenant, error) {
	var t Tenant
	err := s.db.QueryRow(`
		SELECT id, subdomain, name, logo_url, primary_color, secondary_color, border_radius, font_family
		FROM tenants WHERE subdomain = $1`, subdomain).
		Scan(&t.ID, &t.Subdomain, &t.Name, &t.LogoURL, &t.PrimaryColor, &t.SecondaryColor, &t.BorderRadius, &t.FontFamily)
	
	if err != nil {
		return nil, err
	}
	return &t, nil
}
