package xlog

import "testing"

func TestNew(t *testing.T) {
	t.Parallel()

	logger, err := New("api-gateway", "local")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}
