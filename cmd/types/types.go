package types

type Resume struct {
	CV struct {
		Sections struct {
			TechnicalSkills        []string `yaml:"technical_skills"`
			ProfessionalExperience []string `yaml:"professional_experience"`
			Projects               []string `yaml:"projects"`
		} `yaml:"sections"`
	} `yaml:"cv"`
}
