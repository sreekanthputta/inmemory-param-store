// Package models defines data structures for the parameter store.
package models

type ParamType string

const (
	TypeText     ParamType = "text"
	TypePassword ParamType = "password"
)

type Operation string

const (
	OpInsert Operation = "insert"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
)

// Parameter is stored in the append-only log.
type Parameter struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Type      ParamType `json:"type"`
	Operation Operation `json:"operation"`
	Timestamp int64     `json:"timestamp"` // Unix milliseconds
	IP        string    `json:"ip"`
}

// ParameterView is returned by the API (no IP exposed).
type ParameterView struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Type      ParamType `json:"type"`
	Timestamp int64     `json:"timestamp"` // Unix milliseconds
	Masked    bool      `json:"masked"`
}

type UpdateRequest struct {
	Key      string    `json:"key"`
	Value    string    `json:"value"`
	Type     ParamType `json:"type"`
	IsDelete bool      `json:"is_delete,omitempty"`
}

type BatchUpdateRequest struct {
	Updates []UpdateRequest `json:"updates"`
}

type BatchUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type ListResponse struct {
	Parameters []ParameterView `json:"parameters"`
	Count      int             `json:"count"`
}
