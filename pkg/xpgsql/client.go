package xpgsql

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var poolRegisterOnce sync.Once

func envInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func New(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	maxOpen := envInt("DB_MAX_OPEN_CONNS", 20)
	maxIdle := envInt("DB_MAX_IDLE_CONNS", 5)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := UseMetricsPlugin(db); err != nil {
		return nil, fmt.Errorf("register metrics plugin: %w", err)
	}

	poolRegisterOnce.Do(func() {
		prometheus.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "db_pool_open_connections",
				Help: "Number of open database connections.",
			},
			func() float64 {
				stats := sqlDB.Stats()
				return float64(stats.OpenConnections)
			},
		))
		prometheus.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "db_pool_in_use",
				Help: "Number of in-use database connections.",
			},
			func() float64 {
				stats := sqlDB.Stats()
				return float64(stats.InUse)
			},
		))
		prometheus.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "db_pool_idle",
				Help: "Number of idle database connections.",
			},
			func() float64 {
				stats := sqlDB.Stats()
				return float64(stats.Idle)
			},
		))
	})

	return db, nil
}
