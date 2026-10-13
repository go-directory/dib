package dib

/*
dit.go contains types and methods pertaining to
the X.501 Directory Information Tree.
*/

import (
	"bytes"

	"github.com/gofrs/flock"
)

/*
DirectoryInformationTree implements the directory information tree, or "DIT",
defined in [ITU-T Rec. X.501 cl. 7.4].

An instance of [DirectoryInformationTree] shall possess only one root context,
but may contain zero or more glued subordinates. For example, an appropriate
root context may be "o=example", while its glued subordinates should consist
of dedicated branches such as "ou=People,o=example". It is inappropriate to
assign unrelated branches, such as "ou=People,o=acme". Attempts to do this
shall result in an error.

Use of glued subordinates is only practical in cases where a single subordinate
branch, such as "ou=People,o=example" is considered too large on its own to
reside in a single backend instance that is already home to other branches.

[ITU-T Rec. X.501 cl. 7.4]: https://www.itu.int/rec/T-REC-X.501
*/
type DirectoryInformationTree struct {
	dib    *DirectoryInformationBase            // pointer reference to top-level context
	glue   map[string]*DirectoryInformationTree // glued subordinates
	db     Database                             // actual backend database
	dir    string                               // data directory (persistent only)
	typ    int                                  // ephemeral[1], persistent[2]
	lock   *flock.Flock                         // for persistent (file-based) instances only
	root   contextProfile                       // information about the suffix
	parent *DirectoryInformationTree            // if DIT is glued, reference its parent
	numSub uint8                                // subordinates policy: 0:hasSub+numSub, 1:numSub, 2:hasSub
}

func (r *DirectoryInformationTree) Suffix() string {
	return string(r.root.dn.Case)
}

func (r DirectoryInformationTree) DN() DistinguishedName {
	return r.root.dn
}

func (r DirectoryInformationTree) ContextOffset() uint { return r.root.dn.Boundary }

func (r DirectoryInformationTree) Type() int { return r.typ }

func (r DirectoryInformationTree) Parent() *DirectoryInformationTree {
	return r.parent
}

func (r DirectoryInformationTree) lockfile() string { return r.lock.Path() }
func (r DirectoryInformationTree) path() string     { return r.db.Path() }
func (r *DirectoryInformationTree) IsGlue() bool    { return r.parent != nil }

/*
Db returns the underlying [Database] stored in the receiver
instance.
*/
func (r DirectoryInformationTree) Db() Database { return r.db }

/*
CreateOrOpenGlueContext returns a new instance of *[DirectoryInformationTree] following
an attempt to open or create a subordinate "glue" suffix.

For example, if the primary backend contains the "o=example" root suffix, one can use
this method to create a dedicated subordinate "branch" for "ou=People,o=example", which
will be the return value.

If a specified [BackendType] is [Ephemeral], the backend shall always be empty upon
initialization. If the specified [BackendType] is [Persistent], an attempt to read an
existing [Database] shall be made, should one be present, else it will be created anew.
*/
func (r *DirectoryInformationTree) CreateOrOpenGlueContext(suffix LDAPDN) *DirectoryInformationTree {
	entry, ok := r.dib.Config.DIB.DIT(suffix)
	if !ok {
		emitE(r.dib.Loglevel, "CreateOrOpenGlueContext: parent of glue suffix'", string(suffix), "' not found within cn=dib,cn=config")
		return nil
	}

	var glue *DirectoryInformationTree
	var base Database
	var err error

	if base, err = NewDatabase(entry.Type); err == nil {
		var dn DistinguishedName
		if dn, err = newDN(suffix, true); err == nil {
			dn.Boundary = uint(bytes.Count(suffix, []byte(`,`)))
			glue = &DirectoryInformationTree{
				db:     base,
				parent: r,
				dir:    string(entry.DataDir),
				typ:    entry.Type,
				root:   contextProfile{dn: dn},
			}

			if err = glue.open(entry); err == nil {
				r.dib.emitT("CreateOrOpenGlueContext: dit %q (%s): open %q",
					entry.Suffix, entry.Name, glue.dir)
				if r.glue == nil {
					r.glue = make(map[string]*DirectoryInformationTree)
				}
				r.glue[string(blc(entry.Suffix))] = glue
			}
		}
	}

	if err != nil {
		r.dib.emitE("CreateOrOpenGlueContext: failed to create or open %q (%s): %v",
			entry.Suffix, entry.Name, err.Error())
	}

	return glue
}

