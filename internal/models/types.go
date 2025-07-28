package models

import "time"

type ChartRequest struct {
	URL string `json:"url" binding:"required"`
}

type Session struct {
	ID          string                 `json:"id"`
	ChartURL    string                 `json:"chart_url"`
	Values      map[string]interface{} `json:"values"`
	Questions   Questions              `json:"questions"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Questions struct {
	Questions []Question `yaml:"questions" json:"questions"`
}

type Question struct {
	Variable     string      `yaml:"variable" json:"variable"`
	Label        string      `yaml:"label" json:"label"`
	Description  string      `yaml:"description,omitempty" json:"description,omitempty"`
	Type         string      `yaml:"type,omitempty" json:"type,omitempty"`
	Required     bool        `yaml:"required,omitempty" json:"required,omitempty"`
	Default      interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Group        string      `yaml:"group,omitempty" json:"group,omitempty"`
	Options      []string    `yaml:"options,omitempty" json:"options,omitempty"`
	ShowIf       string      `yaml:"show_if,omitempty" json:"show_if,omitempty"`
	SubQuestions []Question  `yaml:"subquestions,omitempty" json:"subquestions,omitempty"`
}

type ChartResponse struct {
	SessionID string                 `json:"session_id"`
	Values    map[string]interface{} `json:"values"`
	Questions Questions              `json:"questions"`
}