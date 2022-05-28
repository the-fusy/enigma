package entity

import (
	"errors"
	"time"
)

var UserStateNotFoundErr = errors.New("user state not found")

const (
	StartState = "start"

	ListTransactionsState = "listTransactions"

	CreateTransactionState = "createTransaction"

	ShowTransactionState = "showTransaction"
)

type UserState struct {
	Name string `json:"name"`

	Raw string `json:"raw"`

	ChatID    int64 `json:"chatID"`
	MessageID *int  `json:"messageID,omitempty"`

	Date *time.Time `json:"date,omitempty"`

	TransactionID *uint64 `json:"transactionID,omitempty"`
}
