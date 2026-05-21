package biz

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"task-platform/internal/task/data"
)

const (
	logChannelCapacity = 1024
	logBatchSize       = 64
	logFlushInterval   = 100 * time.Millisecond
	logShutdownTimeout = 3 * time.Second
	logMaxRetries      = 3
)

var (
	logWriterChannelFull = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "task_platform_log_writer_channel_full_total",
			Help: "Total number of times the operation log channel was full, triggering synchronous fallback.",
		},
	)
	logWriterBatchFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "task_platform_log_writer_batch_failure_total",
			Help: "Total number of batch writes that failed after all retries.",
		},
	)
	logWriterMetricsOnce sync.Once
	logWriterWorkerPanic = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "task_platform_log_writer_worker_panic_total",
			Help: "Total number of times the operation log writer worker panicked and was restarted.",
		},
	)
)

type LogWriter struct {
	repo      data.OperationLogRepository
	ch        chan *data.OperationLog
	done      chan struct{}
	logger    *zap.Logger
	closed    atomic.Bool
	wg        sync.WaitGroup
}

func NewLogWriter(repo data.OperationLogRepository, logger *zap.Logger) *LogWriter {
	if repo == nil {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	logWriterMetricsOnce.Do(func() {
		prometheus.MustRegister(logWriterChannelFull)
		prometheus.MustRegister(logWriterBatchFailure)
		prometheus.MustRegister(logWriterWorkerPanic)
	})
	w := &LogWriter{
		repo:   repo,
		ch:     make(chan *data.OperationLog, logChannelCapacity),
		done:   make(chan struct{}),
		logger: logger,
	}
	w.wg.Add(1)
	go w.run()
	return w
}

func (w *LogWriter) Enqueue(ctx context.Context, log *data.OperationLog) {
	if w == nil {
		return
	}
	if w.closed.Load() {
		w.writeSync(ctx, []*data.OperationLog{log})
		return
	}

	select {
	case w.ch <- log:
	default:
		logWriterChannelFull.Inc()
		w.logger.Warn("operation log channel full, degrading to synchronous write")
		w.writeSync(ctx, []*data.OperationLog{log})
	}
}

func (w *LogWriter) Shutdown() {
	if w == nil {
		return
	}
	w.closed.Store(true)
	close(w.done)
	w.wg.Wait()

	var remaining []*data.OperationLog
	for {
		select {
		case log := <-w.ch:
			remaining = append(remaining, log)
		default:
			goto flush
		}
	}
flush:
	if len(remaining) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), logShutdownTimeout)
		defer cancel()
		w.writeSync(ctx, remaining)
	}
}

func (w *LogWriter) run() {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			logWriterWorkerPanic.Inc()
			w.logger.Error("operation log writer worker panicked, restarting",
				zap.Any("panic", r),
				zap.Stack("stack"))
			if !w.closed.Load() {
				w.wg.Add(1)
				go w.run()
			}
		}
	}()

	batch := make([]*data.OperationLog, 0, logBatchSize)
	ticker := time.NewTicker(logFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			if len(batch) > 0 {
				w.flushWithRetry(batch)
			}
			return
		case log := <-w.ch:
			batch = append(batch, log)
			if len(batch) >= logBatchSize {
				w.flushWithRetry(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.flushWithRetry(batch)
				batch = batch[:0]
			}
		}
	}
}

func (w *LogWriter) flushWithRetry(batch []*data.OperationLog) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < logMaxRetries; i++ {
		err := w.repo.CreateBatch(ctx, batch)
		if err == nil {
			return
		}
		if i < logMaxRetries-1 {
			time.Sleep(time.Duration(1<<uint(i)) * 100 * time.Millisecond)
		}
	}
	logWriterBatchFailure.Inc()
	w.logger.Error("failed to write operation logs after retries",
		zap.Int("count", len(batch)))
}

func (w *LogWriter) writeSync(ctx context.Context, logs []*data.OperationLog) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := w.repo.CreateBatch(ctx, logs); err != nil {
		w.logger.Error("synchronous operation log write failed",
			zap.Int("count", len(logs)),
			zap.Error(err))
	}
}
