package usecase

import (
	"errors"

	"enigma/internal/entity"
)

type GetUserstate struct {
	repo userstateRepository
}

func NewGetUserstate(repo userstateRepository) *GetUserstate {
	return &GetUserstate{
		repo: repo,
	}
}

func (u *GetUserstate) Execute(id int64) (entity.UserState, error) {
	state, err := u.repo.Get(id)
	if err != nil {
		if errors.Is(err, entity.UserStateNotFoundErr) {
			return entity.UserState{}, nil
		}
		return entity.UserState{}, err
	}
	return state, nil
}

type SaveUserstate struct {
	repo userstateRepository
}

func NewSaveUserstate(repo userstateRepository) *SaveUserstate {
	return &SaveUserstate{
		repo: repo,
	}
}

func (u *SaveUserstate) Execute(id int64, state entity.UserState) error {
	return u.repo.Save(id, state)
}
