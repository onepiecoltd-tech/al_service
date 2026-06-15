package model

import (
	"time"

	"github.com/google/uuid"
)

type Exam struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Questions int       `json:"questions"`
	Author    string    `json:"author"`
	State     string    `json:"state"` // published | review | draft
	CreatedAt time.Time `json:"created_at"`
}
