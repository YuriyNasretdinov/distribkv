package db

import (
	"bytes"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var defaultBucket = []byte("default")
var replicaBucket = []byte("replication")

// Database is an open bolt database.
type Database struct {
	db       *bolt.DB
	readOnly bool
}

// NewDatabase returns an instance of a database that we can work with.
func NewDatabase(dbPath string, readOnly bool) (db *Database, closeFunc func() error, err error) {
	boltDb, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, nil, err
	}

	db = &Database{db: boltDb, readOnly: readOnly}
	closeFunc = boltDb.Close

	if err := db.createBuckets(); err != nil {
		closeFunc()
		return nil, nil, fmt.Errorf("creating default bucket: %w", err)
	}

	return db, closeFunc, nil
}

func (d *Database) createBuckets() error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(defaultBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(replicaBucket); err != nil {
			return err
		}
		return nil
	})
}

// SetKey sets the key to the requested value into the default database or returns an error.
func (d *Database) SetKey(key string, value []byte) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}

	return d.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(defaultBucket).Put([]byte(key), value); err != nil {
			return err
		}

		return tx.Bucket(replicaBucket).Put([]byte(key), value)
	})
}

// SetKeyOnReplica sets the key to the requested value into the default database and does not write
// to the replication queue.
// This method is intended to be used only on replicas.
func (d *Database) SetKeyOnReplica(key string, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(defaultBucket).Put([]byte(key), value)
	})
}

func copyByteSlice(b []byte) []byte {
	if b == nil {
		return nil
	}
	res := make([]byte, len(b))
	copy(res, b)
	return res
}

// GetNextKeyForReplication returns the key and value for the keys that have
// changed and have not yet been applied to replicas.
// If there are no new keys, nil key and value will be returned.
func (d *Database) GetNextKeyForReplication() (key, value []byte, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(replicaBucket)
		k, v := b.Cursor().First()
		key = copyByteSlice(k)
		value = copyByteSlice(v)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return key, value, nil
}

// DeleteReplicationKey deletes the key from the replication queue
// if the value matches the contents or if the key is already absent.
func (d *Database) DeleteReplicationKey(key, value []byte) (err error) {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(replicaBucket)

		v := b.Get(key)
		if v == nil {
			return errors.New("key does not exist")
		}

		if !bytes.Equal(v, value) {
			return errors.New("value does not match")
		}

		return b.Delete(key)
	})
}

// GetKey get the value of the requested from a default database.
func (d *Database) GetKey(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		result = copyByteSlice(b.Get([]byte(key)))
		return nil
	})

	if err == nil {
		return result, nil
	}
	return nil, err
}

// DeleteExtraKeys deletes the keys that do not belong to this shard.
func (d *Database) DeleteExtraKeys(isExtra func(string) bool) error {
	var keys []string

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		return b.ForEach(func(k, v []byte) error {
			ks := string(k)
			if isExtra(ks) {
				keys = append(keys, ks)
			}
			return nil
		})
	})

	if err != nil {
		return err
	}

	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)

		for _, k := range keys {
			if err := b.Delete([]byte(k)); err != nil {
				return err
			}
		}
		return nil
	})
}
