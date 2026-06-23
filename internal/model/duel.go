package model

import (
	"time"

	"github.com/google/uuid"
)

type Duel struct {
	ID              uuid.UUID  `json:"id"`
	ChallengerID    uuid.UUID  `json:"challenger_id"`
	OpponentID      uuid.UUID  `json:"opponent_id"`
	Prompt          string     `json:"prompt"`
	ChallengerScore int        `json:"challenger_score"`
	OpponentScore   *int       `json:"opponent_score,omitempty"`
	Status          string     `json:"status"` // pending | completed | declined
	WinnerID        *uuid.UUID `json:"winner_id,omitempty"`
	ChallengerDelta int        `json:"challenger_delta"`
	OpponentDelta   int        `json:"opponent_delta"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`

	// Joined display fields (names of the two players).
	ChallengerName string `json:"challenger_name"`
	OpponentName   string `json:"opponent_name"`
}
