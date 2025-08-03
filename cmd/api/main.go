package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/muratdemir0/gopulse-messages/internal/adapters/db"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/ohttp"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/redis"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/webhook"
	"github.com/muratdemir0/gopulse-messages/internal/app"
	"github.com/muratdemir0/gopulse-messages/internal/config"
	"github.com/muratdemir0/gopulse-messages/internal/infra/cache"
	"github.com/muratdemir0/gopulse-messages/internal/infra/database"
	"github.com/muratdemir0/gopulse-messages/internal/infra/handlers"
	"github.com/muratdemir0/gopulse-messages/internal/infra/middleware"
	redisclient "github.com/redis/go-redis/v9"
)

type App struct {
	config         *config.Config
	db             *db.Client
	redis          *redisclient.Client
	messageService *app.MessageService
	server         *http.Server
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-quit:
		log.Printf("Shutdown signal received: %v", sig)
	}

	app.Stop()
}

func NewApp() (*App, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("Starting %s on port %d", cfg.App.Name, cfg.App.Port)

	app := &App{config: cfg}

	if err := app.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := app.initRedis(); err != nil {
		return nil, fmt.Errorf("failed to initialize redis: %w", err)
	}

	app.initServices()
	app.initServer()

	return app, nil
}

func (a *App) Start() error {
	if err := a.messageService.StartAutoSending(); err != nil {
		log.Printf("Warning: failed to start automatic message sending: %v", err)
	} else {
		log.Println("Automatic message sending started")
	}

	log.Printf("Server starting on port %d", a.config.App.Port)
	return a.server.ListenAndServe()
}

func (a *App) Stop() {
	log.Println("Starting graceful shutdown...")

	a.server.SetKeepAlivesEnabled(false)

	if err := a.messageService.StopAutoSending(); err != nil {
		log.Printf("Warning: failed to stop automatic message sending: %v", err)
	} else {
		log.Println("Automatic message sending stopped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
		if err := a.server.Close(); err != nil {
			log.Fatalf("Forced shutdown failed: %v", err)
		}
	}

	log.Println("Server gracefully stopped")
}

func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}

	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			log.Printf("failed to close redis connection: %v", err)
		}
	}
}

func (a *App) initDatabase() error {
	dbClient, err := connectDatabase(a.config)
	if err != nil {
		return err
	}
	a.db = dbClient
	log.Println("Database connection established")
	return nil
}

func (a *App) initRedis() error {
	redisClient, err := connectRedis(a.config)
	if err != nil {
		return err
	}
	a.redis = redisClient
	log.Println("Redis connection established")
	return nil
}

func (a *App) initServices() {
	httpClient := ohttp.NewClient()
	webhookClient := webhook.NewClient(a.config.Webhook.Host, httpClient)
	cache := cache.NewCache(a.redis, 24*time.Hour)
	messageRepo := database.NewMessageRepository(a.db)

	a.messageService = app.NewMessageService(
		messageRepo,
		webhookClient,
		cache,
		a.config.Webhook.Path,
	)
}

func (a *App) initServer() {
	handler := a.setupRoutes()
	a.server = a.setupHTTPServer(handler)
}

func (a *App) setupRoutes() http.Handler {
	mux := http.NewServeMux()
	handlers.RegisterHealthHandler(mux)
	handlers.RegisterMessageHandler(mux, a.messageService)

	return middleware.Recovery(mux)
}

func (a *App) setupHTTPServer(handler http.Handler) *http.Server {
	readTimeout := getTimeoutValue(a.config.App.ReadTimeout, 30)
	writeTimeout := getTimeoutValue(a.config.App.WriteTimeout, 30)
	idleTimeout := getTimeoutValue(a.config.App.IdleTimeout, 120)
	maxHeaderBytes := getHeaderSize(a.config.App.MaxHeaderMB)

	log.Printf("Server configuration: ReadTimeout=%ds, WriteTimeout=%ds, IdleTimeout=%ds, MaxHeaderBytes=%dMB",
		readTimeout, writeTimeout, idleTimeout, maxHeaderBytes>>20)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", a.config.App.Port),
		ReadTimeout:       time.Duration(readTimeout) * time.Second,
		WriteTimeout:      time.Duration(writeTimeout) * time.Second,
		IdleTimeout:       time.Duration(idleTimeout) * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    maxHeaderBytes,
		Handler:           handler,
	}
}

func loadConfig() (*config.Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	configPath := filepath.Join(".config", fmt.Sprintf("%s.yaml", env))
	return config.Load(configPath)
}

func connectDatabase(cfg *config.Config) (*db.Client, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	return db.NewDB(dsn), nil
}

func connectRedis(cfg *config.Config) (*redisclient.Client, error) {
	redisClient, err := redis.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	return redisClient, nil
}

func getTimeoutValue(configValue, defaultValue int) int {
	if configValue > 0 {
		return configValue
	}
	return defaultValue
}

func getHeaderSize(maxHeaderMB int) int {
	if maxHeaderMB > 0 {
		return maxHeaderMB << 20
	}
	return 1 << 20 // 1 MB default
}
