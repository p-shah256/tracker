package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/p-shah256/tracker/internal/llm"
	"github.com/p-shah256/tracker/pkg/types"
)

type Server struct {
	port int
}

func NewServer(port int) *Server {
	return &Server{
		port: port,
	}
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
	http.HandleFunc("/api/alternative", enableCORS(s.handleAlternative))

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

	extractedSkills, err := llm.ExtractSkills(jobDescContent)
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

	scoredResume, err := llm.ScoreResume(&extractedSkills, resumeText)
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

	var request struct {
		ScoredResume    *types.ScoredResume    `json:"scored_resume"`
		ExtractedSkills *types.ExtractedSkills `json:"extracted_skills"`
		MinScore        int                    `json:"min_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ScoredResume == nil {
		http.Error(w, "No scored resume provided", http.StatusBadRequest)
		return
	}
	if request.ExtractedSkills == nil {
		http.Error(w, "No extracted skills provided", http.StatusBadRequest)
		return
	}
	if request.MinScore <= 0 {
		request.MinScore = 7 // Default minimum score
	}
	//
	// transformedResume, err := llm.TransformHighScoring(request.ScoredResume, request.ExtractedSkills, request.MinScore)
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("Failed to transform resume: %v", err), http.StatusInternalServerError)
	// 	return
	// }
	//
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(transformedResume)
}

func (s *Server) handleAlternative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		BulletPoint    string   `json:"bullet_point"`
		MatchingSkills []string `json:"matching_skills"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if request.BulletPoint == "" {
		http.Error(w, "No bullet point provided", http.StatusBadRequest)
		return
	}
	if len(request.MatchingSkills) == 0 {
		http.Error(w, "No matching skills provided", http.StatusBadRequest)
		return
	}

	alternative, err := llm.GenAlternative(request.BulletPoint, request.MatchingSkills)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate alternative: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"alternative": alternative})
}
