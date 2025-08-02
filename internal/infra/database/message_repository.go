package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/db"
	"github.com/muratdemir0/gopulse-messages/internal/domain"
)

const (
	tableName     = "messages"
	maxRetryCount = 5
)

type MessageRepository struct {
	db *db.Client
}

func NewMessageRepository(db *db.Client) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Update(ctx context.Context, message domain.Message) error {
	updatedAt := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	record := goqu.Record{
		"status":     message.Status,
		"updated_at": updatedAt,
	}

	ds := goqu.Update(tableName).
		Set(record).
		Where(goqu.Ex{
			"id":     message.ID,
			"status": domain.MessageStatusPending,
		})

	result, err := r.db.Update(ctx, ds)
	if err != nil {
		return fmt.Errorf("error updating message id %d: %w", message.ID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrMessageNotFound
	}

	return nil
}

func (r *MessageRepository) GetAll(ctx context.Context) ([]domain.Message, error) {
	ds := goqu.Select("*").From(tableName).Order(goqu.C("created_at").Desc())

	var messages []domain.Message
	err := r.db.Select(ctx, &messages, ds)
	if err != nil {
		return nil, fmt.Errorf("error getting all messages: %w", err)
	}

	return messages, nil
}

func (r *MessageRepository) FindDue(ctx context.Context, limit uint) ([]domain.Message, error) {
	ds := goqu.From(tableName).
		Where(
			goqu.C("status").Eq(domain.MessageStatusPending),
			goqu.C("retry_count").Lt(maxRetryCount),
		).
		Order(goqu.C("created_at").Asc()).
		Limit(limit)

	var messages []domain.Message
	err := r.db.Select(ctx, &messages, ds)
	if err != nil {
		return nil, fmt.Errorf("error finding due messages: %w", err)
	}
	return messages, nil
}

func (r *MessageRepository) IncrementRetry(ctx context.Context, id int64, attemptTime time.Time) error {
	ds := goqu.Update(tableName).
		Set(
			goqu.Record{
				"last_attempt_at": attemptTime,
				"retry_count":     goqu.L("retry_count + 1"),
			},
		).
		Where(goqu.Ex{"id": id})

	_, err := r.db.Update(ctx, ds)
	if err != nil {
		return fmt.Errorf("error incrementing retry for message id %d: %w", id, err)
	}
	return nil
}

func (r *MessageRepository) ListByStatus(ctx context.Context, status string, limit, offset uint) ([]domain.Message, error) {
	ds := goqu.From(tableName).
		Where(goqu.C("status").Eq(status)).
		Order(goqu.C("created_at").Desc()).
		Limit(limit).
		Offset(offset)

	var messages []domain.Message
	err := r.db.Select(ctx, &messages, ds)
	if err != nil {
		return nil, fmt.Errorf("error listing messages by status %s: %w", status, err)
	}
	return messages, nil
}
