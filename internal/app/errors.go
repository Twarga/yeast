package app

type ErrorCode string

const (
	ErrorCodeUnknown         ErrorCode = "unknown"
	ErrorCodeInvalidArgument ErrorCode = "invalid_argument"
	ErrorCodeNotFound        ErrorCode = "not_found"
	ErrorCodeConflict        ErrorCode = "conflict"
	ErrorCodePrecondition    ErrorCode = "failed_precondition"
	ErrorCodeTimeout         ErrorCode = "timeout"
	ErrorCodeRuntime         ErrorCode = "runtime_error"
	ErrorCodeProvisioning    ErrorCode = "provisioning_failed"
	ErrorCodeGuest           ErrorCode = "guest_error"
	ErrorCodeInternal        ErrorCode = "internal"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Details map[string]any
	Cause   error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(ErrorCodeUnknown)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func WrapError(code ErrorCode, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func WithDetails(err error, details map[string]any) error {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		if appErr.Details == nil {
			appErr.Details = make(map[string]any)
		}
		for k, v := range details {
			appErr.Details[k] = v
		}
		return appErr
	}
	return err
}

func WithRecovery(err error, steps ...string) error {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		if appErr.Details == nil {
			appErr.Details = make(map[string]any)
		}
		appErr.Details["recovery"] = steps
		return appErr
	}
	return err
}

func NormalizeError(err error) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		if appErr.Code == "" {
			appErr.Code = ErrorCodeUnknown
		}
		return appErr
	}
	return &AppError{
		Code:    ErrorCodeUnknown,
		Message: err.Error(),
		Cause:   err,
	}
}
