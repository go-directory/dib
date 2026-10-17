package dib

/*
db.go handles the high-level Database interaction for eDB and pDB
instances.
*/

import (
	"os"
	"sync"
	"time"

	"github.com/go-directory/dib/internal/bolt"
)

func NewDatabase(typ int) (base Database, err error) {
	bv := new8BitValue()
	switch typ {
	case 1:
		base = &eDB{
			bk:   make(map[string]*eBucket, 0),
			lock: &sync.RWMutex{},
			ll:   &bv,
		}
	case 2:
		base = &pDB{ll: &bv}
	default:
		err = ErrNilBackend
	}
	return
}

/*
DBOptions implements [bolt.Options] but with added fields.
*/
type DBOptions struct {
	// Timeout is the amount of time to wait to obtain a file lock.
	// When set to zero it will wait indefinitely. This option is only
	// available on Darwin and Linux.
	Timeout time.Duration

	// Sets the DB.NoGrowSync flag before memory mapping the file.
	NoGrowSync bool

	// Open database in read-only mode. Uses flock(..., LOCK_SH |LOCK_NB) to
	// grab a shared lock (UNIX).
	ReadOnly bool

	// Sets the DB.MmapFlags flag before memory mapping the file.
	MmapFlags int

	// InitialMmapSize is the initial mmap size of the database
	// in bytes. Read transactions won't block write transaction
	// if the InitialMmapSize is large enough to hold database mmap
	// size. (See DB.Begin for more information)
	//
	// If <=0, the initial map size is 0.
	// If initialMmapSize is smaller than the previous database size,
	// it takes no effect.
	InitialMmapSize int

	// Use with caution; see boltdb/bolt reference
	NoSync bool

	// Actual database file
	File string

	// Octal mode of file
	Mode os.FileMode
}

/*
pDB defines a traditional *[bolt.DB] instance and qualifies the
[Database] interface signature.
*/
type pDB struct {
	db *bolt.DB
	ll *BitValue
}

/*
Type returns 2, or 0 if uninitialized. A value of 2 indicates the
receiver is Persistent in nature.
*/
func (r *pDB) Type() int {
	var i int
	if !r.IsZero() {
		i = 2
	}
	return i
}

/*
Batch returns an error following a batch operation triggered
by a call to fn.
*/
func (r *pDB) Batch(fn func(Tx) error) error {
	return r.db.Batch(func(btx *bolt.Tx) error {
		return fn(&pTx{ll: r.ll, tx: btx})
	})
}

/*
Close returns an error following an attempt to close the underlying
*/
func (r *pDB) Close() error {
	return r.db.Close()
}

/*
Path returns the path to the currently open Database.
*/
func (r *pDB) Path() string {
	return r.db.Path()
}

/*
SetLoglevel assigns the input *[Loglevel] instance to the receiver.
*/
func (r *pDB) SetLoglevel(ll *BitValue) { r.ll = ll }

/*
View returns an error following an attempt to begin a new view [Tx].
*/
func (r *pDB) View(fn func(tx Tx) error) error {
	return r.db.View(func(btx *bolt.Tx) error {
		return fn(&pTx{ll: r.ll, tx: btx})
	})
}

/*
Begin returns an instance of [Tx] alongside an error following an attempt
to begin a new transaction.
*/
func (r *pDB) Begin(writeable bool) (Tx, error) {
	t, err := r.db.Begin(writeable)
	return &pTx{ll: r.ll, tx: t}, err
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *pDB) IsZero() bool { return r == nil }

/*
Open calls *[bolt.DB.Open] and bears an identical signature. The receiver
MUST be initialized beforehand.
*/
func (r *pDB) Open(options DBOptions) error {
	if len(options.File) == 0 {
		return ErrNilInput
	}

	opts := &bolt.Options{
		Timeout:         options.Timeout,
		NoGrowSync:      options.NoGrowSync,
		ReadOnly:        options.ReadOnly,
		MmapFlags:       options.MmapFlags,
		InitialMmapSize: options.InitialMmapSize,
	}

	db, err := bolt.Open(options.File, options.Mode, opts)
	if err == nil {
		db.NoSync = options.NoSync
		r.db = db
	}

	return err
}

