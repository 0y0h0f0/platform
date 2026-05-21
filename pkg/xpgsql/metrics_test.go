package xpgsql

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestEnvInt_UsesDefault(t *testing.T) {
	v := envInt("TEST_ENV_INT_KEY_NOT_SET", 42)
	if v != 42 {
		t.Errorf("envInt = %d, want 42", v)
	}
}

func TestEnvInt_UsesEnv(t *testing.T) {
	t.Setenv("TEST_ENV_INT_KEY", "99")
	v := envInt("TEST_ENV_INT_KEY", 42)
	if v != 99 {
		t.Errorf("envInt = %d, want 99", v)
	}
}

func TestEnvInt_InvalidValue(t *testing.T) {
	t.Setenv("TEST_ENV_INT_KEY", "notanumber")
	v := envInt("TEST_ENV_INT_KEY", 42)
	if v != 42 {
		t.Errorf("envInt = %d, want 42 (default for invalid)", v)
	}
}

func TestMetricsPlugin_Name(t *testing.T) {
	p := &metricsPlugin{}
	if p.Name() != "xpgsql:metrics" {
		t.Errorf("Name() = %s, want xpgsql:metrics", p.Name())
	}
}

func TestRegisterMetrics_Idempotent(t *testing.T) {
	// First call registers the metrics.
	registerMetrics()
	// Second call must not panic (sync.Once prevents double-registration).
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("registerMetrics panicked on second call: %v", r)
			}
		}()
		registerMetrics()
	}()
	// Verify metrics are registered by unregistering them.
	if !prometheus.Unregister(dbQueryDuration) {
		t.Error("dbQueryDuration was not registered")
	}
	if !prometheus.Unregister(dbQueryErrors) {
		t.Error("dbQueryErrors was not registered")
	}
}
