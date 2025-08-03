package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	app, err := NewApp()
	if err != nil {
		slog.Error("failed to create app", "error", err)
		os.Exit(1)
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
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())
	}

	app.Stop()
}

func NewApp() (*App, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	slog.Info("Starting application", "name", cfg.App.Name, "port", cfg.App.Port)

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
		slog.Warn("failed to start automatic message sending", "error", err)
	} else {
		slog.Info("Automatic message sending started")
	}

	slog.Info("Server starting", "port", a.config.App.Port)
	return a.server.ListenAndServe()
}

func (a *App) Stop() {
	slog.Info("Starting graceful shutdown...")

	a.server.SetKeepAlivesEnabled(false)

	if err := a.messageService.StopAutoSending(); err != nil {
		slog.Warn("failed to stop automatic message sending", "error", err)
	} else {
		slog.Info("Automatic message sending stopped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
		if err := a.server.Close(); err != nil {
			slog.Error("Forced shutdown failed", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Server gracefully stopped")
}

func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			slog.Warn("failed to close database connection", "error", err)
		}
	}

	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			slog.Warn("failed to close redis connection", "error", err)
		}
	}
}

func (a *App) initDatabase() error {
	dbClient, err := connectDatabase(a.config)
	if err != nil {
		return err
	}
	a.db = dbClient
	slog.Info("Database connection established")
	return nil
}

func (a *App) initRedis() error {
	redisClient, err := connectRedis(a.config)
	if err != nil {
		return err
	}
	a.redis = redisClient
	slog.Info("Redis connection established")
	return nil
}

func (a *App) initServices() {
	clientConfig := ohttp.Config{
		RetryConfig: &ohttp.RetryConfig{
			MaxRetries:          3,
			InitialInterval:     100 * time.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2,
			MaxInterval:         10 * time.Second,
			MaxElapsedTime:      15 * time.Second,
		},
	}

	httpClient := ohttp.NewClient(clientConfig)
	webhookClient := webhook.NewClient(a.config.Webhook.Host, httpClient)
	cache := cache.NewCache(a.redis, 24*time.Hour)
	messageRepo := database.NewMessageRepository(a.db)

	a.messageService = app.NewMessageService(
		messageRepo,
		webhookClient,
		cache,
		a.config.Webhook.Path,
		slog.Default(),
	)
}

func (a *App) initServer() {
	handler := a.setupRoutes()
	a.server = a.setupHTTPServer(handler)
}

func (a *App) setupRoutes() http.Handler {
	mux := http.NewServeMux()
	handlers.RegisterHealthHandler(mux)
	handlers.RegisterMessageHandler(mux, a.messageService, slog.Default())

	return middleware.Recovery(mux)
}

func (a *App) setupHTTPServer(handler http.Handler) *http.Server {
	readTimeout := getTimeoutValue(a.config.App.ReadTimeout, 30)
	writeTimeout := getTimeoutValue(a.config.App.WriteTimeout, 30)
	idleTimeout := getTimeoutValue(a.config.App.IdleTimeout, 120)
	maxHeaderBytes := getHeaderSize(a.config.App.MaxHeaderMB)

	slog.Info("Server configuration",
		slog.Int("read_timeout_sec", readTimeout),
		slog.Int("write_timeout_sec", writeTimeout),
		slog.Int("idle_timeout_sec", idleTimeout),
		slog.Int("max_header_mb", maxHeaderBytes>>20))

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
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}

	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		cfg.Redis.Addr = addr
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		cfg.Redis.Password = password
	}
	if db := os.Getenv("REDIS_DB"); db != "" {
		if d, err := strconv.Atoi(db); err == nil {
			cfg.Redis.DB = d
		}
	}

	slog.Info("Database DSN", "dsn", cfg.Database.DSN)
	slog.Info("Redis config", "addr", cfg.Redis.Addr)

	return cfg, nil
}

func connectDatabase(cfg *config.Config) (*db.Client, error) {
	dsn := cfg.Database.DSN
	slog.Info("Connecting to database", "dsn", dsn)
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
