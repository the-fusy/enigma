package telegram

import (
	"strconv"
	"time"

	"enigma/internal/entity"
)

type argsParser func(state entity.UserState, args string) (entity.UserState, error)

func rawParser(state entity.UserState, args string) (entity.UserState, error) {
	state.Raw = args
	return state, nil
}

func dateParser(state entity.UserState, args string) (entity.UserState, error) {
	if args == "" {
		now := time.Now().UTC()
		state.Date = &now
		return state, nil
	}

	date, err := time.Parse("02.01.2006", args)
	if err != nil {
		return state, err
	}

	state.Date = &date

	return state, nil
}

func transactionIDParser(state entity.UserState, args string) (entity.UserState, error) {
	id, err := strconv.ParseInt(args, 10, 0)
	if err != nil {
		return state, err
	}
	uid := uint64(id)
	state.TransactionID = &uid
	return state, nil
}
