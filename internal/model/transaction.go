package model

import (
	"time"

	"github.com/google/uuid"
)

type CoinPack struct {
	ID      uuid.UUID `json:"id"`
	VND     int       `json:"vnd"`
	Coins   int       `json:"coins"`
	Popular bool      `json:"popular"`
}

type Transaction struct {
	ID          uuid.UUID `json:"id"`
	Kind        string    `json:"kind"` // topup | gift
	Coins       int       `json:"coins"`
	VND         int       `json:"vnd"`
	Method      string    `json:"method"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // ok | failed
	CreatedAt   time.Time `json:"created_at"`
}

type AdminTransaction struct {
	Transaction
	User string `json:"user"`
}

type RevenueSummary struct {
	MonthVND    int `json:"month_vnd"`
	TodayVND    int `json:"today_vnd"`
	TopupsMonth int `json:"topups_month"`
	ProTotal    int `json:"pro_total"`
}
