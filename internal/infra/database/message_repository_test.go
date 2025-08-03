//go:build integration

package database_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/db"
	"github.com/muratdemir0/gopulse-messages/internal/domain"
	"github.com/muratdemir0/gopulse-messages/internal/infra/database"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	messageRepo *database.MessageRepository
	dbClient    *db.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %s", err)
	}

	dbClient = db.NewDB(connStr)
	defer func() {
		if err := dbClient.Close(); err != nil {
			log.Fatalf("failed to close database client: %s", err)
		}
	}()

	migration, err := os.ReadFile("../../../migrations/000001_message.up.sql")
	if err != nil {
		log.Fatalf("failed to read migration file: %s", err)
	}

	_, err = dbClient.Goqu.Exec(string(migration))
	if err != nil {
		log.Fatalf("failed to apply migration: %s", err)
	}

	messageRepo = database.NewMessageRepository(dbClient)

	os.Exit(m.Run())
}

func cleanup(t *testing.T) {
	t.Helper()
	ds := goqu.Delete("messages")
	_, err := dbClient.Delete(context.Background(), ds)
	if err != nil {
		t.Fatalf("failed to cleanup messages table: %v", err)
	}
}

func createMessage(t *testing.T, msg *domain.Message) int64 {
	t.Helper()
	query := goqu.Insert("messages").Rows(goqu.Record{
		"recipient": msg.Recipient,
		"content":   msg.Content,
		"status":    msg.Status,
	}).Returning("id")

	var id int64
	q, args, _ := query.ToSQL()
	err := dbClient.Goqu.QueryRow(q, args...).Scan(&id)

	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}
	return id
}

func TestMessageRepository_Update(t *testing.T) {
	defer cleanup(t)
	ctx := context.Background()

	msg := &domain.Message{
		Recipient: "1234567890",
		Content:   "Hello",
		Status:    domain.MessageStatusPending,
	}
	msg.ID = createMessage(t, msg)

	msg.Status = domain.MessageStatusSent
	msg.SentAt = sql.NullTime{Time: time.Now(), Valid: true}
	msg.ResponseID = sql.NullString{String: "response-id", Valid: true}
	msg.ErrorMessage = sql.NullString{String: "error", Valid: true}
	msg.RetryCount = 1

	err := messageRepo.Update(ctx, *msg)
	assert.NoError(t, err)

	var updatedMsg domain.Message
	_, err = dbClient.Goqu.From("messages").Where(goqu.C("id").Eq(msg.ID)).ScanStructContext(ctx, &updatedMsg)

	assert.NoError(t, err)
	assert.Equal(t, msg.Status, updatedMsg.Status)
	assert.WithinDuration(t, msg.SentAt.Time, updatedMsg.SentAt.Time, time.Second)
	assert.Equal(t, msg.ResponseID, updatedMsg.ResponseID)
}

func TestMessageRepository_GetAll(t *testing.T) {
	defer cleanup(t)
	ctx := context.Background()

	createMessage(t, &domain.Message{Recipient: "1", Content: "1", Status: domain.MessageStatusPending})
	createMessage(t, &domain.Message{Recipient: "2", Content: "2", Status: domain.MessageStatusSent})

	messages, err := messageRepo.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestMessageRepository_FindDue(t *testing.T) {
	defer cleanup(t)
	ctx := context.Background()

	createMessage(t, &domain.Message{Recipient: "1", Content: "1", Status: domain.MessageStatusPending})
	createMessage(t, &domain.Message{Recipient: "2", Content: "2", Status: domain.MessageStatusSent})
	createMessage(t, &domain.Message{Recipient: "3", Content: "3", Status: domain.MessageStatusFailed})
	createMessage(t, &domain.Message{Recipient: "4", Content: "4", Status: domain.MessageStatusPending})

	messages, err := messageRepo.FindDue(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	for _, msg := range messages {
		assert.Equal(t, domain.MessageStatusPending, msg.Status)
	}
}

func TestMessageRepository_IncrementRetry(t *testing.T) {
	defer cleanup(t)
	ctx := context.Background()

	msg := &domain.Message{
		Recipient: "1234567890",
		Content:   "Hello",
		Status:    domain.MessageStatusPending,
	}
	msg.ID = createMessage(t, msg)

	err := messageRepo.IncrementRetry(ctx, msg.ID, time.Now())
	assert.NoError(t, err)

	var updatedMsg domain.Message
	_, err = dbClient.Goqu.From("messages").Where(goqu.C("id").Eq(msg.ID)).ScanStruct(&updatedMsg)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedMsg.RetryCount)
}

func TestMessageRepository_ListByStatus(t *testing.T) {
	defer cleanup(t)
	ctx := context.Background()

	createMessage(t, &domain.Message{Recipient: "1", Content: "1", Status: domain.MessageStatusPending})
	createMessage(t, &domain.Message{Recipient: "2", Content: "2", Status: domain.MessageStatusSent})
	createMessage(t, &domain.Message{Recipient: "3", Content: "3", Status: domain.MessageStatusFailed})
	createMessage(t, &domain.Message{Recipient: "4", Content: "4", Status: domain.MessageStatusPending})

	t.Run("list pending", func(t *testing.T) {
		messages, err := messageRepo.ListByStatus(ctx, string(domain.MessageStatusPending), 10, 0)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
	})

	t.Run("list sent", func(t *testing.T) {
		messages, err := messageRepo.ListByStatus(ctx, string(domain.MessageStatusSent), 10, 0)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
	})

	t.Run("list with limit and offset", func(t *testing.T) {
		messages, err := messageRepo.ListByStatus(ctx, string(domain.MessageStatusPending), 1, 1)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
	})
}

func TestMessageRepository_Create(t *testing.T) {
	defer cleanup(t)
	ctx := context.Background()

	msg := &domain.Message{
		Recipient: "1234567890",
		Content:   "Hello, Create!",
		Status:    domain.MessageStatusPending,
	}

	err := messageRepo.Create(ctx, msg)
	assert.NoError(t, err)

	var createdMsg domain.Message
	_, err = dbClient.Goqu.From("messages").
		Where(
			goqu.C("recipient").Eq(msg.Recipient),
			goqu.C("content").Eq(msg.Content),
		).ScanStructContext(ctx, &createdMsg)

	assert.NoError(t, err)
	assert.Equal(t, msg.Recipient, createdMsg.Recipient)
	assert.Equal(t, msg.Content, createdMsg.Content)
	assert.Equal(t, msg.Status, createdMsg.Status)
	assert.NotZero(t, createdMsg.ID)
	assert.False(t, createdMsg.CreatedAt.IsZero())
}
