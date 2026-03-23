package http

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Data    any        `json:"data"`
	Message string     `json:"message"`
	Error   *APIError  `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

func RespondJSON(w http.ResponseWriter, status int, data any, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Data:    data,
		Message: message,
		Error:   nil,
	})
}

func RespondError(w http.ResponseWriter, status int, message string, code string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Data:    nil,
		Message: message,
		Error: &APIError{
			Code:    code,
			Details: details,
		},
	})
}
