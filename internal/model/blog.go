package model

import (
	"time"

	"github.com/google/uuid"
)

type BlogPost struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Author    string    `json:"author"`
	Excerpt   string    `json:"excerpt"`
	Body      string    `json:"body"`
	Reads     int       `json:"reads"`
	Comments  int       `json:"comments"`
	Status    string    `json:"status"` // published | review | draft
	CreatedAt time.Time `json:"created_at"`
}
