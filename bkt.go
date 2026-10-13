package dib

import (
	"sort"

	"github.com/boltdb/bolt"
)

/*
pBucket implements a persistent [Bucket].
*/
type pBucket struct {
	bk *bolt.Bucket
	ll *BitValue
}

/*
eBucket implements an ephemeral [Bucket].
*/
type eBucket struct {
	store *eDB
	ll    *BitValue
	seq   uint64
	sort  bool
	root  *eBucket
	kv    map[string]any
	bk    map[string]*eBucket
	tx    Tx

	// ordering support
	keys []string
	pos  map[string]int
}

/*
Tx returns the tx of the Bucket.
*/
func (r *eBucket) Tx() Tx {
	var tx Tx = &eTx{ll: r.ll}
	if !r.IsZero() {
		tx = r.tx
	}

	return tx
}

/*
Tx returns the tx of the Bucket.
*/
func (r *pBucket) Tx() Tx {
	var tx *pTx = &pTx{ll: r.ll}
	if !r.IsZero() {
		if r.bk != nil {
			tx.tx = r.bk.Tx()
		}
	}

	return tx
}

/*
ForEach executes a function for each key/value pair in a Bucket.

If the provided function returns an error then the iteration is stopped
and the error is returned to the caller. The provided function must not
modify the Bucket; this will result in undefined behavior.
*/
func (r *eBucket) ForEach(fn func(k, v []byte) error) error {
	if r.IsZero() {
		emitErr(r.ll, ErrNilTx)
		return ErrNilTx
	}

	var err error
	c := r.Cursor()
	for k, v := c.First(); k != nil && err == nil; k, v = c.Next() {
		err = fn(k, v)
		emitErr(r.ll, err)
	}
	return err
}

/*
ForEach executes a function for each key/value pair in a Bucket.

If the provided function returns an error then the iteration is stopped
and the error is returned to the caller. The provided function must not
modify the Bucket; this will result in undefined behavior.
*/
func (r *pBucket) ForEach(fn func(k, v []byte) error) (err error) {
	if r.bk != nil {
		err = r.bk.ForEach(fn)
		emitErr(r.ll, err)
	}
	return
}

/*
Type returns 1, or 0 if uninitialized.
*/
func (r *eBucket) Type() int {
	var i int
	if !r.IsZero() {
		i = 1
	}
	return i
}

/*
Type returns 2, or 0 if uninitialized.
*/
func (r *pBucket) Type() int {
	var i int
	if !r.IsZero() {
		i = 2
	}
	return i
}

/*
Writable returns whether the Bucket is writable.
*/
func (r *eBucket) Writable() bool {
	var w bool
	if !r.IsZero() {
		w = r.Tx().Writable()
	}

	return w
}

/*
Writable returns whether the Bucket is writable.
*/
func (r *pBucket) Writable() bool {
	var w bool
	if !r.Tx().IsZero() {
		w = r.Tx().Writable()
	}

	return w
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *eBucket) IsZero() bool { return r == &eBucket{} }

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *pBucket) IsZero() bool { return r == &pBucket{} }

/*
Cursor returns an instance of [Cursor].
*/
func (r *eBucket) Cursor() Cursor {
	var c *eCursor = &eCursor{ll: r.ll}
	if r.IsZero() {
		return c
	}
	// Use existing keys slice; copy for cursor isolation
	keys := make([]string, len(r.keys))
	copy(keys, r.keys)
	c.store = r.store
	c.keys = keys
	c.bk = r
	return c
}

/*
Cursor returns an instance of [Cursor].
*/
func (r *pBucket) Cursor() Cursor {
	var c *pCursor = &pCursor{ll: r.ll}
	if r.bk != nil {
		c.c = r.bk.Cursor()
	}

	return c
}

/*
Sort performs no useful function. This method is only
meaningful when used upon an ephemeral bucket.
*/
func (r *pBucket) Sort(_ bool) {}

/*
Sort will instruct the receiver instance to present
keys sorted naturally if a value of true is provided.

A value of false, the default, disables natural sorting.
*/
func (r *eBucket) Sort(b bool) {
	r.sort = b
}

