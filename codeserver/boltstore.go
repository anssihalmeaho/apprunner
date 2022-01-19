package codeserver

import (
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type boltStore struct {
	db       *bolt.DB
	filename string
}

// NewBoltStore returns new instance for boltdb
// store implementation
func NewBoltStore(packFilename string) CodeStore {
	return &boltStore{filename: packFilename}
}

// Open ...
func (bs *boltStore) Open() (err error) {
	bs.db, err = bolt.Open(bs.filename, 0666, nil)
	if err != nil {
		return err
	}
	err = bs.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("packages"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return err
}

// Put ...
func (bs *boltStore) Put(name string, content []byte) error {
	err := bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("packages"))
		err := b.Put([]byte(name), content)
		return err
	})
	return err
}

// GetAll ...
func (bs *boltStore) GetAll() []string {
	packs := []string{}
	bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("packages"))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			packs = append(packs, string(k))
		}
		return nil
	})
	return packs
}

// GetByName ...
func (bs *boltStore) GetByName(name string) (content []byte, found bool) {
	bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("packages"))
		content = b.Get([]byte(name))
		found = (content != nil)
		return nil
	})
	return
}

// DelByName ...
func (bs *boltStore) DelByName(name string) {
	bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("packages"))
		err := b.Delete([]byte(name))
		return err
	})
}

// Close ...
func (bs *boltStore) Close() {
	bs.db.Close()
}
