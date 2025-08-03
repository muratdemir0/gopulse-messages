package handlers

import "net/http"

type HealthHandler struct {
}

// ServeHTTP godoc
// @Summary Health Check
// @Description Check if the service is up and running.
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "{"status": "ok"}"
// @Router /health [get]
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	JSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func RegisterHealthHandler(mux *http.ServeMux) {
	mux.Handle("/health", &HealthHandler{})
}
