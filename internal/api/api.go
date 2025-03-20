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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func (s *Server) Start() error {
	// http.HandleFunc("/api/optimize", enableCORS(s.handleOptimize))
	http.HandleFunc("/api/score", enableCORS(s.handleScore))
	http.HandleFunc("/api/transformSection", enableCORS(s.handleTransformSection))

	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("Starting API server", "port", s.port)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to parse request", "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.JobDescText == "" || req.Resume == "" {
		http.Error(w, "Missing job description or resume", http.StatusBadRequest)
		return
	}

	skills, err := s.llmClient.ExtractSkills(req.JobDescText)
	if err != nil {
		slog.Error("Extract failed", "err", err)
		http.Error(w, "Extract failed", http.StatusInternalServerError)
		return
	}

	scored, err := s.llmClient.ScoreResume(skills, req.Resume)
	if err != nil {
		slog.Error("Scoring failed", "err", err)
		http.Error(w, "Scoring failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scored); err != nil {
		slog.Error("Response encoding failed", "err", err)
	}
}

func (s *Server) handleTransformSection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

}

// func (s *Server) handleOptimize(w http.ResponseWriter, r *http.Request) {
// 	var itemsToTransform []types.TransformItem

// 	// TODO: yeah I don't like this likde don't hardcode items... "exp, proj"

// 	// Collect all highlights with scores
// 	for _, exp := range scored.ProfessionalExperience {
// 		for _, highlight := range exp.Highlights {
// 			// Inside the loops:
// 			itemsToTransform = append(itemsToTransform, types.TransformItem{
// 				ID:                fmt.Sprintf("exp-%s-%d", exp.Company, len(itemsToTransform)),
// 				OriginalText:      highlight.Text,
// 				OriginalSkills:    highlight.MatchingSkills, // Note this name change
// 				Section:           "experience",
// 				Company:           exp.Company,
// 				Position:          exp.Position,
// 				OriginalScore:     highlight.Score,     // Renamed from Score
// 				CharCountOriginal: len(highlight.Text), // Added this
// 				Reasoning:         highlight.Reasoning,
// 			})
// 		}
// 	}

// 	// TODO: do this twice, once for projects and once for experiecnce
// 	for _, proj := range scored.Projects {
// 		for _, highlight := range proj.Highlights {
// 			// Inside the loops:
// 			itemsToTransform = append(itemsToTransform, types.TransformItem{
// 				ID:                fmt.Sprintf("proj-%s-%d", proj.Name, len(itemsToTransform)),
// 				OriginalText:      highlight.Text,
// 				OriginalSkills:    highlight.MatchingSkills,
// 				Section:           "projects",
// 				Name:              proj.Name,
// 				OriginalScore:     highlight.Score,
// 				CharCountOriginal: len(highlight.Text),
// 				Reasoning:         highlight.Reasoning,
// 			})
// 		}
// 	}

// 	// Sort items by score (descending)
// 	sort.Slice(itemsToTransform, func(i, j int) bool {
// 		return itemsToTransform[i].OriginalScore < itemsToTransform[j].OriginalScore
// 	})

// 	// send only lowest 7
// 	transformedItems, err := s.llmClient.TransformResumeBullets(scored, itemsToTransform[:7])
// 	if err != nil {
// 		slog.Error("Transform failed", "err", err)
// 		http.Error(w, "Transform failed", http.StatusInternalServerError)
// 		return
// 	}

// 	response := types.OptimizeResponse{
// 		ExtractedSkills:  *skills,
// 		ScoredResume:     *scored,
// 		TransformedItems: transformedItems,
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	if err := json.NewEncoder(w).Encode(response); err != nil {
// 		slog.Error("Response encoding failed", "err", err)
// 	}
// }
