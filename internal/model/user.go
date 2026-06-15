package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	PasswordHash string    `json:"-"`
	Handle       string    `json:"handle"`
	Plan         string    `json:"plan"`
	Coins        int       `json:"coins"`
	Elo          int       `json:"elo"`
	Streak       int       `json:"streak"`
	Wins         int       `json:"wins"`
	Presence     string    `json:"presence"`
	StatusMsg    string    `json:"status_msg"`
	Role         string    `json:"role"`   // user | mod | admin
	Status       string    `json:"status"` // active | banned
	CreatedAt    time.Time `json:"created_at"`
}
