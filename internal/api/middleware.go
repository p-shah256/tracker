package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/p-shah256/tracker/pkg/errors"
	"github.com/p-shah256/tracker/pkg/logger"
	"slices"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := logger.WithRequestID(r.Context(), requestID)

		w.Header().Set("X-Request-ID", requestID)

		next(w, r.WithContext(ctx))
	}
}

func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		requestID := logger.GetRequestID(r.Context())

		slog.Info("Request started", "method", r.Method,
			"path", r.URL.Path,
			"request_id", requestID,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		next(rw, r)

		duration := time.Since(start)

		logAttrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"request_id", requestID,
		}

		if rw.statusCode >= 500 {
			slog.Error("Request failed with server error", logAttrs...)
		} else if rw.statusCode >= 400 {
			slog.Warn("Request failed with client error", logAttrs...)
		} else {
			slog.Info("Request completed successfully", logAttrs...)
		}
	}
}

func MethodChecker(allowedMethods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			allowed := slices.Contains(allowedMethods, r.Method)

			if !allowed {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusOK)
					return
				}

				requestID := logger.GetRequestID(r.Context())
				RespondWithError(w, errors.ErrMethodNotAllowed("Method not allowed").WithRequestID(requestID))
				return
			}

			next(w, r)
		}
	}
}

func Recover(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := logger.GetRequestID(r.Context())

				slog.Error("PANIC RECOVERED", "error", err, "request_id", requestID, "path", r.URL.Path,)

				errMsg := "Unexpected server error occurred"
				if errStr, ok := err.(string); ok {
					errMsg = errStr
				}

				RespondWithError(w, errors.ErrInternalServer(errMsg).WithRequestID(requestID))
			}
		}()

		next(w, r)
	}
}

func RespondWithJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

func RespondWithError(w http.ResponseWriter, err *errors.ApiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode())

	if encodeErr := json.NewEncoder(w).Encode(err); encodeErr != nil {
		slog.Error("Failed to encode error response", "err", encodeErr)
	}
}
