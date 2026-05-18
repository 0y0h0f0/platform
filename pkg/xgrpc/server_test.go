package xgrpc

import "testing"

func TestNewServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		enableReflection bool
	}{
		{
			name:             "with reflection",
			enableReflection: true,
		},
		{
			name:             "without reflection",
			enableReflection: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := NewServer(tc.enableReflection)
			if server == nil {
				t.Fatal("NewServer() returned nil")
			}
			if server.GRPC == nil {
				t.Fatal("NewServer() returned nil grpc server")
			}
			if server.Health == nil {
				t.Fatal("NewServer() returned nil health server")
			}
		})
	}
}
