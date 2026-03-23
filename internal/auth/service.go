package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{db: db}
}

type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
}

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrUserInactive = errors.New("user account is inactive")

func (s *AuthService) Authenticate(email, password string) (*User, error) {
	var u User
	err := s.db.QueryRow(`
		SELECT id, tenant_id, name, email, password_hash, role, is_active 
		FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !u.IsActive {
		return nil, ErrUserInactive
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return &u, nil
}

func (s *AuthService) GetUserByID(id uuid.UUID, tenantID uuid.UUID) (*User, error) {
	var u User
	err := s.db.QueryRow(`
		SELECT id, tenant_id, name, email, role, is_active 
		FROM users WHERE id = $1 AND tenant_id = $2`, id, tenantID).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Role, &u.IsActive)

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *AuthService) StoreRefreshToken(userID uuid.UUID, token string, expiresAt time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), userID, token, expiresAt)
	return err
}

func (s *AuthService) ConsumeRefreshToken(token string) (*User, error) {
	// First fetch the token
	var userID uuid.UUID
	var expiresAt time.Time
	err := s.db.QueryRow(`
		SELECT user_id, expires_at FROM refresh_tokens WHERE token = $1
	`, token).Scan(&userID, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("refresh token not found or already consumed")
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Delete from DB (one-time use rotation)
	s.db.Exec("DELETE FROM refresh_tokens WHERE token = $1", token)

	// Fetch user
	var u User
	err = s.db.QueryRow(`
		SELECT id, tenant_id, name, email, role, is_active 
		FROM users WHERE id = $1`, userID).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Role, &u.IsActive)

	if err != nil {
		return nil, err
	}
	return &u, nil
}
