package model

import (
	"time"

	"github.com/google/uuid"
)

type ChatMessage struct {
	ID        uuid.UUID `json:"id"`
	ExamID    uuid.UUID `json:"exam_id"`
	Role      string    `json:"role"` // user | model
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
