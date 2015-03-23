package boltsh

import (
	"github.com/boltdb/bolt"
)

type Level interface {
	/*
	   Prev returns the parent of this level.
	   Retuns nil if this level is root.
	*/
	Prev() Level
	/*
	   Cd changes the current level to the bucket stored under key.
	*/
	Cd(key string) Level
	/*
	   List returns keys for all values and buckets in this bucket.
	   Bucket keys are suffixed with a slash.
	*/
	List() []string
	/*
	   Get returns a value for a key or nil if none found.
	*/
	Get(key string) []byte
}

type RootLevel struct {
	tx *bolt.Tx
}

func NewRootLevel(tx *bolt.Tx) *RootLevel {
	return &RootLevel{tx}
}

func (rl *RootLevel) Prev() Level {
	return nil
}

func (rl *RootLevel) Cd(key string) Level {
	var rval Level
	nested := rl.tx.Bucket([]byte(key))
	if nested != nil {
		rval = &BucketLevel{nested, rl}
	}
	return rval
}

func (rl *RootLevel) List() []string {
	curr := rl.tx.Cursor()
	return list(curr)
}

func (rl *RootLevel) Get(key string) []byte {
	return nil
}

type BucketLevel struct {
	b    *bolt.Bucket
	prev Level
}

func (bl *BucketLevel) Prev() Level {
	return bl.prev
}

func (bl *BucketLevel) Cd(key string) Level {
	var rval Level
	nested := bl.b.Bucket([]byte(key))
	if nested != nil {
		rval = &BucketLevel{nested, bl}
	}
	return rval
}

func (bl *BucketLevel) List() []string {
	curr := bl.b.Cursor()
	return list(curr)
}

func (bl *BucketLevel) Get(key string) []byte {
	return bl.b.Get([]byte(key))
}

/*
list returns keys for all values and buckets in the cursor.
Bucket keys are suffixed with a slash.
*/
func list(curr *bolt.Cursor) []string {
	var rval []string
	for k, v := curr.First(); k != nil; k, v = curr.Next() {
		if v == nil {
			rval = append(rval, (string(k))+"/")
		} else {
			rval = append(rval, string(k))
		}
	}
	return rval
}
