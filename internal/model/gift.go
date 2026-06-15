package model

import "github.com/google/uuid"

type Gift struct {
	ID    uuid.UUID `json:"id"`
	Emoji string    `json:"emoji"`
	Name  string    `json:"name"`
	Price int       `json:"price"`
}
