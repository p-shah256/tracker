package types

// =============== Extraction TYPES ===============
type ExtractedSkill struct {
	Name string `json:"name"`
	// Context    string `json:"context"`
	Importance int `json:"importance"`
}

type CompanyInfo struct {
	Name     string `json:"name"`
	Position string `json:"position"`
	Level    string `json:"level"`
}

type ExtractedSkills struct {
	RequiredSkills   []ExtractedSkill `json:"required_skills"`
	NiceToHaveSkills []ExtractedSkill `json:"nice_to_have_skills"`
	CompanyInfo      CompanyInfo      `json:"company_info"`
}

// =============== scoring TYPES ===============
type ScoredResume struct {
	Sections        []Section `json:"sections"`
	OverallScore    float64   `json:"overall_score"`
	OverallComments string    `json:"overall_comments"`
	WhatToImprove   string    `json:"what_to_improve"`
	PositionLevel   string    `json:"position_level"`
}

type Section struct {
	Name            string           `json:"name"`
	Score           float64          `json:"score"`
	ScoreReasoning  string           `json:"score_reasoning"`
	MissingSkills   []ExtractedSkill `json:"missing_skills,omitempty"`
	OriginalContent string           `json:"original_content"`
}

type TransformResponse struct {
	Name           string            `json:"name"`
	Items          []TransformedItem `json:"items"`
	ImprovementExp string            `json:"improvement_explanation,omitempty"`
}

type TransformedItem struct {
	OriginalBullet    string   `json:"original_bullet"`
	TransformedBullet string   `json:"transformed_bullet,omitempty"`
	CharCountOriginal int      `json:"char_count_original"`
	CharCountNew      int      `json:"char_count_new,omitempty"`
	OriginalSkills    []string `json:"original_skills"`
	AddedSkills       []string `json:"added_skills,omitempty"`
	OriginalScore     float64  `json:"original_score"`
	NewScore          float64  `json:"new_score,omitempty"`
}

type OptimizeRequest struct {
	JobDescText string `json:"jobDescText"`
	Resume      string `json:"resume"`
}
