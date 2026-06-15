package model

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	Reporter  string    `json:"reporter"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"` // err | warn
	Status    string    `json:"status"`   // open | resolved
	Action    string    `json:"action"`   // dismissed | hidden | removed | ""
	CreatedAt time.Time `json:"created_at"`
}
