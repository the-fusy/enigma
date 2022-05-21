package usecase

type Idempotence struct {
	repo idempotenceRepository
}

func NewIdempotence(repo idempotenceRepository) *Idempotence {
	return &Idempotence{
		repo: repo,
	}
}

func (u *Idempotence) Execute(id string) (bool, error) {
	return u.repo.MakeRecord(id)
}
