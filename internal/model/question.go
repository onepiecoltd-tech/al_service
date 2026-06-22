package model

import (
	"time"

	"github.com/google/uuid"
)

type Question struct {
	ID           uuid.UUID `json:"id"`
	ExamID       uuid.UUID `json:"exam_id"`
	Position     int       `json:"position"`
	Prompt       string    `json:"prompt"`
	SampleAnswer string    `json:"sample_answer"`
	CreatedAt    time.Time `json:"created_at"`
}
