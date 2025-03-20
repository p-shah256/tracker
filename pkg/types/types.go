package types

// =============== Extraction TYPES ===============
type ExtractedSkill struct {
	Name    string `json:"name"`
	Context string `json:"context"`
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
	ProfessionalExperience []ExperienceEntry `json:"professional_experience"`
	Projects               []ProjectEntry    `json:"projects"`
	OverallScore           float64           `json:"overall_score"`
}

type ExperienceEntry struct {
	Company        string      `json:"company"`
	Position       string      `json:"position"`
	Score          float64     `json:"score"`
	MatchingSkills []string    `json:"matching_skills"`
	Highlights     []Highlight `json:"highlights"`
}

type ProjectEntry struct {
	Name           string      `json:"name"`
	Score          float64     `json:"score"`
	MatchingSkills []string    `json:"matching_skills"`
	Highlights     []Highlight `json:"highlights"`
}

type Highlight struct {
	Text           string   `json:"text"`
	Score          float64  `json:"score"`
	MatchingSkills []string `json:"matching_skills"`
	Reasoning      string   `json:"reasoning",omitempty`
}

// ScoredHighlight represents a scored resume bullet point
type ScoredHighlight struct {
	Text           string   `json:"text"`
	Score          int      `json:"score"`
	MatchingSkills []string `json:"matching_skills"`
}

// ScoredExperienceItem represents a scored professional experience entry
type ScoredExperienceItem struct {
	Company        string            `json:"company"`
	Position       string            `json:"position"`
	Score          int               `json:"score"`
	MatchingSkills []string          `json:"matching_skills"`
	Highlights     []ScoredHighlight `json:"highlights"`
}

// ScoredProjectItem represents a scored project entry
type ScoredProjectItem struct {
	Name           string            `json:"name"`
	Score          int               `json:"score"`
	MatchingSkills []string          `json:"matching_skills"`
	Highlights     []ScoredHighlight `json:"highlights"`
}

// TransformedHighlight represents a transformed resume bullet point
type TransformedHighlight struct {
	Original         string   `json:"original"`
	Transformed      string   `json:"transformed"`
	EmphasizedSkills []string `json:"emphasized_skills"`
}

// TransformedExperienceItem represents a transformed professional experience entry
type TransformedExperienceItem struct {
	Company    string                 `json:"company"`
	Position   string                 `json:"position"`
	Highlights []TransformedHighlight `json:"highlights"`
}

// TransformedProjectItem represents a transformed project entry
type TransformedProjectItem struct {
	Name       string                 `json:"name"`
	Highlights []TransformedHighlight `json:"highlights"`
}

// TransformedResume represents the output of the transformation engine
type TransformedResume struct {
	ProfessionalExperience []TransformedExperienceItem `json:"professional_experience"`
	Projects               []TransformedProjectItem    `json:"projects"`
}

type TransformRequest struct {
	ExtractedSkills string `json:"extractedSkills"`
	Items           string `json:"items"`
	EmphasisLevel   string `json:"emphasisLevel"`
}

type TransformItem struct {
	ID                string   `json:"id"`
	OriginalText      string   `json:"original_text"`
	TransformedText   string   `json:"transformed_text,omitempty"`
	CharCountOriginal int      `json:"char_count_original"`
	CharCountNew      int      `json:"char_count_new,omitempty"`
	OriginalSkills    []string `json:"original_skills"`
	AddedSkills       []string `json:"added_skills,omitempty"`
	OriginalScore     float64  `json:"original_score"`
	NewScore          float64  `json:"new_score,omitempty"`
	Section           string   `json:"section"`
	Company           string   `json:"company,omitempty"`
	Position          string   `json:"position,omitempty"`
	Name              string   `json:"name,omitempty"`
	Reasoning         string   `json:"reasoning,omitempty"`
	ImprovementExp    string   `json:"improvement_explanation,omitempty"`
}

type TransformResponse struct {
	Items []TransformItem `json:"items"`
}

type AlternativeRequest struct {
	ExtractedSkills string `json:"extractedSkills"`
	OriginalText    string `json:"originalText"`
	MatchingSkills  string `json:"matchingSkills"`
	EmphasisLevel   string `json:"emphasisLevel"`
}

type AlternativeResponse struct {
	AlternativeText string `json:"alternative_text"`
}

// single EP types
type OptimizeRequest struct {
	JobDescText string `json:"jobDescText"`
	Resume      string `json:"resume"`
}

type OptimizeResponse struct {
	ExtractedSkills  ExtractedSkills `json:"extractedSkills"`
	ScoredResume     ScoredResume    `json:"scoredResume"`
	TransformedItems []TransformItem `json:"transformItems"`
}
