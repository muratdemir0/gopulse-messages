//go:build integration

package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("gopulse_messages_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("Failed to get container port: %v", err)
	}

	os.Setenv("DB_HOST", host)
	os.Setenv("DB_PORT", fmt.Sprintf("%d", port.Int()))
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "gopulse_messages_test")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	testDB = NewDB(dsn)

	_, err = testDB.Goqu.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			last_login_date TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	code := m.Run()

	if err := testDB.Close(); err != nil {
		log.Printf("Failed to close database connection: %v", err)
	}

	if err := container.Terminate(ctx); err != nil {
		log.Printf("Failed to terminate container: %v", err)
	}

	os.Exit(code)
}

type TestUser struct {
	ID            int64      `db:"id"`
	FirstName     string     `db:"first_name"`
	LastName      string     `db:"last_name"`
	Email         string     `db:"email"`
	PasswordHash  string     `db:"password_hash"`
	LastLoginDate *time.Time `db:"last_login_date"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"`
}

func setupTest(t *testing.T) (*Client, context.Context) {
	ctx := context.Background()
	_, err := testDB.Goqu.ExecContext(ctx, "TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	return testDB, ctx
}

func TestClient_QueryRow(t *testing.T) {
	client, ctx := setupTest(t)

	email := "test@example.com"
	insertDS := client.Goqu.Insert("users").Rows(
		goqu.Record{
			"first_name":    "John",
			"last_name":     "Doe",
			"email":         email,
			"password_hash": "hash123",
			"created_at":    time.Now(),
		},
	)

	result, err := client.Insert(ctx, insertDS)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	selectDS := client.Goqu.From("users").
		Where(goqu.Ex{"id": id})

	var user TestUser
	err = client.QueryRow(ctx, &user, selectDS)
	require.NoError(t, err)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
}

func TestClient_Select(t *testing.T) {
	client, ctx := setupTest(t)

	emails := []string{"test1@example.com", "test2@example.com"}
	now := time.Now()

	for _, email := range emails {
		insertDS := client.Goqu.Insert("users").Rows(
			goqu.Record{
				"first_name":    "John",
				"last_name":     "Doe",
				"email":         email,
				"password_hash": "hash123",
				"created_at":    now,
			},
		)
		_, err := client.Insert(ctx, insertDS)
		require.NoError(t, err)
	}

	selectDS := client.Goqu.From("users").
		Where(goqu.Ex{"created_at": now})

	var users []TestUser
	err := client.Select(ctx, &users, selectDS)
	require.NoError(t, err)
	assert.Len(t, users, len(emails))
}

func TestClient_Transaction(t *testing.T) {
	client, ctx := setupTest(t)

	txCtx, err := client.BeginTx(ctx)
	require.NoError(t, err)

	email := "tx@example.com"
	insertDS := client.Goqu.Insert("users").Rows(
		goqu.Record{
			"first_name":    "John",
			"last_name":     "Doe",
			"email":         email,
			"password_hash": "hash123",
			"created_at":    time.Now(),
		},
	)

	result, err := client.Insert(txCtx, insertDS)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	var user TestUser
	selectDS := client.Goqu.From("users").
		Where(goqu.Ex{"id": id})

	err = client.QueryRow(txCtx, &user, selectDS)
	require.NoError(t, err)
	assert.Equal(t, email, user.Email)

	err = client.RollbackTx(txCtx)
	require.NoError(t, err)

	err = client.QueryRow(ctx, &user, selectDS)
	assert.ErrorIs(t, err, ErrNoRows)
}

func TestClient_Update(t *testing.T) {
	client, ctx := setupTest(t)

	email := "update@example.com"
	insertDS := client.Goqu.Insert("users").Rows(
		goqu.Record{
			"first_name":    "John",
			"last_name":     "Doe",
			"email":         email,
			"password_hash": "hash123",
			"created_at":    time.Now(),
		},
	)

	result, err := client.Insert(ctx, insertDS)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	newEmail := "updated@example.com"
	updateDS := client.Goqu.Update("users").
		Set(goqu.Record{"email": newEmail}).
		Where(goqu.Ex{"id": id})

	_, err = client.Update(ctx, updateDS)
	require.NoError(t, err)

	var user TestUser
	selectDS := client.Goqu.From("users").
		Where(goqu.Ex{"id": id})

	err = client.QueryRow(ctx, &user, selectDS)
	require.NoError(t, err)
	assert.Equal(t, newEmail, user.Email)
}

func TestClient_Delete(t *testing.T) {
	client, ctx := setupTest(t)

	email := "delete@example.com"
	insertDS := client.Goqu.Insert("users").Rows(
		goqu.Record{
			"first_name":    "John",
			"last_name":     "Doe",
			"email":         email,
			"password_hash": "hash123",
			"created_at":    time.Now(),
		},
	)

	result, err := client.Insert(ctx, insertDS)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	deleteDS := client.Goqu.Delete("users").
		Where(goqu.Ex{"id": id})

	_, err = client.Delete(ctx, deleteDS)
	require.NoError(t, err)

	var user TestUser
	selectDS := client.Goqu.From("users").
		Where(goqu.Ex{"id": id})

	err = client.QueryRow(ctx, &user, selectDS)
	assert.ErrorIs(t, err, ErrNoRows)
}
