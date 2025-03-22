package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/p-shah256/tracker/internal/llm"
	"github.com/p-shah256/tracker/pkg/errors"
	"github.com/p-shah256/tracker/pkg/logger"
	"github.com/p-shah256/tracker/pkg/types"
)

type Server struct {
	port      int
	llmClient llm.LLM
}

func NewServer(port int) (*Server, error) {
	llm, err := llm.New(os.Getenv("GEMINI_KEY"))
	if err != nil {
		return nil, fmt.Errorf("cannot init llm client %w", err)
	}
	return &Server{
		port:      port,
		llmClient: *llm,
	}, nil
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func applyMiddleware(handler http.HandlerFunc, methods ...string) http.HandlerFunc {
	return enableCORS(
		RequestID(
			Logger(
				Recover(
					MethodChecker(methods...)(handler),
				),
			),
		),
	)
}

func (s *Server) Start() error {
	http.HandleFunc("/score", applyMiddleware(s.handleScore, http.MethodPost))
	http.HandleFunc("/transformSection", applyMiddleware(s.handleTransformSection, http.MethodPost))
	http.HandleFunc("/upload/resume", applyMiddleware(s.handleUploadResume, http.MethodPost))
	http.HandleFunc("/health", applyMiddleware(s.handleHealthCheck, http.MethodGet))

	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("Starting API server", "port", s.port)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleScore(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GetRequestID(r.Context())

	var req types.OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to parse request",
			"err", err,
			"request_id", requestID,
		)
		RespondWithError(w, errors.ErrBadRequest("Invalid JSON format: "+err.Error()).WithRequestID(requestID))
		return
	}

	if req.JobDescText == "" {
		RespondWithError(w, errors.ErrBadRequest("Job description is required").WithRequestID(requestID))
		return
	}

	if req.Resume == "" {
		RespondWithError(w, errors.ErrBadRequest("Resume content is required").WithRequestID(requestID))
		return
	}

	skills, err := s.llmClient.ExtractSkills(req.JobDescText)
	if err != nil {
		slog.Error("Skills extraction failed",
			"err", err,
			"request_id", requestID,
		)
		RespondWithError(w, errors.ErrLLMProcessing("Failed to extract skills: "+err.Error()).WithRequestID(requestID))
		return
	}

	scored, err := s.llmClient.ScoreResume(skills, req.Resume)
	if err != nil {
		slog.Error("Resume scoring failed",
			"err", err,
			"request_id", requestID,
		)
		RespondWithError(w, errors.ErrLLMProcessing("Failed to score resume: "+err.Error()).WithRequestID(requestID))
		return
	}

	RespondWithJSON(w, http.StatusOK, scored)
}

func (s *Server) handleTransformSection(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GetRequestID(r.Context())

	var req types.Section
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to parse section",
			"err", err,
			"request_id", requestID,
		)
		RespondWithError(w, errors.ErrBadRequest("Invalid JSON format: "+err.Error()).WithRequestID(requestID))
		return
	}

	if req.Name == "" {
		RespondWithError(w, errors.ErrBadRequest("Section name is required").WithRequestID(requestID))
		return
	}

	transformedItems, err := s.llmClient.TransformResumeBullets(&req)
	if err != nil {
		slog.Error("Section transformation failed",
			"err", err,
			"section", req.Name,
			"request_id", requestID,
		)
		RespondWithError(w, errors.ErrLLMProcessing("Failed to transform section: "+err.Error()).WithRequestID(requestID))
		return
	}

	RespondWithJSON(w, http.StatusOK, transformedItems)
}

func (s *Server) handleUploadResume(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GetRequestID(r.Context())

	// Parse the multipart form with 10 MB max memory
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		slog.Error("Failed to parse multipart form", "err", err, "request_id", requestID)
		RespondWithError(w, errors.ErrBadRequest("Invalid form data: "+err.Error()).WithRequestID(requestID))
		return
	}

	file, header, err := r.FormFile("resume")
	if err != nil {
		slog.Error("Failed to get file from request", "err", err, "request_id", requestID)
		RespondWithError(w, errors.ErrBadRequest("Failed to get file: "+err.Error()).WithRequestID(requestID))
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		slog.Error("Invalid file type", "filename", header.Filename, "request_id", requestID)
		RespondWithError(w, errors.ErrBadRequest("Only PDF files are allowed").WithRequestID(requestID))
		return
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		slog.Error("Failed to read file", "err", err, "filename", header.Filename, "request_id", requestID)
		RespondWithError(w, errors.ErrInternalServer("Failed to process file: "+err.Error()).WithRequestID(requestID))
		return
	}

	slog.Info("Resume uploaded", "filename", header.Filename, "size", len(fileBytes), "request_id", requestID)

	// TODO: decide if you want to extract text here or just send it over to LLM?

	response := map[string]string{
		"message":  "Resume uploaded successfully",
		"filename": header.Filename,
		"size":     fmt.Sprintf("%d bytes", len(fileBytes)),
	}

	RespondWithJSON(w, http.StatusOK, response)
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GetRequestID(r.Context())

	// Create a more detailed health check response
	health := map[string]any{
		"status":     "up",
		"timestamp":  time.Now().Unix(),
		"version":    "1.0.0",
		"request_id": requestID,
	}

	llmStatus := "unknown"
	dbStatus := "unknown"
	// This would depend on your LLM client implementation
	// For example, you could add a Ping() method to your llm.LLM type
	// if err := s.llmClient.Ping(); err == nil {
	//    llmStatus = "available"
	// } else {
	//    llmStatus = "unavailable: " + err.Error()
	// }

	health["services"] = map[string]any{
		"llm": llmStatus,
		"db":  dbStatus,
	}

	RespondWithJSON(w, http.StatusOK, health)
}