/*
Update calls *[bolt.DB.Update] and bears an identical signature.
*/
func (r *pDB) Update(fn func(tx Tx) error) (err error) {
	return r.db.Update(func(btx *bolt.Tx) error {
		return fn(&pTx{ll: r.ll, tx: btx})
	})
}

/*
eDB implements a faux *[bolt.DB] instance which actually manifests
as a runtime-specific map instance, intended for ephemeral, non-persistent
content such as the cn=Monitor DIT. Instances of this type qualify the
[Database] interface signature.
*/
type eDB struct {
	bk   map[string]*eBucket
	ll   *BitValue
	lock *sync.RWMutex
	txid int
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *eDB) IsZero() bool {
	return r == nil
}

/*
Batch performs no useful task, as batch operations are only supported
on Persistent databases. This method exists solely to satisfy Go's
interface signature requirements with respect to [Database].
*/
func (r *eDB) Batch(fn func(Tx) error) error {
	return r.Update(func(tx Tx) error {
		return fn(tx)
	})
}

/*
Type returns 1, or 0 if uninitialized. A value of 1 indicates the
receiver is Ephemeral in nature.
*/
func (r *eDB) Type() int {
	var i int
	if !r.IsZero() {
		i = 1
	}
	return i
}

/*
Close always returns a nil error, as this method is not applicable for the
receiver instance type.  This method exists only to satisfy Go's interface
signature requirements with respect to the [Database] type.
*/
func (r *eDB) Close() error { return nil }

/*
Path returns the path to the currently open Database.
*/
func (r *eDB) Path() string { return "" }

/*
SetLoglevel assigns the input *[Loglevel] instance to the receiver.
*/
func (r *eDB) SetLoglevel(ll *BitValue) { r.ll = ll }

/*
Open initializes a new ephemeral map instance within the receiver.  The
receiver MUST be initialized beforehand via new(eDB).  When calling
the package level [newE] function, it is not necessary to call this method.

All input variables are intentionally shadowed, and are present only to
satify Go's interface signature requirements with respect to the [Database]
type.

The returned error instance shall always be nil and may be safely shadowed.

In general, it is more convenient to call [newE] than to use this method.
*/
func (r *eDB) Open(_ DBOptions) error {
	ll := new8BitValue()
	var err error
	if !r.IsZero() {
		if r.bk == nil {
			r.lock = &sync.RWMutex{}
			r.bk = make(map[string]*eBucket, 0)
			r.ll = &ll
		}
	} else {
		err = ErrNotInitialized
	}

	return err
}

/*
Update returns an error following an attempt to execute fn to commit [Tx] tx.
*/
func (r *eDB) Update(fn func(tx Tx) error) error {
	if r.IsZero() {
		return ErrNilReceiver
	}
	r.lock.Lock()         // acquire write lock
	defer r.lock.Unlock() // release write lock when function returns
	r.txid += 1

	mtx := &eTx{
		ll:       r.ll,
		txid:     r.txid,
		store:    r,
		bk:       r.bk,
		writable: true}
	return fn(mtx)
}

/*
View returns an error following an attempt to begin a new view transaction.
*/
func (r *eDB) View(fn func(tx Tx) error) error {
	if r.IsZero() {
		return ErrNilReceiver
	}
	r.lock.RLock()         // acquire read lock
	defer r.lock.RUnlock() // release read lock when function returns
	r.txid += 1

	mtx := &eTx{
		ll:       r.ll,
		txid:     r.txid,
		store:    r,
		bk:       r.bk,
		writable: false}
	return fn(mtx)
}

/*
Begin returns an instance of [Tx] alongside an error following an attempt
to begin a new transaction.
*/
func (r *eDB) Begin(writable bool) (Tx, error) {
	r.txid += 1
	return &eTx{
		ll:       r.ll,
		txid:     r.txid,
		store:    r,
		bk:       r.bk,
		writable: writable,
		lock:     r.lock}, nil
}
