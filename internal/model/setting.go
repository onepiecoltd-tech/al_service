package model

import "time"

type Setting struct {
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	Value     bool      `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