/*
Exists returns a Boolean value indicative of the provided
key input value being present in the receiver instance.

This method returns true if the underlying value associated
with key is not nil, even if said value is zero.
*/
func (r *pBucket) Exists(key []byte) bool {
	return r.Get(key) != nil
}

/*
Get returns the []byte value associated with key.
*/
func (r *pBucket) Get(key []byte) (val []byte) {
	if !r.IsZero() && r.bk != nil {
		val = r.bk.Get(key)
	}

	return
}

/*
Exists returns a Boolean value indicative of the provided
key input value being present in the receiver instance.

This method returns true if the underlying value associated
with key is not nil, even if said value is zero.
*/
func (r *eBucket) Exists(key []byte) bool {
	return r.Get(key) != nil
}

/*
Get returns the []byte value associated with key.
*/
func (r *eBucket) Get(key []byte) (val []byte) {
	if value, ok := r.kv[keyString(key)]; ok {
		val = value.([]byte)
	}

	return
}

/*
Put returns an error following an attempt to associate key with value
within the receiver instance.
*/
func (r *pBucket) Put(key, value []byte) error {
	err := r.bk.Put(key, value)
	emitErr(r.ll, err)
	return err
}

/*
PutBig returns an error following an attempt to encode the input values
argument as a big endian "blob" and write it to the receiver.

Usage of this method should only apply to very large value sets (where
write performance is critical). Usage of this method should be tracked.
*/
func (r *pBucket) PutBig(key []byte, values [][]byte) error {
	err := r.Put(key, encodeByteSlices(values))
	emitErr(r.ll, err)
	return err
}

func (r *eBucket) insertKey(key string) {
	if r.pos == nil {
		r.pos = make(map[string]int, len(r.kv))
	}
	if r.sort {
		// find insertion index using binary search on keys
		i := sort.Search(len(r.keys), func(i int) bool {
			return !natLess(r.keys[i], key) // keys[i] >= key
		})
		r.keys = append(r.keys, "")
		copy(r.keys[i+1:], r.keys[i:])
		r.keys[i] = key
		// update pos map
		for j := i; j < len(r.keys); j++ {
			r.pos[r.keys[j]] = j
		}
	} else {
		r.keys = append(r.keys, key)
		r.pos[key] = len(r.keys) - 1
	}
}

/*
PutMirror returns an error following an attempt in associating a -> b
in the receiver, and b -> a in the opposite [Bucket] o. Neither [Bucket]
will have its sequence advanced, as this is expected of the caller if
desired.

Use of this method is intended for cases in which a particular [Bucket]
needs a reverse-lookup counterpart [Bucket] instance, usually for read
performance reasons.

This method is NOT intended for use with regards to "blob"-based multi
valued key values.
*/
func (r *pBucket) PutMirror(a, b []byte, o Bucket) error {
	emitErr(r.ll, r.Put(a, b))
	emitErr(r.ll, o.Put(b, a))
	return nil
}

/*
PutKey returns an error following an attempt to add the specified key
to the receiver. A zero length (non nil) []byte instance is automatically
set as the value.

This is useful for maintaining buckets that are purely ordered keys with
no values (or where the values are actually part of the key).
*/
func (r *pBucket) PutKey(a []byte) error {
	emitErr(r.ll, r.Put(a, []byte{}))
	return nil
}

/*
GetBig returns slices of []byte instances alongside an error following
an attempt to decode the big endian "blob" associated with key into an
instance of [][]byte.

It is an error to use this method upon any [Bucket] with a key that
was not used in a previous call to the Bucket.PutBig method.

To extract but NOT decode the value, simply use Bucket.Get instead.
*/
func (r *pBucket) GetBig(key []byte) (decoded [][]byte, err error) {
	if get := r.Get(key); get != nil {
		decoded, err = decodeByteSlices(get)
		emitErr(r.ll, err)
	}

	return
}

/*
Put returns an error following an attempt to associate key with value
within the receiver instance. The same requirements that *bolt.Bucket
instances honor -- such as a positive writability state -- apply here.
*/
func (r *eBucket) Put(key []byte, v []byte) error {
	k := keyString(key)
	if r.kv == nil {
		r.kv = make(map[string]any)
	}
	if _, exists := r.kv[k]; !exists {
		r.insertKey(k)
	}
	r.kv[k] = append([]byte(nil), v...)
	return nil
}

