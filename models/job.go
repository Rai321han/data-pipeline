package models

import "time"

type Job struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	InputPath   string    `json:"input_path"`
	OutputPath  string    `json:"output_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type JobData struct {
	SiteURL     string   `json:"site_url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Properties  []string `json:"properties"`
}
