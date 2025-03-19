package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/p-shah256/tracker/internal/llm"
	"github.com/p-shah256/tracker/pkg/types"
)

type Server struct {
	port      int
	llmClient llm.LLM
}

func NewServer(port int) (*Server, error) {
	llm, err := llm.New(os.Getenv("GEMINI_KEY"))
	if err != nil {
		return nil, fmt.Errorf("cannot init llm client", "err", err)
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/api/extract", enableCORS(s.handleExtract))
	http.HandleFunc("/api/match", enableCORS(s.handleMatch))
	http.HandleFunc("/api/transform", enableCORS(s.handleTransform))

	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("Starting API server", "port", s.port)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	jobDescContent := r.FormValue("jobDescText")
	if jobDescContent == "" {
		http.Error(w, "No job description provided", http.StatusBadRequest)
		return
	}

	extractedSkills, err := s.llmClient.ExtractSkills(jobDescContent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract skills: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(extractedSkills)
}

func (s *Server) handleMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	extractedSkillsJSON := r.FormValue("extractedSkills")
	if extractedSkillsJSON == "" {
		http.Error(w, "No extracted skills provided", http.StatusBadRequest)
		return
	}
	var extractedSkills types.ExtractedSkills
	if err := json.Unmarshal([]byte(extractedSkillsJSON), &extractedSkills); err != nil {
		http.Error(w, "Invalid extracted skills format", http.StatusBadRequest)
		return
	}

	resumeText := r.FormValue("resumeText")
	if resumeText == "" {
		http.Error(w, "No resume text provided", http.StatusBadRequest)
		return
	}

	scoredResume, err := s.llmClient.ScoreResume(&extractedSkills, resumeText)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to score resume: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scoredResume)
}

func (s *Server) handleTransform(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request types.TransformRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Failed to parse request: "+err.Error(), http.StatusBadRequest)
		return
	}
	extractedSkillsJSON := request.ExtractedSkills
	if extractedSkillsJSON == "" {
		http.Error(w, "No extracted skills provided", http.StatusBadRequest)
		return
	}

	itemsJSON := request.Items
	if itemsJSON == "" {
		http.Error(w, "No items provided", http.StatusBadRequest)
		return
	}

	emphasisLevel := request.EmphasisLevel
	if emphasisLevel == "" {
		emphasisLevel = "Moderate" // Default value
	}

	var extractedSkills types.ExtractedSkills
	if err := json.Unmarshal([]byte(extractedSkillsJSON), &extractedSkills); err != nil {
		http.Error(w, "Invalid extracted skills format", http.StatusBadRequest)
		return
	}

	var items []types.TransformItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		http.Error(w, "Invalid items format", http.StatusBadRequest)
		return
	}

	// Transform the items
	transformedItems, err := s.llmClient.TransformResumeBullets(&extractedSkills, items, emphasisLevel)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to transform resume: %v", err), http.StatusInternalServerError)
		return
	}

	response := types.TransformResponse{
		Items: transformedItems,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