/*
PutBig returns an error following an attempt to encode the input values
argument as a big endian "blob" and write it to the receiver.

Usage of this method should only apply to very large value sets (where
write performance is critical). Usage of this method should be tracked.
*/
func (r *eBucket) PutBig(key []byte, values [][]byte) error {
	err := r.Put(key, encodeByteSlices(values))
	emitErr(r.ll, err)
	return err
}

/*
GetBig returns slices of []byte instances alongside an error following
an attempt to decode the big endian "blob" associated with key into an
instance of [][]byte.

It is an error to use this method upon any [Bucket] with a key that
was not used in a previous call to the Bucket.PutBig method.

To extract but NOT decode the value, simply use Bucket.Get instead.
*/
func (r *eBucket) GetBig(key []byte) (decoded [][]byte, err error) {
	if get := r.Get(key); get != nil {
		decoded, err = decodeByteSlices(get)
		emitErr(r.ll, err)
	}

	return
}

/*
PutMirror returns an error following an attempt in associating a -> b
in the receiver, and b -> a in the opposite [Bucket] o. Neither [Bucket]
will have its sequence advanced, as this is expected of the caller.

Use of this method is intended for cases in which a particular [Bucket]
needs a reverse-lookup counterpart [Bucket] instance, usually for read
performance reasons.

This method is NOT intended for use with regards to "blob"-based multi
valued key values.
*/
func (r *eBucket) PutMirror(a, b []byte, o Bucket) error {
	emitErr(r.ll, r.Put(a, b))
	emitErr(r.ll, o.Put(b, a))
	return nil
}

/*
PutKey returns an error following an attempt to add the specified key
to the receiver. A zero length (non nil) []byte instance is automatically
set as the value.

This is useful for maintaining buckets that are purely ordered keys with
no values (or where the values are actually part of the key).
*/
func (r *eBucket) PutKey(a []byte) error {
	r.Put(a, []byte{})
	return nil
}

/*
DeleteBucket deletes a Bucket at the given key. Returns an error if the
Bucket does not exists, or if the key represents a non-Bucket value.
*/
func (r *pBucket) DeleteBucket(key []byte) error {
	var err error = ErrNilTx
	if !r.Tx().IsZero() {
		err = r.bk.DeleteBucket(key)
	}
	return err
}

/*
DeleteBucket deletes a Bucket at the given key. Returns an error if the
Bucket does not exists, or if the key represents a non-Bucket value.
*/
func (r *eBucket) DeleteBucket(key []byte) error {
	if r.tx.IsZero() {
		return ErrNilTx
	} else if !r.Writable() {
		return ErrNotWritable
	}

	strKey := keyString(key)
	if _, ok := r.bk[strKey]; !ok {
		return errBucketName(ErrBucketNotFound, strKey)
	}

	delete(r.bk, strKey)
	return nil
}

/*
Delete returns an error following an attempt to remove key and its associated
value from within the receiver instance.
*/
func (r *pBucket) Delete(key []byte) error {
	var err error = ErrNilTx
	if !r.Tx().IsZero() {
		if r.bk != nil {
			err = r.bk.Delete(key)
		}
	}
	return err
}

/*
Delete returns an error following an attempt to remove key and its associated
value from within the receiver instance.
*/
func (r *eBucket) Delete(k []byte) error {
	key := string(k)
	if _, exists := r.kv[key]; exists {
		delete(r.kv, key)
		r.deleteKey(key)
	}
	return nil
}

func (r *eBucket) deleteKey(key string) {
	if idx, ok := r.pos[key]; ok {
		// remove from keys slice
		copy(r.keys[idx:], r.keys[idx+1:])
		r.keys = r.keys[:len(r.keys)-1]
		delete(r.pos, key)
		// update pos for shifted keys
		for j := idx; j < len(r.keys); j++ {
			r.pos[r.keys[j]] = j
		}
	}
}

