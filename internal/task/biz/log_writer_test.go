package biz

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"task-platform/internal/task/data"
)

type noopOpLogRepo struct {
	mu   sync.Mutex
	logs []*data.OperationLog
}

func (r *noopOpLogRepo) CreateBatch(_ context.Context, logs []*data.OperationLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, logs...)
	return nil
}

func (r *noopOpLogRepo) ListByProject(_ context.Context, _ string, _ int, _ string) ([]*data.OperationLog, string, error) {
	return nil, "", nil
}

func (r *noopOpLogRepo) ListByTask(_ context.Context, _ string, _ int, _ string) ([]*data.OperationLog, string, error) {
	return nil, "", nil
}

func (r *noopOpLogRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.logs)
}

func TestLogWriter_NilReceiver(t *testing.T) {
	var w *LogWriter
	w.Enqueue(context.Background(), &data.OperationLog{Action: "test"})
	w.Shutdown()
}

func TestLogWriter_NilRepoReturnsNil(t *testing.T) {
	w := NewLogWriter(nil, nil)
	if w != nil {
		t.Error("expected nil LogWriter when repo is nil")
	}
}

func TestLogWriter_EnqueueAndFlush(t *testing.T) {
	repo := &noopOpLogRepo{}
	w := NewLogWriter(repo, zap.NewNop())
	defer w.Shutdown()

	for i := 0; i < 10; i++ {
		w.Enqueue(context.Background(), &data.OperationLog{
			OperatorID: "user-1",
			Action:     "task.create",
		})
	}

	time.Sleep(200 * time.Millisecond)

	if repo.count() != 10 {
		t.Errorf("got %d logs, want 10", repo.count())
	}
}

func TestLogWriter_BatchFlush(t *testing.T) {
	repo := &noopOpLogRepo{}
	w := NewLogWriter(repo, zap.NewNop())
	defer w.Shutdown()

	for i := 0; i < 200; i++ {
		w.Enqueue(context.Background(), &data.OperationLog{
			OperatorID: "user-1",
			Action:     "task.create",
		})
	}

	time.Sleep(200 * time.Millisecond)

	if repo.count() != 200 {
		t.Errorf("got %d logs, want 200", repo.count())
	}
}

func TestLogWriter_Shutdown(t *testing.T) {
	repo := &noopOpLogRepo{}
	w := NewLogWriter(repo, zap.NewNop())

	for i := 0; i < 50; i++ {
		w.Enqueue(context.Background(), &data.OperationLog{
			OperatorID: "user-1",
			Action:     "task.create",
		})
	}

	w.Shutdown()

	if count := repo.count(); count != 50 {
		t.Errorf("got %d logs after shutdown, want 50", count)
	}
}

func TestLogWriter_ChannelFullDegrades(t *testing.T) {
	repo := &noopOpLogRepo{}
	w := NewLogWriter(repo, zap.NewNop())
	defer w.Shutdown()

	for i := 0; i < 2000; i++ {
		w.Enqueue(context.Background(), &data.OperationLog{
			OperatorID: "user-1",
			Action:     "task.create",
		})
	}

	time.Sleep(200 * time.Millisecond)

	if count := repo.count(); count != 2000 {
		t.Errorf("got %d logs, want 2000", count)
	}
}
