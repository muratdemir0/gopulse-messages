package app

import (
	"context"
	"log"
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
}

func NewScheduler(interval time.Duration, task func(ctx context.Context) error) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		interval: interval,
		task:     task,
		running:  false,
		stopCh:   make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

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
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Scheduler) run() {
	defer func() {
		close(s.stopCh)
	}()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	if err := s.task(s.ctx); err != nil {
		log.Printf("failed to execute task: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.task(s.ctx); err != nil {
				log.Printf("failed to execute task: %v", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}
