package dib

import (
	"sync"

	"github.com/boltdb/bolt"
)

/*
eTx implements an EPHEMERAL [Tx].
*/
type eTx struct {
	store     *eDB
	ll        *BitValue
	writable  bool
	root      *eBucket
	bk        map[string]*eBucket
	kv        map[string]any
	lock      *sync.RWMutex
	txid      int
	committed bool
}

func (r *eTx) DB() Database {
	if !r.IsZero() {
		return r.store
	}

	return &eDB{ll: r.ll}
}

func (r *eTx) ID() int { return r.txid }

/*
ForEach executes a function for each Bucket in the root.

If the provided function returns an error then the iteration is stopped
and the error is returned to the caller.
*/
func (r *eTx) ForEach(fn func(name []byte, b Bucket) error) error {
	if r.IsZero() {
		emitErr(r.ll, ErrNilTx)
		return ErrNilTx
	}

	return r.root.ForEach(func(k, v []byte) error {
		err := fn(k, r.root.Bucket(k))
		emitErr(r.ll, err)
		return err
	})
}

/*
Writable returns whether the Bucket is writable.
*/
func (r *eTx) Writable() bool {
	if !r.IsZero() {
		return r.writable
	}

	return false
}

/*
Type returns 1, or 0 if uninitialized.
*/
func (r *eTx) Type() int {
	var t int
	if !r.IsZero() {
		t = 1
	}
	return t
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *eTx) IsZero() bool {
	return r == nil
}

/*
Bucket returns the underlying instance of [Bucket] within the receiver
instance.
*/
func (r *eTx) Bucket(name []byte) (b Bucket) {
	if r.committed {
		panic(ErrNilTx)
	}
	strName := keyString(name)

	// Check if Bucket exists
	var ok bool
	if b, ok = r.bk[strName]; ok {
		b.(*eBucket).tx = r
	} else {
		b = &eBucket{ll: r.ll}
	}
	return
}

/*
CreateBucket returns an instance of [Bucket] alongside an error following
an attempt to create a new [Bucket] identified by name.
*/
func (r *eTx) CreateBucket(name []byte) (Bucket, error) {
	strName := keyString(name)

	// Check if Bucket already exists
	if _, ok := r.bk[strName]; ok {
		err := errBucketName(ErrBucketExists, strName)
		emitErr(r.ll, err)
		return &eBucket{ll: r.ll}, err
	}

	// Create new Bucket
	r.bk[strName] = &eBucket{
		store: r.store,
		ll:    r.ll,
		tx:    r,
		kv:    make(map[string]any),
		bk:    make(map[string]*eBucket),
	}

	// Return new Bucket
	return r.bk[strName], nil
}

/*
CreateBucketIfNotExists returns an instance of [Bucket] alongside an error
following an attempt to retrieve OR create a [Bucket] identified by name.
*/
func (r *eTx) CreateBucketIfNotExists(name []byte) (Bucket, error) {
	strName := keyString(name)

	if buck, ok := r.bk[strName]; ok {
		return buck, nil
	}

	// Create new Bucket
	r.bk[strName] = &eBucket{
		store: r.store,
		ll:    r.ll,
		tx:    r,
		kv:    make(map[string]any),
		bk:    make(map[string]*eBucket),
	}

	// Return new Bucket
	return r.bk[strName], nil
}

/*
DeleteBucket returns an error following an attempt to remove key and its
associated value from within the receiver instance.
*/
func (r *eTx) DeleteBucket(name []byte) error {
	strName := keyString(name)

	// Check if Bucket exists
	if _, ok := r.bk[strName]; !ok {
		err := errBucketName(ErrBucketNotFound, strName)
		emitErr(r.ll, err)
		return err
	}

	// Delete Bucket
	delete(r.bk, strName)

	return nil
}

/*
Commit always returns nil following an attempt to handle read/write lockers.
*/
func (r *eTx) Commit() error {
	if r.committed {
		emitErr(r.ll, ErrNilTx)
		return ErrNilTx
	}
	r.committed = true
	return nil
}

/*
Rollback always returns [ErrNotImplemented], as rollbacks do not currently
apply to the receiver type.
*/
func (r *eTx) Rollback() error {
	return ErrNotImplemented
}

/*
pTx implements a PERSISTENT [Tx] through an embedded instance
of *[bolt.Tx].
*/
type pTx struct {
	tx *bolt.Tx
	ll *BitValue
}

func (r *pTx) DB() Database {
	if !r.IsZero() {
		return &pDB{ll: r.ll, db: r.tx.DB()}
	}

	return &pDB{ll: r.ll}
}

func (r *pTx) ID() int { return r.tx.ID() }

/*
ForEach executes a function for each Bucket in the root.

If the provided function returns an error then the iteration is stopped
and the error is returned to the caller.
*/
func (r *pTx) ForEach(fn func(name []byte, b Bucket) error) error {
	if r.IsZero() {
		emitErr(r.ll, ErrNilTx)
		return ErrNilTx
	}

	return r.tx.ForEach(func(name []byte, bk *bolt.Bucket) error {
		err := fn(name, &pBucket{ll: r.ll, bk: bk})
		emitErr(r.ll, err)
		return err
	})
}

/*
Writable returns whether the Bucket is writable.
*/
func (r *pTx) Writable() bool {
	if !r.IsZero() {
		if r.tx != nil {
			return r.tx.Writable()
		}
	}

	return false
}

/*
Type returns 2, or 0 if uninitialized.
*/
func (r *pTx) Type() int {
	if r.IsZero() {
		return 0
	}
	return 2
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *pTx) IsZero() bool {
	return r == nil
}

/*
Bucket returns the underlying instance of [Bucket] within the receiver
instance.
*/
func (r *pTx) Bucket(name []byte) Bucket {
	return &pBucket{ll: r.ll, bk: r.tx.Bucket(name)}
}

/*
CreateBucket returns an instance of [Bucket] alongside an error following
an attempt to create a new [Bucket] identified by name.
*/
func (r *pTx) CreateBucket(name []byte) (Bucket, error) {
	b, err := r.tx.CreateBucket(name)
	emitErr(r.ll, err)
	return &pBucket{ll: r.ll, bk: b}, err
}

/*
CreateBucketIfNotExists returns an instance of [Bucket] alongside an error
following an attempt to retrieve OR create a [Bucket] identified by name.
*/
func (r *pTx) CreateBucketIfNotExists(name []byte) (Bucket, error) {
	b, err := r.tx.CreateBucketIfNotExists(name)
	emitErr(r.ll, err)
	return &pBucket{ll: r.ll, bk: b}, err
}

/*
DeleteBucket returns an error following an attempt to remove key and its associated
value from within the receiver instance.
*/
func (r *pTx) DeleteBucket(name []byte) error {
	err := r.tx.DeleteBucket(name)
	emitErr(r.ll, err)
	return err
}

/*
Commit returns an error following an attempt to commit the [Tx] instance held
within the receiver instance.
*/
func (r *pTx) Commit() error {
	err := r.tx.Commit()
	emitErr(r.ll, err)
	return err
}

/*
Rollback returns an error following an attempt to rollback the [Tx] instance
held by the receiver instance.
*/
func (r *pTx) Rollback() error {
	err := r.tx.Rollback()
	emitErr(r.ll, err)
	return err
}
