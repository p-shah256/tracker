package types

// Resume represents the full resume structure from YAML
// TODO: add all the types, get the whole resume in memory and then just replace whatever is returned by GPT

// TODO: upgrade gemini model
type Resume struct {
	CV struct {
		Sections struct {
			TechnicalSkills         []string           `yaml:"technical_skills"`
			ProfessionalExperience  []ExperienceItem   `yaml:"professional_experience"`
			Projects                []ProjectItem      `yaml:"projects"`
			OpenSourceContributions []ContributionItem `yaml:"open_source_contributions"`
		} `yaml:"sections"`
	} `yaml:"cv"`
}

// ExperienceItem represents a professional experience entry
type ExperienceItem struct {
	Company    string   `yaml:"company"`
	Position   string   `yaml:"position"`
	StartDate  string   `yaml:"start_date"`
	EndDate    string   `yaml:"end_date"`
	Location   string   `yaml:"location"`
	Highlights []string `yaml:"highlights"`
}

// ProjectItem represents a project entry
type ProjectItem struct {
	Name       string   `yaml:"name"`
	StartDate  string   `yaml:"start_date"`
	EndDate    string   `yaml:"end_date"`
	Highlights []string `yaml:"highlights"`
}

// ContributionItem represents an open source contribution
type ContributionItem struct {
	Name       string   `yaml:"name"`
	StartDate  string   `yaml:"start_date"`
	EndDate    string   `yaml:"end_date"`
	Highlights []string `yaml:"highlights"`
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

// DBFriendly represents the structured job description data
type DBFriendly struct {
	Company  string   `json:"company"`
	Position Position `json:"position"`
	Skills   []Skill  `json:"skills"`
}