/*
DeleteMirror returns an error following an attempt to delete key a
from the receiver, and key b from the opposite [Bucket] o. Neither
[Bucket] will have its sequence modified, as this is expected of
the caller if desired.

Use of this method is intended for cases in which the respective values
were added using the PutMirror method upon the same [Bucket] instances.

This method is NOT intended for use with regards to "blob"-based multi
valued key values.
*/
func (r *pBucket) DeleteMirror(a, b []byte, o Bucket) error {
	emitErr(r.ll, r.Delete(a))
	emitErr(r.ll, o.Delete(b))
	return nil
}

/*
DeleteMirror returns an error following an attempt to delete key a
from the receiver, and key b from the opposite [Bucket] o. Neither
[Bucket] will have its sequence modified, as this is expected of
the caller if desired.

Use of this method is intended for cases in which the respective values
were added using the PutMirror method upon the same [Bucket] instances.

This method is NOT intended for use with regards to "blob"-based multi
valued key values.
*/
func (r *eBucket) DeleteMirror(a, b []byte, o Bucket) error {
	emitErr(r.ll, r.Delete(a))
	emitErr(r.ll, o.Delete(b))
	return nil
}

/*
Buckets returns slices of []byte instances, each representing
the key of a [Bucket] instance.

Note that this is inefficient by nature and should only be used
as a last resort, or in rare special cases.
*/
func (r *eBucket) Buckets() (b [][]byte) {
	if !r.IsZero() {
		for k := range r.bk {
			b = append(b, []byte(k))
		}
	}

	return
}

/*
Buckets returns slices of []byte instances, each representing
the key of a [Bucket] instance.

Note that this is inefficient by nature and should only be used
as a last resort, or in rare special cases.
*/
func (r *pBucket) Buckets() (b [][]byte) {
	_ = r.ForEach(func(name, value []byte) error {
		if r.Get(name) == nil {
			b = append(b, name)
		}
		return nil
	})

	return
}

/*
Keys returns an instance of [][]byte, where each slice []byte
instance contains a single key found within the receiver instance.

Note that this is inefficient by nature and should only be used
as a last resort, or in rare special cases.
*/
func (r *eBucket) Keys() (k [][]byte) {
	if !r.IsZero() {
		for x, v := range r.kv {
			if v != nil {
				k = append(k, []byte(x))
			}
		}
	}

	return
}

/*
Keys returns an instance of [][]byte, where each slice []byte
instance contains a single key found within the receiver instance.

Note that this is inefficient by nature and should only be used
as a last resort, or in rare special cases.
*/
func (r *pBucket) Keys() (k [][]byte) {
	r.ForEach(func(name, value []byte) error {
		if r.Get(name) != nil {
			k = append(k, name)
		}
		return nil
	})

	return
}

/*
CreateBucket returns an instance of [Bucket] alongside an error following
an attempt to create a new [Bucket] identified by name.
*/
func (r *pBucket) CreateBucket(name []byte) (Bucket, error) {
	b, err := r.bk.CreateBucket(name)
	emitErr(r.ll, err)
	return &pBucket{ll: r.ll, bk: b}, err
}

/*
CreateBucket returns an instance of [Bucket] alongside an error following
an attempt to create a new [Bucket] identified by name.
*/
func (r *eBucket) CreateBucket(name []byte) (Bucket, error) {
	if r.tx.IsZero() {
		emitErr(r.ll, ErrNilTx)
		return &eBucket{ll: r.ll}, ErrNilTx
	}
	// Convert byte slice to string
	strName := keyString(name)

	// Check if Bucket already exists
	if _, ok := r.bk[strName]; ok {
		err := errBucketName(ErrBucketExists, strName)
		emitErr(r.ll, err)
		return nil, err
	}

	// Create new Bucket
	r.bk[strName] = &eBucket{
		store: r.store,
		ll:    r.ll,
		root:  r,
		tx:    r.tx,
		bk:    make(map[string]*eBucket),
		kv:    make(map[string]any),
	}
	return r.bk[strName], nil
}

