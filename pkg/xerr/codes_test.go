package xerr_test

import (
	"encoding/json"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"task-platform/pkg/xerr"
)

func TestCodeConstants(t *testing.T) {
	codes_ := map[string]string{
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

	for name, code := range codes_ {
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

func TestNewError(t *testing.T) {
	err := xerr.NewError(xerr.CodeNotFound, "user not found")
	if err.Code != xerr.CodeNotFound {
		t.Errorf("Code = %s, want %s", err.Code, xerr.CodeNotFound)
	}
	if err.Message != "user not found" {
		t.Errorf("Message = %s", err.Message)
	}
	if err.Error() != "user not found" {
		t.Errorf("Error() = %s", err.Error())
	}
}

func TestError_GRPCStatus(t *testing.T) {
	err := xerr.NewError(xerr.CodeNotFound, "user not found")
	st := err.GRPCStatus()
	if st.Code() != codes.NotFound {
		t.Errorf("GRPCStatus code = %v, want NotFound", st.Code())
	}
	if st.Message() != "user not found" {
		t.Errorf("GRPCStatus message = %s", st.Message())
	}
}

func TestCodeToGRPCStatus(t *testing.T) {
	tests := []struct {
		xerrCode string
		grpcCode codes.Code
	}{
		{xerr.CodeOK, codes.OK},
		{xerr.CodeInvalidArgument, codes.InvalidArgument},
		{xerr.CodeUnauthenticated, codes.Unauthenticated},
		{xerr.CodePermissionDenied, codes.PermissionDenied},
		{xerr.CodeNotFound, codes.NotFound},
		{xerr.CodeAlreadyExists, codes.AlreadyExists},
		{xerr.CodeFailedPrecondition, codes.FailedPrecondition},
		{xerr.CodeAborted, codes.Aborted},
		{xerr.CodeResourceExhausted, codes.ResourceExhausted},
		{xerr.CodeInternal, codes.Internal},
		{xerr.CodeUnavailable, codes.Unavailable},
		{xerr.CodeDeadlineExceeded, codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		st := xerr.CodeToGRPCStatus(tt.xerrCode, tt.xerrCode)
		if st.Code() != tt.grpcCode {
			t.Errorf("CodeToGRPCStatus(%s) code = %v, want %v", tt.xerrCode, st.Code(), tt.grpcCode)
		}
	}
}

func TestGRPCStatusToHTTP(t *testing.T) {
	tests := []struct {
		grpcCode codes.Code
		httpCode int
	}{
		{codes.OK, 200},
		{codes.InvalidArgument, 400},
		{codes.Unauthenticated, 401},
		{codes.PermissionDenied, 403},
		{codes.NotFound, 404},
		{codes.AlreadyExists, 409},
		{codes.FailedPrecondition, 400},
		{codes.Aborted, 409},
		{codes.ResourceExhausted, 429},
		{codes.Internal, 500},
		{codes.Unavailable, 503},
		{codes.DeadlineExceeded, 504},
	}

	for _, tt := range tests {
		got := xerr.GRPCStatusToHTTP(tt.grpcCode)
		if got != tt.httpCode {
			t.Errorf("GRPCStatusToHTTP(%v) = %d, want %d", tt.grpcCode, got, tt.httpCode)
		}
	}
}

func TestToHTTPResponse_NilError(t *testing.T) {
	resp := xerr.ToHTTPResponse(nil, "req-1")
	if resp.Code != xerr.CodeOK {
		t.Errorf("Code = %s", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("Message = %s", resp.Message)
	}
}

func TestToHTTPResponse_XerrError(t *testing.T) {
	err := xerr.NewError(xerr.CodeNotFound, "user not found")
	resp := xerr.ToHTTPResponse(err, "req-1")
	if resp.Code != xerr.CodeNotFound {
		t.Errorf("Code = %s", resp.Code)
	}
	if resp.Message != "user not found" {
		t.Errorf("Message = %s", resp.Message)
	}
	if resp.RequestID != "req-1" {
		t.Errorf("RequestID = %s", resp.RequestID)
	}
}

func TestToHTTPResponse_GRPCError(t *testing.T) {
	err := status.Error(codes.Unavailable, "service down")
	resp := xerr.ToHTTPResponse(err, "req-2")
	if resp.Code != xerr.CodeUnavailable {
		t.Errorf("Code = %s", resp.Code)
	}
	if resp.Message != "service down" {
		t.Errorf("Message = %s", resp.Message)
	}
}

func TestGRPCStatusToCode(t *testing.T) {
	tests := []struct {
		grpcCode codes.Code
		want     string
	}{
		{codes.OK, xerr.CodeOK},
		{codes.InvalidArgument, xerr.CodeInvalidArgument},
		{codes.Unauthenticated, xerr.CodeUnauthenticated},
		{codes.PermissionDenied, xerr.CodePermissionDenied},
		{codes.NotFound, xerr.CodeNotFound},
		{codes.AlreadyExists, xerr.CodeAlreadyExists},
		{codes.FailedPrecondition, xerr.CodeFailedPrecondition},
		{codes.Aborted, xerr.CodeAborted},
		{codes.ResourceExhausted, xerr.CodeResourceExhausted},
		{codes.Internal, xerr.CodeInternal},
		{codes.Unavailable, xerr.CodeUnavailable},
		{codes.DeadlineExceeded, xerr.CodeDeadlineExceeded},
	}

	for _, tt := range tests {
		st := status.New(tt.grpcCode, "")
		got := xerr.GRPCStatusToCode(st)
		if got != tt.want {
			t.Errorf("GRPCStatusToCode(%v) = %s, want %s", tt.grpcCode, got, tt.want)
		}
	}
}

func TestCodeToGRPCStatus_Unknown(t *testing.T) {
	st := xerr.CodeToGRPCStatus("UNKNOWN_CODE", "test")
	if st.Code() != codes.Internal {
		t.Errorf("unknown code should map to Internal, got %v", st.Code())
	}
}

func TestGRPCStatusToHTTP_Unknown(t *testing.T) {
	got := xerr.GRPCStatusToHTTP(codes.DataLoss)
	if got != 500 {
		t.Errorf("unknown gRPC code should map to 500, got %d", got)
	}
}

func TestGRPCStatusToCode_Unknown(t *testing.T) {
	st := status.New(codes.DataLoss, "")
	got := xerr.GRPCStatusToCode(st)
	if got != xerr.CodeInternal {
		t.Errorf("unknown gRPC code should map to CodeInternal, got %s", got)
	}
}

func TestToHTTPResponse_PlainError(t *testing.T) {
	err := errors.New("something went wrong")
	resp := xerr.ToHTTPResponse(err, "req-3")
	if resp.Code != xerr.CodeInternal {
		t.Errorf("Code = %s", resp.Code)
	}
	if resp.Message != "internal server error" {
		t.Errorf("Message = %s", resp.Message)
	}
}
