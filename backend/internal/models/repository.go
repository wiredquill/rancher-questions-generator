package models

import "time"

type Repository struct {
	Name        string         `json:"name"`
	URL         string         `json:"url"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type"` // "http", "oci"
	Auth        *Authentication `json:"auth,omitempty"`
	AddedAt     time.Time      `json:"added_at"`
}

type Authentication struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	SecretName string `json:"secret_name,omitempty"`
	BaseURL    string `json:"base_url,omitempty"` // For credential reuse (e.g., dp.apps.rancher.io)
}

type Chart struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Versions    []string `json:"versions,omitempty"`
	AppVersion  string   `json:"app_version,omitempty"`
	Description string   `json:"description,omitempty"`
	Repository  string   `json:"repository"`
	Keywords    []string `json:"keywords,omitempty"`
	Icon        string   `json:"icon,omitempty"`
}

type RepositoryRequest struct {
	Name        string         `json:"name" binding:"required"`
	URL         string         `json:"url" binding:"required"`
	Description string         `json:"description,omitempty"`
	Auth        *Authentication `json:"auth,omitempty"`
}

type ChartSearchRequest struct {
	Query      string `json:"query,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type ChartProcessRequest struct {
	Repository string `json:"repository" binding:"required"`
	Chart      string `json:"chart" binding:"required"`
	Version    string `json:"version,omitempty"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Repository  string    `json:"repository"`
	Chart       string    `json:"chart"`
	Version     string    `json:"version"`
	Questions   Questions `json:"questions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StorageClass struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
	IsDefault   bool   `json:"is_default"`
}