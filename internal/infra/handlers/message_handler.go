package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/muratdemir0/gopulse-messages/api/rest"
	"github.com/muratdemir0/gopulse-messages/internal/app"
)

type MessageHandler struct {
	service *app.MessageService
	logger  *slog.Logger
}

// StartAutoSending godoc
// @Summary Start automatic message sending
// @Description Starts the background job that automatically sends messages.
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "message: Automatic message sending started, status: active"
// @Failure 500 {object} ErrorResponse "Failed to start automatic message sending"
// @Router /messages/start [post]
func (h *MessageHandler) StartAutoSending(w http.ResponseWriter, r *http.Request) {
	if err := h.service.StartAutoSending(); err != nil {
		h.logger.Error("Failed to start automatic message sending", "error", err)
		Error(w, r, http.StatusInternalServerError, "Failed to start automatic message sending")
		return
	}

	JSON(w, r, http.StatusOK, map[string]interface{}{
		"message": "Automatic message sending started",
		"status":  "active",
	})
}

// StopAutoSending godoc
// @Summary Stop automatic message sending
// @Description Stops the background job that automatically sends messages.
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "message: Automatic message sending stopped, status: inactive"
// @Failure 500 {object} ErrorResponse "Failed to stop automatic message sending"
// @Router /messages/stop [post]
func (h *MessageHandler) StopAutoSending(w http.ResponseWriter, r *http.Request) {
	if err := h.service.StopAutoSending(); err != nil {
		h.logger.Error("Failed to stop automatic message sending", "error", err)
		Error(w, r, http.StatusInternalServerError, "Failed to stop automatic message sending")
		return
	}

	JSON(w, r, http.StatusOK, map[string]interface{}{
		"message": "Automatic message sending stopped",
		"status":  "inactive",
	})
}

// GetMessages godoc
// @Summary Get sent messages
// @Description Retrieves a list of sent messages with optional pagination.
// @Tags messages
// @Produce json
// @Param limit query int false "Number of messages to return" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} rest.MessagesListResponse
// @Failure 400 {object} ErrorResponse "Invalid limit or offset parameter"
// @Failure 500 {object} ErrorResponse "Failed to retrieve messages"
// @Router /messages [get]
func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := uint(10)
	offset := uint(0)

	if limitStr != "" {
		if l, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			limit = uint(l)
		} else {
			h.logger.Warn("Invalid limit parameter", "limit", limitStr)
			Error(w, r, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
	}

	if offsetStr != "" {
		if o, err := strconv.ParseUint(offsetStr, 10, 32); err == nil {
			offset = uint(o)
		} else {
			h.logger.Warn("Invalid offset parameter", "offset", offsetStr)
			Error(w, r, http.StatusBadRequest, "Invalid offset parameter")
			return
		}
	}

	messages, err := h.service.GetSentMessages(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to retrieve messages", "error", err)
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

func RegisterMessageHandler(mux *http.ServeMux, service *app.MessageService, logger *slog.Logger) {
	h := &MessageHandler{
		service: service,
		logger:  logger.With(slog.String("component", "message_handler")),
	}

	mux.HandleFunc("POST /messages/start", h.StartAutoSending)
	mux.HandleFunc("POST /messages/stop", h.StopAutoSending)
	mux.HandleFunc("GET /messages", h.GetMessages)
}
