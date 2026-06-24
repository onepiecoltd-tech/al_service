package model

import (
	"time"

	"github.com/google/uuid"
)

type Exam struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Language  string     `json:"language"` // target language code: en | zh | ko | ja | ...
	Questions int        `json:"questions"`
	Author    string     `json:"author"`
	State     string     `json:"state"` // published | review | draft
	OwnerID   *uuid.UUID `json:"owner_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
