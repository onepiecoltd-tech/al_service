package model

import (
	"time"

	"github.com/google/uuid"
)

type DirectMessage struct {
	ID         uuid.UUID `json:"id"`
	SenderID   uuid.UUID `json:"sender_id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}
