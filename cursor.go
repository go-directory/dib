package dib

import (
	"github.com/boltdb/bolt"
)

/*
eCursor implements a Cursor for traversing keys and values in an *[eBucket]
instance.
*/
type eCursor struct {
	store   *eDB
	ll      *BitValue
	bk      *eBucket
	keys    []string
	current int
}

/*
Type returns 1, or 0 if uninitialized.
*/
func (r *eCursor) Type() int {
	if r.IsZero() {
		return 0
	}
	return 1
}

/*
Type returns 2, or 0 if uninitialized.
*/
func (r *pCursor) Type() int {
	var typ int
	if !r.IsZero() {
		typ = 2
	}
	return typ
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *eCursor) IsZero() bool {
	return len(r.keys) == 0
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *pCursor) IsZero() bool {
	return r == nil
}

/*
First returns the first key and value byte slices.
*/
func (r *eCursor) First() ([]byte, []byte) {
	if r.IsZero() {
		return nil, nil
	}

	r.current = 0
	val, ok := r.bk.kv[r.keys[r.current]].([]byte)
	if !ok {
		// Handle the case where the value is not a []byte
		return []byte(r.keys[r.current]), nil
	}
	return []byte(r.keys[r.current]), val
}

/*
Last returns the last (final) key and value byte slices.
*/
func (r *eCursor) Last() ([]byte, []byte) {
	r.current = len(r.keys) - 1
	val, ok := r.bk.kv[r.keys[r.current]].([]byte)
	if !ok {
		// Handle the case where the value is not a []byte
		return []byte(r.keys[r.current]), nil
	}
	return []byte(r.keys[r.current]), val
}

func (r *eCursor) Delete() error {
	var err error
	if !r.IsZero() {
		delete(r.bk.kv, r.keys[r.current])
	}
	return err
}

/*
Seek returns the specified key and value byte slices.
*/
func (r *eCursor) Seek(key []byte) ([]byte, []byte) {
	k := keyString(key)
	for i, key := range r.keys {
		if key >= k {
			r.current = i
			val, ok := r.bk.kv[r.keys[r.current]].([]byte)
			if !ok {
				// Handle the case where the value is not a []byte
				return []byte(r.keys[r.current]), nil
			}
			return []byte(key), val
		}
	}
	return nil, nil
}

/*
Next returns the next (subsequent) key and value byte slices.
*/
func (r *eCursor) Next() ([]byte, []byte) {
	if r.IsZero() {
		return nil, nil
	}

	r.current++
	if r.current >= len(r.keys) {
		return nil, nil
	}
	val, ok := r.bk.kv[r.keys[r.current]].([]byte)
	if !ok {
		// Handle the case where the value is not a []byte
		return []byte(r.keys[r.current]), nil
	}
	return []byte(r.keys[r.current]), val
}

/*
Previous returns the previous (antecedent) key and value byte slices.
*/
func (r *eCursor) Prev() ([]byte, []byte) {
	r.current--
	if r.current < 0 {
		return nil, nil
	}

	val, ok := r.bk.kv[r.keys[r.current]].([]byte)
	if !ok {
		// Handle the case where the value is not a []byte
		return []byte(r.keys[r.current]), nil
	}
	return []byte(r.keys[r.current]), val
}

/*
Bucket returns the underlying instance of [Bucket] within the receiver
instance.
*/
func (r *eCursor) Bucket() Bucket {
	if !r.IsZero() {
		return r.bk
	}

	return &eBucket{ll: r.ll}
}

/*
pCursor implements a Cursor for traversing keys and values in a *[pBucket]
instance.
*/
type pCursor struct {
	c  *bolt.Cursor
	ll *BitValue
}

/*
First returns the first key and value byte slices.
*/
func (r *pCursor) First() ([]byte, []byte) {
	if !r.IsZero() {
		if r.c != nil {
			return r.c.First()
		}
	}
	return nil, nil
}

/*
First returns the last (final) key and value byte slices.
*/
func (r *pCursor) Last() ([]byte, []byte) {
	if !r.IsZero() {
		if r.c != nil {
			return r.c.Last()
		}
	}
	return nil, nil
}

func (r *pCursor) Delete() error {
	var err error
	if !r.IsZero() {
		err = r.c.Delete()
		emitErr(r.ll, err)
	}
	return err
}

/*
Seek returns the specified key and value byte slices.
*/
func (r *pCursor) Seek(key []byte) ([]byte, []byte) {
	if !r.IsZero() {
		if r.c != nil {
			return r.c.Seek(key)
		}
	}
	return nil, nil
}

/*
Next returns the next (subsequent) key and value byte slices.
*/
func (r *pCursor) Next() ([]byte, []byte) {
	if !r.IsZero() {
		if r.c != nil {
			return r.c.Next()
		}
	}
	return nil, nil
}

/*
Previous returns the previous (antecedent) key and value byte slices.
*/
func (r *pCursor) Prev() ([]byte, []byte) {
	if !r.IsZero() {
		if r.c != nil {
			return r.c.Prev()
		}
	}
	return nil, nil
}

/*
Bucket returns the underlying instance of [Bucket] within the receiver
instance.
*/
func (r *pCursor) Bucket() Bucket {
	if !r.IsZero() {
		if r.c != nil {
			return &pBucket{ll: r.ll, bk: r.c.Bucket()}
		}
	}

	return &pBucket{ll: r.ll}
}
