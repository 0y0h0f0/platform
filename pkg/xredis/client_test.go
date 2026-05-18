package xredis

import "testing"

func TestNew_Unreachable(t *testing.T) {
	_, err := New("127.0.0.1:19999", "", 0)
	if err == nil {
		t.Fatal("expected error for unreachable redis")
	}
}
