package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var SecretKey = []byte("super-secret-key-replace-in-production")

// ErrTokenExpired is returned by ValidateToken when the JWT signature is valid
// but the token's expiry time has passed. Callers (middleware, handlers) can
// distinguish this from a structurally invalid or tampered token and respond
// with the correct error code (TOKEN_EXPIRED vs INVALID_TOKEN).
var ErrTokenExpired = errors.New("token is expired")

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func GenerateTokenPairs(userID, tenantID uuid.UUID, role string) (string, string, error) {
	// 1 Hour Access Token
	accessClaims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(SecretKey)
	if err != nil {
		return "", "", err
	}

	// 7 Days Refresh Token (stored in DB for rotation; JWT format for consistency)
	refreshClaims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(SecretKey)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ValidateToken parses and validates tokenStr against SecretKey.
//
// Error contract:
//   - Returns (claims, nil)           → token is valid and unexpired.
//   - Returns (nil, ErrTokenExpired)  → valid signature but past ExpiresAt.
//   - Returns (nil, err)              → structurally invalid, tampered, or wrong algorithm.
//
// Callers MUST check errors.Is(err, ErrTokenExpired) before treating a failure
// as a security violation — an expired token is a normal lifecycle event, not an attack.
func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})

	if err != nil {
		// Detect expiry specifically so middleware can return TOKEN_EXPIRED instead
		// of the generic INVALID_TOKEN code. The golang-jwt library wraps the
		// underlying cause in a join error, so errors.Is is the correct check.
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
