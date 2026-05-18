package xlog

import "go.uber.org/zap"

func New(serviceName, env string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"

	return cfg.Build(
		zap.Fields(
			zap.String("service", serviceName),
			zap.String("env", env),
		),
	)
}
