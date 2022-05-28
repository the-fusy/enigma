package entity

import "time"

type Transaction struct {
	ID          uint64    `json:"id"`
	Date        time.Time `json:"date"`
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
}
