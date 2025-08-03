package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Path    string `json:"path"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Default().ErrorContext(r.Context(), "failed to encode success response", slog.String("error", err.Error()))
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	ErrorWithCode(w, r, status, message, "")
}

func ErrorWithCode(w http.ResponseWriter, r *http.Request, status int, message string, code string) {
	resp := ErrorResponse{
		Status:  status,
		Message: message,
		Code:    code,
		Path:    r.URL.Path,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Default().ErrorContext(r.Context(), "failed to encode error response", slog.String("error", err.Error()))
	}
}