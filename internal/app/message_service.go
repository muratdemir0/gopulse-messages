package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/muratdemir0/gopulse-messages/internal/adapters/webhook"
	"github.com/muratdemir0/gopulse-messages/internal/domain"
	"github.com/muratdemir0/gopulse-messages/internal/infra/cache"
)

type messageCacheData struct {
	MessageID string `json:"messageId"`
	SentAt    string `json:"sentAt"`
}

type MessageService struct {
	messageRepo   domain.MessageRepository
	webhookClient *webhook.Client
	cache         *cache.Cache
	scheduler     *Scheduler
	webhookPath   string
	logger        *slog.Logger
}

func NewMessageService(
	messageRepo domain.MessageRepository,
	webhookClient *webhook.Client,
	cache *cache.Cache,
	webhookPath string,
	logger *slog.Logger,
) *MessageService {
	service := &MessageService{
		messageRepo:   messageRepo,
		webhookClient: webhookClient,
		cache:         cache,
		webhookPath:   webhookPath,
		logger:        logger.With(slog.String("component", "message_service")),
	}

	service.scheduler = NewScheduler(2*time.Minute, service.processMessages, logger)

	return service
}

func (s *MessageService) StartAutoSending() error {
	s.scheduler.Start()
	s.logger.Info("Automatic message sending started")
	return nil
}

func (s *MessageService) StopAutoSending() error {
	s.scheduler.Stop()
	s.logger.Info("Automatic message sending stopped")
	return nil
}

func (s *MessageService) processMessages(ctx context.Context) error {
	messages, err := s.messageRepo.FindDue(ctx, 2)
	if err != nil {
		s.logger.Error("Error finding due messages", "error", err)
		return err
	}

	if len(messages) == 0 {
		s.logger.Info("No pending messages to process")
		return nil
	}

	s.logger.Info("Processing messages", "count", len(messages))

	for _, message := range messages {
		if err := s.processMessage(ctx, message); err != nil {
			s.logger.Error("Error sending message", "message_id", message.ID, "error", err)
			if err := s.messageRepo.IncrementRetry(ctx, message.ID, time.Now()); err != nil {
				s.logger.Error("Error incrementing retry for message", "message_id", message.ID, "error", err)
			}
		}
	}

	return nil
}

func (s *MessageService) processMessage(ctx context.Context, message domain.Message) error {
	webhookReq := s.buildWebhookRequest(message)

	resp, err := s.sendWebhookRequest(ctx, webhookReq)
	if err != nil {
		return s.handleSendFailure(ctx, message, err)
	}

	return s.handleSendSuccess(ctx, message, resp)
}

func (s *MessageService) buildWebhookRequest(message domain.Message) webhook.Request {
	return webhook.Request{
		To:      message.Recipient,
		Content: message.Content,
	}
}

func (s *MessageService) sendWebhookRequest(ctx context.Context, req webhook.Request) (*webhook.Response, error) {
	return s.webhookClient.Send(ctx, req, s.webhookPath)
}

func (s *MessageService) handleSendFailure(ctx context.Context, message domain.Message, sendErr error) error {
	updatedMessage := message
	updatedMessage.Status = domain.MessageStatusFailed
	updatedMessage.ErrorMessage = sql.NullString{String: sendErr.Error(), Valid: true}

	if updateErr := s.messageRepo.Update(ctx, updatedMessage); updateErr != nil {
		s.logger.Error("Error updating failed message", "message_id", message.ID, "error", updateErr)
	}

	return fmt.Errorf("webhook send failed: %w", sendErr)
}

func (s *MessageService) handleSendSuccess(ctx context.Context, message domain.Message, resp *webhook.Response) error {
	now := time.Now()

	if err := s.updateMessageAsSuccessful(ctx, message, resp, now); err != nil {
		return err
	}

	s.cacheMessageResult(ctx, message.ID, resp.MessageID, now)
	s.logger.Info("Successfully sent message", "message_id", message.ID, "recipient", message.Recipient)

	return nil
}

func (s *MessageService) updateMessageAsSuccessful(ctx context.Context, message domain.Message, resp *webhook.Response, sentAt time.Time) error {
	updatedMessage := message
	updatedMessage.Status = domain.MessageStatusSent
	updatedMessage.SentAt = sql.NullTime{Time: sentAt, Valid: true}
	updatedMessage.ResponseID = sql.NullString{String: resp.MessageID, Valid: true}

	if err := s.messageRepo.Update(ctx, updatedMessage); err != nil {
		s.logger.Error("Error updating sent message", "message_id", message.ID, "error", err)
		return fmt.Errorf("failed to update message status: %w", err)
	}

	return nil
}

func (s *MessageService) cacheMessageResult(ctx context.Context, messageID int64, responseID string, sentAt time.Time) {
	cacheKey := fmt.Sprintf("message:%d", messageID)

	data := messageCacheData{
		MessageID: responseID,
		SentAt:    sentAt.Format(time.RFC3339),
	}

	cacheData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("Error marshaling cache data for message", "message_id", messageID, "error", err)
		return
	}

	if err := s.cache.Set(ctx, cacheKey, string(cacheData)); err != nil {
		s.logger.Error("Error caching message", "message_id", messageID, "error", err)
	}
}

func (s *MessageService) GetSentMessages(ctx context.Context, limit, offset uint) ([]domain.Message, error) {
	return s.messageRepo.ListByStatus(ctx, string(domain.MessageStatusSent), limit, offset)
}
