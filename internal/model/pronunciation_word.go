package model

import (
	"time"

	"github.com/google/uuid"
)

type PronunciationWord struct {
	ID        uuid.UUID `json:"id"`
	Word      string    `json:"word"`
	Phonetic  string    `json:"phonetic"`
	CreatedAt time.Time `json:"created_at"`
}
