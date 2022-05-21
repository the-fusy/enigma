package idempotence

import (
	bolt "go.etcd.io/bbolt"
)

var (
	idempotenceBucketName = []byte("idempotence")
)

type BoltDBRepository struct {
	db *bolt.DB
}

func NewBoltDB(db *bolt.DB) (*BoltDBRepository, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(idempotenceBucketName)
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

func (t *BoltDBRepository) MakeRecord(id string) (ok bool, err error) {
	err = t.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(idempotenceBucketName)
		v := bucket.Get([]byte(id))
		if v != nil {
			ok = false
			return nil
		}

		err := bucket.Put([]byte(id), []byte{})
		if err != nil {
			return err
		}

		ok = true
		return nil
	})
	return
}
