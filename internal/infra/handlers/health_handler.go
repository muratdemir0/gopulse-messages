package handlers

import "net/http"

type HealthHandler struct {
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	JSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func RegisterHealthHandler(mux *http.ServeMux) {
	mux.Handle("/health", &HealthHandler{})
}
