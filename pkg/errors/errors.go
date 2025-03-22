package errors

import (
	"fmt"
	"net/http"
)

type ApiError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

var (
	ErrBadRequest       = func(detail string) *ApiError { return New(http.StatusBadRequest, "Bad Request", detail) }
	ErrUnauthorized     = func(detail string) *ApiError { return New(http.StatusUnauthorized, "Unauthorized", detail) }
	ErrForbidden        = func(detail string) *ApiError { return New(http.StatusForbidden, "Forbidden", detail) }
	ErrNotFound         = func(detail string) *ApiError { return New(http.StatusNotFound, "Not Found", detail) }
	ErrMethodNotAllowed = func(detail string) *ApiError { return New(http.StatusMethodNotAllowed, "Method Not Allowed", detail) }
	ErrInternalServer   = func(detail string) *ApiError {
		return New(http.StatusInternalServerError, "Internal Server Error", detail)
	}
	ErrServiceUnavailable = func(detail string) *ApiError {
		return New(http.StatusServiceUnavailable, "Service Unavailable", detail)
	}
	ErrLLMProcessing = func(detail string) *ApiError {
		return New(http.StatusInternalServerError, "LLM Processing Failed", detail)
	}
)

func New(code int, message, detail string) *ApiError {
	return &ApiError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}

func (e *ApiError) WithRequestID(requestID string) *ApiError {
	e.RequestID = requestID
	return e
}

func (e *ApiError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Detail)
	}
	return e.Message
}

func (e *ApiError) StatusCode() int {
	return e.Code
}
