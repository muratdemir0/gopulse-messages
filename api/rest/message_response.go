package rest

import (
	"time"

	"github.com/muratdemir0/gopulse-messages/internal/domain"
)

type MessageResponse struct {
	ID            int64   `json:"id"`
	Recipient     string  `json:"recipient"`
	Content       string  `json:"content"`
	Status        string  `json:"status"`
	SentAt        *string `json:"sentAt,omitempty"`
	RetryCount    int     `json:"retryCount"`
	LastAttemptAt *string `json:"lastAttemptAt,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     *string `json:"updatedAt,omitempty"`
	ResponseID    *string `json:"responseId,omitempty"`
	ResponseCode  *int64  `json:"responseCode,omitempty"`
	ErrorMessage  *string `json:"errorMessage,omitempty"`
}

type MessagesListResponse struct {
	Messages []MessageResponse `json:"messages"`
	Count    int               `json:"count"`
}

func ToMessageResponse(msg domain.Message) MessageResponse {
	resp := MessageResponse{
		ID:         msg.ID,
		Recipient:  msg.Recipient,
		Content:    msg.Content,
		Status:     string(msg.Status),
		RetryCount: msg.RetryCount,
		CreatedAt:  msg.CreatedAt.Format(time.RFC3339),
	}

	if msg.SentAt.Valid {
		sentAt := msg.SentAt.Time.Format(time.RFC3339)
		resp.SentAt = &sentAt
	}

	if msg.LastAttemptAt.Valid {
		lastAttemptAt := msg.LastAttemptAt.Time.Format(time.RFC3339)
		resp.LastAttemptAt = &lastAttemptAt
	}

	if msg.UpdatedAt.Valid {
		updatedAt := msg.UpdatedAt.Time.Format(time.RFC3339)
		resp.UpdatedAt = &updatedAt
	}

	if msg.ResponseID.Valid {
		resp.ResponseID = &msg.ResponseID.String
	}

	if msg.ResponseCode.Valid {
		resp.ResponseCode = &msg.ResponseCode.Int64
	}

	if msg.ErrorMessage.Valid {
		resp.ErrorMessage = &msg.ErrorMessage.String
	}

	return resp
}

func ToMessageResponses(messages []domain.Message) []MessageResponse {
	responses := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		responses[i] = ToMessageResponse(msg)
	}
	return responses
}
