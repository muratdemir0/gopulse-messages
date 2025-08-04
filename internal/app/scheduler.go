package app

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Scheduler struct {
	interval time.Duration
	task     func(ctx context.Context) error
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	logger   *slog.Logger
}

func NewScheduler(interval time.Duration, task func(ctx context.Context) error, logger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		interval: interval,
		task:     task,
		running:  false,
		stopCh:   make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger.With(slog.String("component", "scheduler")),
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.stopCh = make(chan struct{})
	s.running = true
	go s.run()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	s.cancel()
	<-s.stopCh

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.stopCh = make(chan struct{})
}

func (s *Scheduler) run() {
	localStopCh := s.stopCh
	defer func() {
		close(localStopCh)
	}()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	if err := s.task(s.ctx); err != nil {
		s.logger.Error("failed to execute task", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.task(s.ctx); err != nil {
				s.logger.Error("failed to execute task", "error", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}
