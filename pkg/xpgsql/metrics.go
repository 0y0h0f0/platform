package xpgsql

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const tracerName = "task-platform/xpgsql"

var (
	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds.",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"operation"},
	)
	dbQueryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors.",
		},
		[]string{"operation"},
	)
	metricsRegisterOnce sync.Once
)

func registerMetrics() {
	metricsRegisterOnce.Do(func() {
		prometheus.MustRegister(dbQueryDuration)
		prometheus.MustRegister(dbQueryErrors)
	})
}

type metricsPlugin struct {
	db     *gorm.DB
	tracer trace.Tracer
}

func (p *metricsPlugin) Name() string {
	return "xpgsql:metrics"
}

func (p *metricsPlugin) Initialize(db *gorm.DB) error {
	p.db = db
	p.tracer = otel.Tracer(tracerName)
	registerMetrics()

	regs := []func(string, func(*gorm.DB)) error{
		db.Callback().Create().Before("gorm:create").Register,
		db.Callback().Query().Before("gorm:query").Register,
		db.Callback().Update().Before("gorm:update").Register,
		db.Callback().Delete().Before("gorm:delete").Register,
		db.Callback().Row().Before("gorm:row").Register,
		db.Callback().Raw().Before("gorm:raw").Register,
		db.Callback().Create().After("gorm:create").Register,
		db.Callback().Query().After("gorm:query").Register,
		db.Callback().Update().After("gorm:update").Register,
		db.Callback().Delete().After("gorm:delete").Register,
		db.Callback().Row().After("gorm:row").Register,
		db.Callback().Raw().After("gorm:raw").Register,
	}

	names := []string{
		"metrics:before_create", "metrics:before_query", "metrics:before_update",
		"metrics:before_delete", "metrics:before_row", "metrics:before_raw",
		"metrics:after_create", "metrics:after_query", "metrics:after_update",
		"metrics:after_delete", "metrics:after_row", "metrics:after_raw",
	}

	callbacks := []func(*gorm.DB){
		p.before("create"), p.before("query"), p.before("update"),
		p.before("delete"), p.before("row"), p.before("raw"),
		p.after("create"), p.after("query"), p.after("update"),
		p.after("delete"), p.after("row"), p.after("raw"),
	}

	for i, register := range regs {
		if err := register(names[i], callbacks[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p *metricsPlugin) before(op string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		db.InstanceSet("metrics:start:"+op, time.Now())

		ctx := db.Statement.Context
		if ctx == nil {
			return
		}

		ctx, span := p.tracer.Start(ctx, "DB "+op,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemPostgreSQL,
				attribute.String("db.operation", op),
			),
		)
		db.InstanceSet("metrics:span:"+op, span)
		db.Statement.Context = ctx
	}
}

func (p *metricsPlugin) after(op string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startVal, exists := db.InstanceGet("metrics:start:" + op)
		if exists {
			start, ok := startVal.(time.Time)
			if !ok {
				return
			}
			duration := time.Since(start).Seconds()
			dbQueryDuration.WithLabelValues(op).Observe(duration)
		}
		if db.Error != nil {
			dbQueryErrors.WithLabelValues(op).Inc()
		}

		spanVal, exists := db.InstanceGet("metrics:span:" + op)
		if !exists {
			return
		}
		span, ok := spanVal.(trace.Span)
		if !ok {
			return
		}

		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			span.SetStatus(codes.Error, db.Error.Error())
			span.RecordError(db.Error)
		}
		span.End()
	}
}

func UseMetricsPlugin(db *gorm.DB) error {
	return db.Use(&metricsPlugin{})
}
