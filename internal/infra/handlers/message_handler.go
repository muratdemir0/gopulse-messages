package handlers

import (
	"net/http"
	"strconv"

	"github.com/muratdemir0/gopulse-messages/api/rest"
	"github.com/muratdemir0/gopulse-messages/internal/app"
)

type MessageHandler struct {
	service *app.MessageService
}

func (h *MessageHandler) StartAutoSending(w http.ResponseWriter, r *http.Request) {
	if err := h.service.StartAutoSending(); err != nil {
		Error(w, r, http.StatusInternalServerError, "Failed to start automatic message sending")
		return
	}

	JSON(w, r, http.StatusOK, map[string]interface{}{
		"message": "Automatic message sending started",
		"status":  "active",
	})
}

func (h *MessageHandler) StopAutoSending(w http.ResponseWriter, r *http.Request) {
	if err := h.service.StopAutoSending(); err != nil {
		Error(w, r, http.StatusInternalServerError, "Failed to stop automatic message sending")
		return
	}

	JSON(w, r, http.StatusOK, map[string]interface{}{
		"message": "Automatic message sending stopped",
		"status":  "inactive",
	})
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := uint(10)
	offset := uint(0)

	if limitStr != "" {
		if l, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			limit = uint(l)
		} else {
			Error(w, r, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
	}

	if offsetStr != "" {
		if o, err := strconv.ParseUint(offsetStr, 10, 32); err == nil {
			offset = uint(o)
		} else {
			Error(w, r, http.StatusBadRequest, "Invalid offset parameter")
			return
		}
	}

	messages, err := h.service.GetSentMessages(r.Context(), limit, offset)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, "Failed to retrieve messages")
		return
	}

	messageResponses := rest.ToMessageResponses(messages)
	response := rest.MessagesListResponse{
		Messages: messageResponses,
		Count:    len(messageResponses),
	}

	JSON(w, r, http.StatusOK, response)
}

func RegisterMessageHandler(mux *http.ServeMux, service *app.MessageService) {
	h := &MessageHandler{service: service}

	mux.HandleFunc("POST /messages/start", h.StartAutoSending)
	mux.HandleFunc("POST /messages/stop", h.StopAutoSending)
	mux.HandleFunc("GET /messages", h.GetMessages)
}