/*
CreateBucketIfNotExists returns an instance of [Bucket] alongside an error
following an attempt to retrieve OR create a [Bucket] identified by name.
*/
func (r *eBucket) CreateBucketIfNotExists(name []byte) (Bucket, error) {
	b := &eBucket{ll: r.ll}

	if r.tx.IsZero() {
		return b, ErrNilTx
	}

	// Convert byte slice to string
	strName := keyString(name)
	emitT(r.ll, "Creating or calling bucket %q", strName)

	var ok bool
	if b, ok = r.bk[strName]; !ok {
		// Create new Bucket
		r.bk[strName] = &eBucket{
			store: r.store,
			ll:    r.ll,
			root:  r,
			tx:    r.tx,
			bk:    make(map[string]*eBucket),
			kv:    make(map[string]any),
		}
		b = r.bk[strName]
	}

	return b, nil
}

/*
CreateBucketIfNotExists returns an instance of [Bucket] alongside an error
following an attempt to retrieve OR create a [Bucket] identified by name.
*/
func (r *pBucket) CreateBucketIfNotExists(name []byte) (Bucket, error) {
	var (
		b   *pBucket = &pBucket{ll: r.ll}
		err error
	)

	emitT(r.ll, "Creating or calling bucket %q", keyString(name))
	if !r.Tx().IsZero() {
		var _b *bolt.Bucket
		_b, err = r.bk.CreateBucketIfNotExists(name)
		emitErr(r.ll, err)
		b = &pBucket{ll: r.ll, bk: _b}
	}

	return b, err
}

/*
Bucket returns an instance of [Bucket] found within the receiver instance,
or a zero [Bucket] if not present.
*/
func (r *pBucket) Bucket(name []byte) Bucket {
	var b *pBucket = &pBucket{ll: r.ll}
	if !r.Tx().IsZero() {
		if r.bk != nil {
			b = &pBucket{ll: r.ll, bk: r.bk.Bucket(name)}
		}
	}
	return b
}

/*
Bucket returns an instance of [Bucket] found within the receiver instance,
or a zero [Bucket] if not present.
*/
func (r *eBucket) Bucket(name []byte) Bucket {
	var b *eBucket = &eBucket{ll: r.ll}
	if !r.IsZero() {
		strName := keyString(name)
		b, _ = r.bk[strName]
	}
	return b
}

/*
Sequence returns the current integer for the Bucket without incrementing it.
*/
func (r *eBucket) Sequence() uint64 {
	var seq uint64
	if !r.IsZero() {
		seq = r.seq
	}

	return seq
}

/*
Sequence returns the current integer for the Bucket without incrementing it.
*/
func (r *pBucket) Sequence() uint64 {
	var seq uint64
	if !r.Tx().IsZero() {
		if r.bk != nil {
			seq = r.bk.Sequence()
		}
	}

	return seq
}

/*
NextSequence returns an autoincrementing integer for the Bucket.
*/
func (r *pBucket) NextSequence() (uint64, error) {
	var err error = ErrNilTx
	var seq uint64
	if !r.Tx().IsZero() {
		if r.bk != nil {
			seq, err = r.bk.NextSequence()
		}
	}

	return seq, err
}

/*
SubSequence is a niladic and void method which decrements the
underlying sequence.
*/
func (r *pBucket) SubSequence() {
	if !r.Tx().IsZero() {
		if r.bk != nil {
			r.bk.SetSequence(r.bk.Sequence() - 1)
		}
	}
}

/*
NextSequence returns an autoincrementing integer for the Bucket.
*/
func (r *eBucket) NextSequence() (uint64, error) {
	r.seq++
	return r.seq, nil
}

/*
SubSequence is a niladic and void method which decrements the
underlying sequence.
*/
func (r *eBucket) SubSequence() {
	if r.seq > 0 {
		r.seq--
	}
}

/*
SetSequence updates the sequence number for the Bucket.
*/
func (r *pBucket) SetSequence(v uint64) error {
	var err error = ErrNilTx
	if !r.Tx().IsZero() {
		if r.bk != nil {
			err = r.bk.SetSequence(v)
		}
	}

	return err
}

/*
SetSequence updates the sequence number for the Bucket.
*/
func (r *eBucket) SetSequence(v uint64) error {
	var err error = ErrNilTx
	if !r.Tx().IsZero() {
		r.seq = v
		err = nil
	}

	return err
}
