package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/muratdemir0/gopulse-messages/internal/adapters/webhook"
	"github.com/muratdemir0/gopulse-messages/internal/domain"
	"github.com/muratdemir0/gopulse-messages/internal/infra/cache"
)

type MessageService struct {
	messageRepo   domain.MessageRepository
	webhookClient *webhook.Client
	cache  *cache.Cache
	scheduler     *Scheduler
	webhookPath   string
}

func NewMessageService(
	messageRepo domain.MessageRepository,
	webhookClient *webhook.Client,
	cache *cache.Cache,
	webhookPath string,
) *MessageService {
	service := &MessageService{
		messageRepo:   messageRepo,
		webhookClient: webhookClient,
		cache:  cache,
		webhookPath:   webhookPath,
	}

	service.scheduler = NewScheduler(2*time.Minute, service.processMessages)

	return service
}

func (s *MessageService) StartAutoSending() error {
	s.scheduler.Start()
	log.Println("Automatic message sending started")
	return nil
}

func (s *MessageService) StopAutoSending() error {
	s.scheduler.Stop()
	log.Println("Automatic message sending stopped")
	return nil
}

func (s *MessageService) processMessages(ctx context.Context) error {
	messages, err := s.messageRepo.FindDue(ctx, 2)
	if err != nil {
		log.Printf("Error finding due messages: %v", err)
		return err
	}

	if len(messages) == 0 {
		log.Println("No pending messages to process")
		return nil
	}

	log.Printf("Processing %d messages", len(messages))

	for _, message := range messages {
		if err := s.sendMessage(ctx, message); err != nil {
			log.Printf("Error sending message ID %d: %v", message.ID, err)
			if err := s.messageRepo.IncrementRetry(ctx, message.ID, time.Now()); err != nil {
				log.Printf("Error incrementing retry for message ID %d: %v", message.ID, err)
			}
		}
	}

	return nil
}

func (s *MessageService) sendMessage(ctx context.Context, message domain.Message) error {
	webhookReq := webhook.Request{
		To:      message.Recipient,
		Content: message.Content,
	}

	resp, err := s.webhookClient.Send(ctx, webhookReq, s.webhookPath)
	if err != nil {
		updatedMessage := message
		updatedMessage.Status = domain.MessageStatusFailed
		updatedMessage.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}

		if updateErr := s.messageRepo.Update(ctx, updatedMessage); updateErr != nil {
			log.Printf("Error updating failed message ID %d: %v", message.ID, updateErr)
		}

		return fmt.Errorf("webhook send failed: %w", err)
	}

	now := time.Now()
	updatedMessage := message
	updatedMessage.Status = domain.MessageStatusSent
	updatedMessage.SentAt = sql.NullTime{Time: now, Valid: true}
	updatedMessage.ResponseID = sql.NullString{String: resp.MessageID, Valid: true}

	if err := s.messageRepo.Update(ctx, updatedMessage); err != nil {
		log.Printf("Error updating sent message ID %d: %v", message.ID, err)
		return fmt.Errorf("failed to update message status: %w", err)
	}

	// Cache the sent message information
	cacheKey := fmt.Sprintf("message:%d", message.ID)
	cacheData := fmt.Sprintf(`{"messageId":"%s","sentAt":"%s"}`, resp.MessageID, now.Format(time.RFC3339))
	if err := s.cache.Set(ctx, cacheKey, cacheData); err != nil {
		log.Printf("Error caching message ID %d: %v", message.ID, err)
	}

	log.Printf("Successfully sent message ID %d to %s", message.ID, message.Recipient)
	return nil
}

func (s *MessageService) GetSentMessages(ctx context.Context, limit, offset uint) ([]domain.Message, error) {
	return s.messageRepo.ListByStatus(ctx, string(domain.MessageStatusSent), limit, offset)
}
