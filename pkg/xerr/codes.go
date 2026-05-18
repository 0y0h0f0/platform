package xerr

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
