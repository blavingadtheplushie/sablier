package api

import "github.com/espcaa/sablier/internal/event"

// upload format shared with the server
type Batch struct {
	InstallID string        `json:"install_id"`
	Events    []event.Event `json:"events"`
	Projects  []Project     `json:"projects,omitempty"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"` // folder name, not full path
}

type Result struct {
	Accepted   int `json:"accepted"`
	Duplicates int `json:"duplicates"`
}

const MaxBatch = 500
