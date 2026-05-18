package xhttp

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewEngineRoutes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		readyState bool
		wantStatus int
	}{
		{
			name:       "root",
			path:       "/",
			readyState: false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "healthz",
			path:       "/healthz",
			readyState: false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "readyz not ready",
			path:       "/readyz",
			readyState: false,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "readyz ready",
			path:       "/readyz",
			readyState: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "metrics",
			path:       "/metrics",
			readyState: true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ready := &atomic.Bool{}
			engine := NewEngine("api-gateway", ready)

			ready.Store(tc.readyState)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.wantStatus)
			}
		})
	}
}
