package xerr_test

import (
	"encoding/json"
	"testing"

	"task-platform/pkg/xerr"
)

func TestCodeConstants(t *testing.T) {
	codes := map[string]string{
		"OK":                  xerr.CodeOK,
		"INVALID_ARGUMENT":    xerr.CodeInvalidArgument,
		"UNAUTHENTICATED":     xerr.CodeUnauthenticated,
		"PERMISSION_DENIED":   xerr.CodePermissionDenied,
		"NOT_FOUND":           xerr.CodeNotFound,
		"ALREADY_EXISTS":      xerr.CodeAlreadyExists,
		"FAILED_PRECONDITION": xerr.CodeFailedPrecondition,
		"ABORTED":             xerr.CodeAborted,
		"RESOURCE_EXHAUSTED":  xerr.CodeResourceExhausted,
		"INTERNAL":            xerr.CodeInternal,
		"UNAVAILABLE":         xerr.CodeUnavailable,
		"DEADLINE_EXCEEDED":   xerr.CodeDeadlineExceeded,
	}

	for name, code := range codes {
		if code != name {
			t.Errorf("Code%s = %s, want %s", name, code, name)
		}
	}
}

func TestHTTPResponse(t *testing.T) {
	resp := xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: "abc-123",
		Details:   []xerr.FieldDetail{{Field: "email", Reason: "invalid"}},
		Data:      map[string]string{"key": "val"},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["code"] != "OK" {
		t.Errorf("code = %v", m["code"])
	}
	if m["request_id"] != "abc-123" {
		t.Errorf("request_id = %v", m["request_id"])
	}
}
