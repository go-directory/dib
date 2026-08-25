package dib

/*
dit.go contains types and methods pertaining to
the X.501 Directory Information Tree.
*/

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"github.com/go-directory/common"
	"github.com/go-directory/syntax"
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

func (r DirectoryInformationTree) DN() syntax.DistinguishedName {
	return r.root.dn
}

func (r DirectoryInformationTree) ContextOffset() int {
	return r.root.off
}

func (r DirectoryInformationTree) Type() int { return r.typ }

func (r DirectoryInformationTree) lockfile() string {
	return r.lock.Path()
}

func (r DirectoryInformationTree) Parent() *DirectoryInformationTree {
	return r.parent
}

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
func (r *DirectoryInformationTree) CreateOrOpenGlueContext(cfg Config) *DirectoryInformationTree {
	if !r.verify(cfg) {
		return nil
	}

	bdn := []byte(cfg.Suffix)
	var glue *DirectoryInformationTree

	var base Database
	var err error
	if base, err = NewDatabase(cfg.Type); err == nil {
		offset := bytes.Count(bdn, []byte(`,`))
		var dn syntax.DistinguishedName
		if dn, err = syntax.NewDistinguishedName(bdn, true); err == nil {
			glue = &DirectoryInformationTree{
				db:     base,
				parent: r,
				dir:    cfg.DataDir,
				typ:    cfg.Type,
				root: contextProfile{
					dn:  dn,
					off: offset,
				},
			}
			if err = glue.open(cfg); err == nil {
				r.dib.emitT("CreateOrOpenGlueContext: dit %q (%s): open %q",
					cfg.Suffix, cfg.Name, glue.dir)
				if r.glue == nil {
					r.glue = make(map[string]*DirectoryInformationTree)
				}
				r.glue[lc(cfg.Suffix)] = glue
			}
		}
	}

	if err != nil {
		r.dib.emitE("CreateOrOpenGlueContext: failed to create or open %q (%s): %v",
			cfg.Suffix, cfg.Name, err.Error())
	}

	return glue
}

func (r DirectoryInformationTree) verify(cfg Config) bool {
	var v bool = true // assume true by default
	if err := cfg.Verify(); err != nil {
		r.dib.emitE("CreateOrOpenGlueContext: %v", err.Error())
		v = false
	} else if cfg.Suffix == "" {
		r.dib.emitE("CreateOrOpenGlueContext: suffix is zero")
		v = false
	} else if !bytes.HasSuffix(blc([]byte(cfg.Suffix)), r.root.dn.Normal) {
		r.dib.emitE("CreateOrOpenGlueContext: no appropriate backend suffix from which to extend")
		v = false
	}
	// TODO: check that cfg.Suffix is *immediate* child of r.Suffix()

	return v
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

type Config struct {
	Name    string      // alphanumeric "friendly" backend name
	Suffix  string      // root suffix of backend
	DataDir string      // data directory
	Mode    os.FileMode // mode used for data file; default: 0660
	Type    int         // type of backend to open or init (1=E/2=P)
	NoSync  bool        // Use with caution; see boltdb/bolt reference.

	// misc. bit-shifted options (uint32)
	//
	// 1 - ReadOnly
	// 2
	// 4
	// 8
	// 16
	// 32
	// 64
	Opts common.BitValue

	file string // actual data file
}

func (r *Config) prepareBitValue() {
	if r.Opts.Kind() == 0x0 || r.Opts.Size() == 0 {
		// initialize bitvalue store
		r.Opts = common.NewBitValue(common.Uint32)
	} else if r.Opts.Size() != 32 {
		// bitvalue type is too small, so
		// backup whatever is already set
		// (if anything) ...
		val := r.Opts.Value()

		// Initialize new 32 bit bitvalue.
		r.Opts = common.NewBitValue(common.Uint32)

		// Shift the value we set aside.
		switch tv := val.(type) {
		case uint8:
			r.Opts.Shift(tv)
		case uint16:
			r.Opts.Shift(tv)
		}
	}
}

/*
Verify returns an error following an analysis of the
receiver instance.
*/
func (r *Config) Verify() (err error) {
	r.prepareBitValue()

	if r.Name == "" {
		err = errors.New("Config MUST have a unique alphanumeric name")
		return
	}

	if ownerCanRead := r.Mode&0400 != 0; !ownerCanRead {
		err = errors.New("Owner MUST have read privileges at a minimum")
		return
	} else if ownerCanWrite := r.Mode&0600 != 0; !ownerCanWrite {
		r.Opts.Shift(1) // enact READ-ONLY
	}

	switch r.Type {
	case Persistent:
		if r.DataDir == "" {
			err = errors.New("Invalid Config.DataDir value for Persistent database")
			return
		}
		var exists bool
		if exists, err = fileExists(r.DataDir, true); !exists {
			return
		}

		if r.Name == "rootdse" {
			err = errors.New("rootdse Config.Type MUST be Ephemeral")
			return
		}

		if r.Suffix == "" {
			err = errors.New("Empty suffix can only serve rootdse")
			return
		}

		r.file = strings.TrimRight(r.DataDir, `/`) + `/` + r.Name + `/data.db`
		os.Mkdir(r.DataDir+`/`+r.Name, r.Mode+0100)

	case Ephemeral:
		if r.Name == "rootdse" {
			if r.Suffix != "" {
				err = errors.New("Inappropriate rootdse suffix (should be empty)")
				return
			}
		}
	default:
		err = errors.New("Invalid Config.Type value; must be Persistent or Ephemeral")
	}

	return
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
	dn  syntax.DistinguishedName
	id  []byte // BE uint64 identifier
	off int    // context delim. offset
	soc string // structuralObjectClass numeric oid
	gsr string // governingStructureRule number
	acr string // applicable dITContentRule
}

func (r DirectoryInformationTree) path() string {
	return r.db.Path()
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
func (r *DirectoryInformationTree) open(cfg Config) (err error) {
	opts := DBOptions{
		File:   cfg.file,
		Mode:   cfg.Mode,
		NoSync: cfg.NoSync,
	}

	r.dib.emitT("DIT open (%s): file[%q] mode[%d] nosync[%t]",
		cfg.Suffix, cfg.file, cfg.Mode, cfg.NoSync)

	if r.typ == Persistent {
		r.lock = flock.NewFlock(getFlockPath(cfg.file))
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

func (r *DirectoryInformationTree) IsGlue() bool {
	return r.parent != nil
}
