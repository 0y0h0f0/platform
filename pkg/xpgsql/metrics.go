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

	_ = db.Callback().Create().Before("gorm:create").Register("metrics:before_create", p.before("create"))
	_ = db.Callback().Query().Before("gorm:query").Register("metrics:before_query", p.before("query"))
	_ = db.Callback().Update().Before("gorm:update").Register("metrics:before_update", p.before("update"))
	_ = db.Callback().Delete().Before("gorm:delete").Register("metrics:before_delete", p.before("delete"))
	_ = db.Callback().Row().Before("gorm:row").Register("metrics:before_row", p.before("row"))
	_ = db.Callback().Raw().Before("gorm:raw").Register("metrics:before_raw", p.before("raw"))

	_ = db.Callback().Create().After("gorm:create").Register("metrics:after_create", p.after("create"))
	_ = db.Callback().Query().After("gorm:query").Register("metrics:after_query", p.after("query"))
	_ = db.Callback().Update().After("gorm:update").Register("metrics:after_update", p.after("update"))
	_ = db.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", p.after("delete"))
	_ = db.Callback().Row().After("gorm:row").Register("metrics:after_row", p.after("row"))
	_ = db.Callback().Raw().After("gorm:raw").Register("metrics:after_raw", p.after("raw"))

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
			start := startVal.(time.Time)
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
		span := spanVal.(trace.Span)

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
