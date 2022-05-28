package transaction

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"time"

	"enigma/internal/entity"

	bolt "go.etcd.io/bbolt"
)

var NotFoundErr = errors.New("transaction not found")

var (
	transactionsBucketName = []byte("transactions")
	byIDBucketName         = []byte("byID")
	byDateBucketName       = []byte("byDate")
)

type BoltDBRepository struct {
	db *bolt.DB
}

func NewBoltDB(db *bolt.DB) (*BoltDBRepository, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		tBucket, err := tx.CreateBucketIfNotExists(transactionsBucketName)
		if err != nil {
			return err
		}

		_, err = tBucket.CreateBucketIfNotExists(byIDBucketName)
		if err != nil {
			return err
		}

		_, err = tBucket.CreateBucketIfNotExists(byDateBucketName)
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

func (t *BoltDBRepository) Create(transaction entity.Transaction) error {
	return t.db.Update(func(tx *bolt.Tx) error {
		tBucket := tx.Bucket(transactionsBucketName)
		byIDBucket := tBucket.Bucket(byIDBucketName)
		byDateBucket := tBucket.Bucket(byDateBucketName)

		id, err := byIDBucket.NextSequence()
		if err != nil {
			return err
		}

		transaction.ID = id

		raw, err := json.Marshal(transaction)
		if err != nil {
			return err
		}

		key := itob(transaction.ID)

		err = byIDBucket.Put(key, raw)
		if err != nil {
			return err
		}

		bucket, err := byDateBucket.CreateBucketIfNotExists([]byte(transaction.Date.Format("2006-01-02")))
		if err != nil {
			return err
		}

		err = bucket.Put(key, raw)
		if err != nil {
			return err
		}

		return nil
	})
}

func (t *BoltDBRepository) GetByID(id uint64) (entity.Transaction, error) {
	var transaction entity.Transaction
	err := t.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(transactionsBucketName).Bucket(byIDBucketName).Get(itob(id))
		if raw == nil {
			return NotFoundErr
		}

		err := json.Unmarshal(raw, &transaction)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return entity.Transaction{}, err
	}

	return transaction, nil
}

func (t *BoltDBRepository) GetByDate(date time.Time) ([]entity.Transaction, error) {
	var transactions []entity.Transaction
	err := t.db.View(func(tx *bolt.Tx) error {
		dateKey := []byte(date.Format("2006-01-02"))
		bucket := tx.Bucket(transactionsBucketName).Bucket(byDateBucketName).Bucket(dateKey)
		if bucket == nil {
			return nil
		}

		err := bucket.ForEach(func(k, v []byte) error {
			var transaction entity.Transaction
			err := json.Unmarshal(v, &transaction)
			if err != nil {
				return err
			}
			transactions = append(transactions, transaction)
			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
