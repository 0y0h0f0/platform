package xerr

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CodeOK                 = "OK"
	CodeInvalidArgument    = "INVALID_ARGUMENT"
	CodeUnauthenticated    = "UNAUTHENTICATED"
	CodePermissionDenied   = "PERMISSION_DENIED"
	CodeNotFound           = "NOT_FOUND"
	CodeAlreadyExists      = "ALREADY_EXISTS"
	CodeFailedPrecondition = "FAILED_PRECONDITION"
	CodeAborted            = "ABORTED"
	CodeResourceExhausted  = "RESOURCE_EXHAUSTED"
	CodeInternal           = "INTERNAL"
	CodeUnavailable        = "UNAVAILABLE"
	CodeDeadlineExceeded   = "DEADLINE_EXCEEDED"
)

type FieldDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type HTTPResponse struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	RequestID string        `json:"request_id"`
	Details   []FieldDetail `json:"details,omitempty"`
	Data      any           `json:"data,omitempty"`
}

type Error struct {
	Code    string
	Message string
}

func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) GRPCStatus() *status.Status {
	return CodeToGRPCStatus(e.Code, e.Message)
}

func CodeToGRPCStatus(code, message string) *status.Status {
	grpcCode := mapCodeToGRPC(code)
	return status.New(grpcCode, message)
}

func GRPCStatusToHTTP(grpcCode codes.Code) int {
	switch grpcCode {
	case codes.OK:
		return 200
	case codes.InvalidArgument:
		return 400
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.FailedPrecondition:
		return 400
	case codes.Aborted:
		return 409
	case codes.ResourceExhausted:
		return 429
	case codes.Internal:
		return 500
	case codes.Unavailable:
		return 503
	case codes.DeadlineExceeded:
		return 504
	default:
		return 500
	}
}

func mapCodeToGRPC(code string) codes.Code {
	switch code {
	case CodeOK:
		return codes.OK
	case CodeInvalidArgument:
		return codes.InvalidArgument
	case CodeUnauthenticated:
		return codes.Unauthenticated
	case CodePermissionDenied:
		return codes.PermissionDenied
	case CodeNotFound:
		return codes.NotFound
	case CodeAlreadyExists:
		return codes.AlreadyExists
	case CodeFailedPrecondition:
		return codes.FailedPrecondition
	case CodeAborted:
		return codes.Aborted
	case CodeResourceExhausted:
		return codes.ResourceExhausted
	case CodeInternal:
		return codes.Internal
	case CodeUnavailable:
		return codes.Unavailable
	case CodeDeadlineExceeded:
		return codes.DeadlineExceeded
	default:
		return codes.Internal
	}
}

func GRPCStatusToCode(st *status.Status) string {
	switch st.Code() {
	case codes.OK:
		return CodeOK
	case codes.InvalidArgument:
		return CodeInvalidArgument
	case codes.Unauthenticated:
		return CodeUnauthenticated
	case codes.PermissionDenied:
		return CodePermissionDenied
	case codes.NotFound:
		return CodeNotFound
	case codes.AlreadyExists:
		return CodeAlreadyExists
	case codes.FailedPrecondition:
		return CodeFailedPrecondition
	case codes.Aborted:
		return CodeAborted
	case codes.ResourceExhausted:
		return CodeResourceExhausted
	case codes.Internal:
		return CodeInternal
	case codes.Unavailable:
		return CodeUnavailable
	case codes.DeadlineExceeded:
		return CodeDeadlineExceeded
	default:
		return CodeInternal
	}
}

func safeMessage(st *status.Status) string {
	switch st.Code() {
	case codes.Internal, codes.Unknown:
		return "internal server error"
	default:
		return st.Message()
	}
}

func ToHTTPResponse(err error, requestID string) *HTTPResponse {
	if err == nil {
		return &HTTPResponse{
			Code:      CodeOK,
			Message:   "ok",
			RequestID: requestID,
		}
	}

	var e *Error
	if errors.As(err, &e) {
		msg := e.Message
		if e.Code == CodeInternal {
			msg = "internal server error"
		}
		return &HTTPResponse{
			Code:      e.Code,
			Message:   msg,
			RequestID: requestID,
		}
	}

	grpcStatus, ok := status.FromError(err)
	if ok {
		return &HTTPResponse{
			Code:      GRPCStatusToCode(grpcStatus),
			Message:   safeMessage(grpcStatus),
			RequestID: requestID,
		}
	}

	return &HTTPResponse{
		Code:      CodeInternal,
		Message:   "internal server error",
		RequestID: requestID,
	}
}
