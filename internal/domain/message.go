package domain

import (
	"context"
	"database/sql"
	"time"
)

type MessageStatus string

const (
	MessageStatusPending MessageStatus = "pending"
	MessageStatusSent    MessageStatus = "sent"
	MessageStatusFailed  MessageStatus = "failed"
)

type Message struct {
	ID            int64          `db:"id"`
	Recipient     string         `db:"recipient"`
	Content       string         `db:"content"`
	Status        MessageStatus  `db:"status"`
	SentAt        sql.NullTime   `db:"sent_at"`
	RetryCount    int            `db:"retry_count"`
	LastAttemptAt sql.NullTime   `db:"last_attempt_at"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     sql.NullTime   `db:"updated_at"`
	ResponseID    sql.NullString `db:"response_id"`
	ResponseCode  sql.NullInt64  `db:"response_code"`
	ErrorMessage  sql.NullString `db:"error_message"`
}

type MessageRepository interface {
	Update(ctx context.Context, message Message) error
	GetAll(ctx context.Context) ([]Message, error)
	FindDue(ctx context.Context, limit uint) ([]Message, error)
	IncrementRetry(ctx context.Context, id int64, attemptTime time.Time) error
	ListByStatus(ctx context.Context, status string, limit, offset uint) ([]Message, error)
}
