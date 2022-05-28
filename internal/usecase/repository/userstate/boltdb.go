package userstate

import (
	"encoding/binary"
	"encoding/json"

	"enigma/internal/entity"

	bolt "go.etcd.io/bbolt"
)

var (
	stateBucketName = []byte("state")
)

type BoltDBRepository struct {
	db *bolt.DB
}

func NewBoltDB(db *bolt.DB) (*BoltDBRepository, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(stateBucketName)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &BoltDBRepository{db: db}, nil
}

func (t *BoltDBRepository) Save(userID int64, state entity.UserState) error {
	return t.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucketName)

		raw, err := json.Marshal(state)
		if err != nil {
			return err
		}

		key := itob(userID)

		err = bucket.Put(key, raw)
		if err != nil {
			return err
		}

		return nil
	})
}

func (t *BoltDBRepository) Get(userID int64) (entity.UserState, error) {
	var state entity.UserState

	err := t.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(stateBucketName).Get(itob(userID))
		if raw == nil {
			return entity.UserStateNotFoundErr
		}

		err := json.Unmarshal(raw, &state)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return entity.UserState{}, err
	}

	return state, nil
}

func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
