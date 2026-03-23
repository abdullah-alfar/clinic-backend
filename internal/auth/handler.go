package auth

import (
	"encoding/json"
	"net/http"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/jwt"
	"clinic-backend/internal/shared"
)

type AuthHandler struct {
	svc *AuthService
}

func NewAuthHandler(svc *AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		TenantID string `json:"tenant_id"`
	} `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	user, err := h.svc.Authenticate(req.Email, req.Password)
	if err != nil {
		if err == ErrInvalidCredentials || err == ErrUserInactive {
			myhttp.RespondError(w, http.StatusUnauthorized, "authentication failed", "AUTH_FAILED", err.Error())
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		return
	}

	accessToken, refreshToken, err := jwt.GenerateTokenPairs(user.ID, user.TenantID, user.Role)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to generate tokens", "TOKEN_ERROR", nil)
		return
	}

	// Store refresh token with 7 days expiry
	h.svc.StoreRefreshToken(user.ID, refreshToken, time.Now().Add(7*24*time.Hour))

	res := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Role     string `json:"role"`
			TenantID string `json:"tenant_id"`
		}{
			ID:       user.ID.String(),
			Name:     user.Name,
			Role:     user.Role,
			TenantID: user.TenantID.String(),
		},
	}

	myhttp.RespondJSON(w, http.StatusOK, res, "login successful")
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "missing context", "UNAUTHORIZED", nil)
		return
	}

	user, err := h.svc.GetUserByID(userCtx.UserID, userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusNotFound, "user not found", "NOT_FOUND", nil)
		return
	}

	res := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"role":      user.Role,
		"tenant_id": user.TenantID,
	}

	myhttp.RespondJSON(w, http.StatusOK, res, "success")
}

func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	user, err := h.svc.ConsumeRefreshToken(req.RefreshToken)
	if err != nil {
		myhttp.RespondError(w, http.StatusUnauthorized, "invalid or expired refresh token", "AUTH_FAILED", err.Error())
		return
	}

	accessToken, newRefreshToken, err := jwt.GenerateTokenPairs(user.ID, user.TenantID, user.Role)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to generate token", "TOKEN_ERROR", nil)
		return
	}

	// Store new rotating token
	h.svc.StoreRefreshToken(user.ID, newRefreshToken, time.Now().Add(7*24*time.Hour))

	res := map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	}
	myhttp.RespondJSON(w, http.StatusOK, res, "refresh successful")
}
