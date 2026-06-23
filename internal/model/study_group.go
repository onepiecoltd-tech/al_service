package model

import (
	"time"

	"github.com/google/uuid"
)

type StudyGroup struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	OwnerID     uuid.UUID `json:"owner_id"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
}
