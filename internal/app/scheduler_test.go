//go:build unit

package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/muratdemir0/gopulse-messages/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestScheduler_StartStop(t *testing.T) {
	var executions int32
	task := func(ctx context.Context) error {
		atomic.AddInt32(&executions, 1)
		return nil
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scheduler := app.NewScheduler(10*time.Millisecond, task, logger)

	scheduler.Start()
	time.Sleep(25 * time.Millisecond)
	scheduler.Stop()

	executionsAfterStop := atomic.LoadInt32(&executions)
	assert.True(t, executionsAfterStop >= 2, "expected at least 2 executions, got %d", executionsAfterStop)

	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, executionsAfterStop, atomic.LoadInt32(&executions), "executions should not increase after stop")
}

func TestScheduler_TaskIsExecutedImmediately(t *testing.T) {
	executed := make(chan bool, 1)
	task := func(ctx context.Context) error {
		executed <- true
		return nil
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scheduler := app.NewScheduler(1*time.Hour, task, logger)

	scheduler.Start()
	defer scheduler.Stop()

	select {
	case <-executed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("task was not executed immediately")
	}
}

func TestScheduler_IdempotentStart(t *testing.T) {
	var executions int32
	task := func(ctx context.Context) error {
		atomic.AddInt32(&executions, 1)
		return nil
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scheduler := app.NewScheduler(10*time.Millisecond, task, logger)

	scheduler.Start()
	scheduler.Start()

	time.Sleep(25 * time.Millisecond)
	scheduler.Stop()

	executionsAfterStop := atomic.LoadInt32(&executions)
	assert.True(t, executionsAfterStop >= 2 && executionsAfterStop <= 4, "expected around 2-4 executions, got %d", executionsAfterStop)
}

func TestScheduler_IdempotentStop(t *testing.T) {
	task := func(ctx context.Context) error {
		return nil
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scheduler := app.NewScheduler(10*time.Millisecond, task, logger)

	scheduler.Start()
	time.Sleep(5 * time.Millisecond)

	scheduler.Stop()

	assert.NotPanics(t, func() {
		scheduler.Stop()
	})
}

func TestScheduler_TaskErrorLogging(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, nil))

	testErr := errors.New("task error")
	task := func(ctx context.Context) error {
		return testErr
	}

	scheduler := app.NewScheduler(10*time.Millisecond, task, logger)

	scheduler.Start()
	time.Sleep(25 * time.Millisecond)
	scheduler.Stop()

	output := logOutput.String()
	assert.Contains(t, output, "failed to execute task")
	assert.Contains(t, output, testErr.Error())
}

func TestScheduler_StopCancelsContext(t *testing.T) {
	taskStarted := make(chan bool, 1)
	task := func(ctx context.Context) error {
		taskStarted <- true
		<-ctx.Done()
		return nil
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scheduler := app.NewScheduler(100*time.Millisecond, task, logger)

	scheduler.Start()

	select {
	case <-taskStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not start")
	}

	stopCompleted := make(chan bool)
	go func() {
		scheduler.Stop()
		close(stopCompleted)
	}()

	select {
	case <-stopCompleted:
	case <-time.After(1 * time.Second):
		t.Fatal("Stop() did not complete in time, context was likely not cancelled")
	}
}

func TestScheduler_MultipleStartStopCycles(t *testing.T) {
	var executions int32
	task := func(ctx context.Context) error {
		atomic.AddInt32(&executions, 1)
		return nil
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scheduler := app.NewScheduler(10*time.Millisecond, task, logger)


	scheduler.Start()
	time.Sleep(25 * time.Millisecond)
	scheduler.Stop()

	firstCycleExecutions := atomic.LoadInt32(&executions)
	assert.True(t, firstCycleExecutions >= 2, "expected at least 2 executions in first cycle, got %d", firstCycleExecutions)

	scheduler.Start()
	time.Sleep(25 * time.Millisecond)
	scheduler.Stop()

	totalExecutions := atomic.LoadInt32(&executions)
	assert.True(t, totalExecutions > firstCycleExecutions, "second cycle should have additional executions")

	scheduler.Start()
	time.Sleep(25 * time.Millisecond)
	scheduler.Stop()

	finalExecutions := atomic.LoadInt32(&executions)
	assert.True(t, finalExecutions > totalExecutions, "third cycle should have additional executions")
}
