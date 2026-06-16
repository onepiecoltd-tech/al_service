package model

import "github.com/google/uuid"

type Badge struct {
	ID    uuid.UUID `json:"id"`
	Emoji string    `json:"emoji"`
	Name  string    `json:"name"`
	Tone  string    `json:"tone"` // son | gold | reu | ink
}
