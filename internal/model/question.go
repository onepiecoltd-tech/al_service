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
	Type         string    `json:"type"` // skill: listening | reading | writing | speaking
	CreatedAt    time.Time `json:"created_at"`
}

// AdminQuestion is a question enriched with its exam's name and language, for
// the admin question-management list (filter by language/skill/answered, search).
type AdminQuestion struct {
	ID           uuid.UUID `json:"id"`
	ExamID       uuid.UUID `json:"exam_id"`
	ExamName     string    `json:"exam_name"`
	Language     string    `json:"language"`
	Position     int       `json:"position"`
	Prompt       string    `json:"prompt"`
	SampleAnswer string    `json:"sample_answer"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"created_at"`
}

// QuestionNeedingAnswer is a question with no sample answer yet, plus its exam's
// language, for the AI answer-backfill job.
type QuestionNeedingAnswer struct {
	ID       uuid.UUID
	Prompt   string
	Language string
}
