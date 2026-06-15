package model

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Icon      string    `json:"icon"`
	Text      string    `json:"text"`
	Tone      string    `json:"tone"` // son | error | gold | reu
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}
