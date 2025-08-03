package handlers

import "net/http"

type HealthHandler struct {
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func RegisterHealthHandler(mux *http.ServeMux) {
	mux.Handle("/health", &HealthHandler{})
}