// bucket literals
var (
	canonK  = []byte(`_canon`)  // canonical values
	id2dnK  = []byte(`_id2dn`)  // id -> orig DN
	dn2idK  = []byte(`_dn2id`)  // orig DN -> id
	nml2idK = []byte(`_nml2id`) // normalized DN -> id
	id2nmlK = []byte(`_id2nml`) // id -> normalized DN
	subentK = []byte(`_subent`) // id -> zero bytes
	c2pK    = []byte(`_c2p`)    // child ID to parent ID
	p2cK    = []byte(`_p2c`)    // parent bucket SUFFIX LITERAL (not a bucket unto itself)
)

/*
buckets defines critical database buckets.
*/
var buckets = [][]byte{
	canonK,
	id2dnK,
	dn2idK,
	nml2idK,
	id2nmlK,
	subentK,
	c2pK,
}

const (
	Null int = iota
	Ephemeral
	Persistent
)

func getBackendType(typ ...any) (t int) {
	t = Persistent
	if len(typ) == 0 {
		return // use default
	}

	switch tv := typ[0].(type) {
	case string:
		switch lc(tv) {
		case "ephemeral":
			t = Ephemeral
		}
	case int:
		if 0 <= tv && tv <= 2 {
			t = tv
		}
	}

	return
}

/*
contextProfile implements an abstract
namingContext container for access to
various DN elements such as tokenized
vs. non-tokenized, offset, structural
class and attribute governance info.
*/
type contextProfile struct {
	dn DistinguishedName
	id []byte // BE uint64 identifier
}

/*
backendType returns [Ephemeral] (1) or [Persistent] (2) depending on the
manner in which the receiver was initialized.

If the receiver was not initialized, [Null] (0) is returned.
*/
func (r DirectoryInformationTree) backendType() (t int) {
	t = Null
	if &r != nil {
		t = r.db.Type()
	}
	return
}

/*
open returns an error following an attempt to create and/or mount the
specified [backend].

If the [backend] is [Persistent] and has been previously initialized, it
is only mounted at the path indicated. In the unlikely occurrence of missing
critical [Bucket] instances in an otherwise healthy [backend], they are
silently created.

If the [backend] is [Persistent] and has not yet been initialized, a new
instance is created from scratch at the path indicated.

If the [backend] is [Ephemeral], naturally it resides in memory only --
initialized from scratch at the start of each runtime process.
*/
func (r *DirectoryInformationTree) open(cfg dITConfig) (err error) {
	datafile := string(cfg.DataFile)
	opts := DBOptions{
		File:   datafile,
		Mode:   cfg.Mode,
		NoSync: cfg.NoSync,
	}

	r.dib.emitT("DIT open (%s): file[%q] mode[%d] nosync[%t]",
		cfg.Suffix, cfg.DataFile, cfg.Mode, cfg.NoSync)

	if r.typ == Persistent {
		r.lock = flock.NewFlock(getFlockPath(datafile))
		var locked bool
		if locked, err = r.lock.TryLock(); err != nil {
			return
		} else if !locked {
			err = ErrLockFailed
			return
		}

		if err = r.db.Open(opts); err != nil {
			return
		}
	}

	if r == nil {
		err = ErrNilReceiver
		return
	}

	err = r.db.Update(func(tx Tx) (err error) {
		for i := 0; i < len(buckets) && err == nil; i++ {
			r.dib.emitT("DIT open (%s): possibly creating bucket %q",
				cfg.Suffix, buckets[i])
			_, err = tx.CreateBucketIfNotExists(buckets[i])
		}
		return
	})
	return
}

func (r *DirectoryInformationTree) Close() {
	if r.typ == Persistent {
		if err := r.db.Close(); err == nil {
			if err = r.lock.Unlock(); err != nil {
				r.dib.emitE("DIT close (%s): %v", r.root.dn.Case, err.Error())
			}
		}
	}
}
