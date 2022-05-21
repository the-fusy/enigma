package usecase

import (
	"time"

	"enigma/internal/entity"
)

type CreateTransaction struct {
	repo transactionRepository
}

func NewCreateTransaction(repo transactionRepository) *CreateTransaction {
	return &CreateTransaction{
		repo: repo,
	}
}

func (c *CreateTransaction) Execute(t entity.Transaction) error {
	return c.repo.Create(t)
}

type GetTransactionByID struct {
	repo transactionRepository
}

func NewGetTransactionByID(repo transactionRepository) *GetTransactionByID {
	return &GetTransactionByID{
		repo: repo,
	}
}

func (g *GetTransactionByID) Execute(id int) (entity.Transaction, error) {
	return g.repo.GetByID(id)
}

type GetTransactionsByDate struct {
	repo transactionRepository
}

func NewGetTransactionsByDate(repo transactionRepository) *GetTransactionsByDate {
	return &GetTransactionsByDate{
		repo: repo,
	}
}

func (g *GetTransactionsByDate) Execute(date time.Time) ([]entity.Transaction, error) {
	return g.repo.GetByDate(date)
}
