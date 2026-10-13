package dib

/*
Database implements an ephemeral (map) or persistent (boltdb) database.

Methods without comments share the same semantics as their boltdb counterparts.
See the official boltdb reference for details.

Methods with comments are custom additions to this wrapper package, or
deserve special mention as it pertains to functionality differences between
ephemeral and persistent instances.
*/
type Database interface {
	// Type returns an integer representing one (1) of the following
	// meanings:
	//
	//  - 0 (nil or bogus)
	//  - 1 (ephemeral)
	//  - 2 (persistent)
	Type() int

	// IsZero returns a Boolean value indicative of the receiver
	// instance being in a nil or uninitialized state.
	IsZero() bool

	// Batch note: only meaningful when used upon persistent
	// (boltdb) instances.
	Batch(func(Tx) error) error
	Begin(bool) (Tx, error)
	Close() error
	Open(DBOptions) error
	Path() string
	Update(func(Tx) error) error
	View(func(Tx) error) error

	// SetLoglevel assigns the instance of *BitValue to
	// the receiver. This will be propagated to all subsequent
	// instances of Bucket, Cursor and Tx.
	SetLoglevel(*BitValue)
}

/*
Tx describes a given transaction to be submitted to a qualifying instance
of [Database].

Methods without comments share the same semantics as their boltdb counterparts.
See the official boltdb reference for details.

Methods with comments are custom additions to this wrapper package.
*/
type Tx interface {
	// Type returns an integer representing one (1) of the following
	// meanings:
	//
	//  - 0 (nil or bogus)
	//  - 1 (ephemeral)
	//  - 2 (persistent)
	Type() int

	// IsZero returns a Boolean value indicative of the receiver
	// instance being in a nil or uninitialized state.
	IsZero() bool

	Bucket([]byte) Bucket
	Commit() error
	CreateBucket([]byte) (Bucket, error)
	CreateBucketIfNotExists([]byte) (Bucket, error)
	DeleteBucket([]byte) error
	ForEach(func([]byte, Bucket) error) error
	ID() int
	Rollback() error
	Writable() bool
}

/*
Bucket implements a key/value store for interface qualifier type instances.

Methods without comments share the same semantics as their boltdb counterparts.
See the official boltdb reference for details.

Methods with comments are custom additions to this wrapper package.
*/
type Bucket interface {
	// GetBig returns slices of []byte instances alongside an error
	// following an attempt to read and decode a single big endian
	// "blob" value associated with the input key value. It is an
	// error to use this method in cases where the value was not
	// written using the PutBig method.
	GetBig([]byte) ([][]byte, error)

	// PutBig returns an error following an attempt to encode the input
	// [][]byte instance into a single big endian "blob" and write it
	// to the underlying Bucket using the input key identifier. This is
	// intended for instances where a particular multi-valued key value
	// has the potential to become very large, AND where performance is
	// critical with regards to reading and writing such values.
	PutBig([]byte, [][]byte) error

	// PutMirror returns an error following an attempt to associate
	// the first and second input values within the receiver instance,
	// the second and first values in the opposite Bucket. This is used
	// when the receiver instance is expected to mirror a reverse-lookup
	// counterpart Bucket instance, usually for performance reasons. See
	// also DeleteMirror.
	PutMirror([]byte, []byte, Bucket) error

	// PutKey assigns the specified key to a zero length (non nil) []byte
	// instance automatically. This is useful for maintaining buckets that
	// either contain no values, or where the values are actually part of
	// the key.
	PutKey([]byte) error

	// DeleteMirror returns an error following an attempt to delete the
	// first input value from the receiver, and the second input value
	// from the opposite Bucket. This method is intended for use in
	// cases where the respective values were added using the PutMirror
	// method.
	DeleteMirror([]byte, []byte, Bucket) error

	// Type returns an integer representing one (1) of the following
	// meanings:
	//
	//  - 0 (nil or bogus)
	//  - 1 (ephemeral)
	//  - 2 (persistent)
	Type() int

	// IsZero returns a Boolean value indicative of the receiver
	// instance being in a nil or uninitialized state.
	IsZero() bool

	// Exists returns a Boolean value indicative of the provided key
	// input value being present in the receiver instance. This method
	// returns true if the underlying value associated with key is not
	// nil, even if said value is zero.
	Exists([]byte) bool

	// Keys returns slices of []byte instances, each representing
	// a single key residing within the receiver instance. No
	// regard is given for the state of any value associated with
	// the evaluated keys.
	Keys() [][]byte

	// Sort will instruct the receiver instance to present keys
	// sorted naturally if a value of true is provided.
	//
	// A value of false, the default, disables natural sorting.
	//
	// This method is only meaningful when used upon an ephemeral
	// Bucket instance.
	Sort(bool)

	// Buckets returns slices of []byte instances, each representing
	// a single (nested) Bucket residing within the receiver instance.
	Buckets() [][]byte

	// SubSequence subtracts one (1) from the receiver's sequence value.
	SubSequence()

	Bucket([]byte) Bucket
	CreateBucket([]byte) (Bucket, error)
	CreateBucketIfNotExists([]byte) (Bucket, error)
	Cursor() Cursor
	Delete([]byte) error
	DeleteBucket([]byte) error
	ForEach(func([]byte, []byte) error) error
	Get([]byte) []byte
	NextSequence() (uint64, error)
	Put([]byte, []byte) error
	Sequence() uint64
	SetSequence(uint64) error
	Tx() Tx
	Writable() bool
}

/*
Cursor implements a Cursor for traversing keys and values in a [Bucket].

Methods without comments share the same semantics as their boltdb counterparts.
See the official boltdb reference for details.

Methods with comments are custom additions to this wrapper package.
*/
type Cursor interface {
	// IsZero returns a Boolean value indicative of the receiver
	// instance being in a nil or uninitialized state.
	IsZero() bool

	// Type returns an integer representing one (1) of the following
	// meanings:
	//
	//  - 0 (nil or bogus)
	//  - 1 (ephemeral)
	//  - 2 (persistent)
	Type() int

	Bucket() Bucket
	First() ([]byte, []byte)
	Last() ([]byte, []byte)
	Next() ([]byte, []byte)
	Prev() ([]byte, []byte)
	Seek([]byte) ([]byte, []byte)
	Delete() error
}
