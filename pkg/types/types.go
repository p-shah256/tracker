package types

// Resume represents the full resume structure from YAML
type Resume struct {
	CV struct {
		Sections struct {
			TechnicalSkills         []string           `yaml:"technical_skills" json:"technical_skills"`
			ProfessionalExperience  []ExperienceItem   `yaml:"professional_experience" json:"professional_experience"`
			Projects                []ProjectItem      `yaml:"projects" json:"projects"`
			// OpenSourceContributions []ContributionItem `yaml:"open_source_contributions" json:"open_source_contributions"`
		} `yaml:"sections" json:"sections"`
	} `yaml:"cv" json:"cv"`
}

// ExperienceItem represents a professional experience entry
type ExperienceItem struct {
	Company    string   `yaml:"company" json:"company"`
	Position   string   `yaml:"position" json:"position"`
	StartDate  string   `yaml:"start_date" json:"start_date"`
	EndDate    string   `yaml:"end_date" json:"end_date"`
	Location   string   `yaml:"location" json:"location"`
	Highlights []string `yaml:"highlights" json:"highlights"`
}

// ProjectItem represents a project entry
type ProjectItem struct {
	Name       string   `yaml:"name" json:"name"`
	StartDate  string   `yaml:"start_date" json:"start_date"`
	EndDate    string   `yaml:"end_date" json:"end_date"`
	Highlights []string `yaml:"highlights" json:"highlights"`
}

// ContributionItem represents an open source contribution
type ContributionItem struct {
	Name       string   `yaml:"name" json:"name"`
	StartDate  string   `yaml:"start_date" json:"start_date"`
	EndDate    string   `yaml:"end_date" json:"end_date"`
	Highlights []string `yaml:"highlights" json:"highlights"`
}

// Skill represents a skill requirement from a job description
type Skill struct {
	Name          string `json:"name"`          // e.g., "JavaScript", "Problem Solving"
	Type          string `json:"type"`          // "technical", "soft", or "domain"
	Priority      int    `json:"priority"`      // 1-5, where 5 is "they'll sell their soul for this skill"
	IsMustHave    bool   `json:"isMustHave"`    // true or false
	YearsRequired *int   `json:"yearsRequired"` // null if not specified
	Context       string `json:"context"`       // original requirement text
}

// Position represents the job position details
type Position struct {
	Name  string `json:"name"`
	Level int    `json:"level"` // years of experience
}

// JdJson represents the structured job description data
type JdJson struct {
	Company  string   `json:"company"`
	Position Position `json:"position"`
	Skills   []Skill  `json:"skills"`
}

// ===== New types for the pipeline approach =====

// ExtractedSkill represents a skill extracted from a job description
type ExtractedSkill struct {
	Name    string `json:"name"`
	Context string `json:"context"`
}

// CompanyInfo represents basic company and position information
type CompanyInfo struct {
	Name     string `json:"name"`
	Position string `json:"position"`
	Level    string `json:"level"`
}

// ExtractedSkills represents the output of the extraction engine
type ExtractedSkills struct {
	RequiredSkills    []ExtractedSkill `json:"required_skills"`
	NiceToHaveSkills  []ExtractedSkill `json:"nice_to_have_skills"`
	CompanyInfo       CompanyInfo      `json:"company_info"`
}

// ScoredHighlight represents a scored resume bullet point
type ScoredHighlight struct {
	Text          string   `json:"text"`
	Score         int      `json:"score"`
	MatchingSkills []string `json:"matching_skills"`
}

// ScoredExperienceItem represents a scored professional experience entry
type ScoredExperienceItem struct {
	Company        string           `json:"company"`
	Position       string           `json:"position"`
	Score          int              `json:"score"`
	MatchingSkills []string         `json:"matching_skills"`
	Highlights     []ScoredHighlight `json:"highlights"`
}

// ScoredProjectItem represents a scored project entry
type ScoredProjectItem struct {
	Name           string           `json:"name"`
	Score          int              `json:"score"`
	MatchingSkills []string         `json:"matching_skills"`
	Highlights     []ScoredHighlight `json:"highlights"`
}

// ScoredResume represents the output of the matching engine
type ScoredResume struct {
	ProfessionalExperience []ScoredExperienceItem `json:"professional_experience"`
	Projects               []ScoredProjectItem    `json:"projects"`
}

// TransformedHighlight represents a transformed resume bullet point
type TransformedHighlight struct {
	Original        string   `json:"original"`
	Transformed     string   `json:"transformed"`
	EmphasizedSkills []string `json:"emphasized_skills"`
}

// TransformedExperienceItem represents a transformed professional experience entry
type TransformedExperienceItem struct {
	Company    string                `json:"company"`
	Position   string                `json:"position"`
	Highlights []TransformedHighlight `json:"highlights"`
}

// TransformedProjectItem represents a transformed project entry
type TransformedProjectItem struct {
	Name       string                `json:"name"`
	Highlights []TransformedHighlight `json:"highlights"`
}

// TransformedResume represents the output of the transformation engine
type TransformedResume struct {
	ProfessionalExperience []TransformedExperienceItem `json:"professional_experience"`
	Projects               []TransformedProjectItem    `json:"projects"`
}
