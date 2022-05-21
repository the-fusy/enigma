package usecase

import (
	"time"

	"enigma/internal/entity"
)

type transactionRepository interface {
	Create(entity.Transaction) error
	GetByID(int) (entity.Transaction, error)
	GetByDate(time.Time) ([]entity.Transaction, error)
}

type idempotenceRepository interface {
	// MakeRecord return true if it was first time to call this method with same id
	MakeRecord(string) (bool, error)
}
