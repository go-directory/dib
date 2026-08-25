package dib

/*
dib.go contains types and methods pertaining to the
X.501 Directory Information Base.
*/

import (
	"bytes"

	"github.com/go-directory/common"
	"github.com/go-directory/config"
	"github.com/go-directory/syntax"
)

/*
DirectoryInformationBase implements the directory information base, or "DIB",
defined in [ITU-T Rec. X.501 cl. 7].

Instances of this type are intended to store one or more parallel (sibling)
instances, i.e.: 'o=example' or 'cn=Monitor', of [DirectoryInformationTree]
which exist on a single DSA. New root trees can be created using the
[DirectoryInformationBase.CreateOrOpenDIT] method.

Glue contexts -- for the purpose of partitioning a certain "branch" of a DIT
to its own dedicated backend while remaining logically subordinate to its
parent -- are supported. New glue contexts can be created using the
[DirectoryInformationTree.CreateOrOpenGlueContext] method.

A DSA is expected to only operate a single instance of this type.

[ITU-T Rec. X.501 cl. 7]: https://www.itu.int/rec/T-REC-X.501
*/
type DirectoryInformationBase struct {
	DITs     []*DirectoryInformationTree
	BuiltIns map[string]*DirectoryInformationTree
	Loglevel *common.BitValue
	Config   *config.Config
}

/*
New returns a freshly initialized instance of *[DirectoryInformationBase].
*/
func New() *DirectoryInformationBase {
	ll := common.NewBitValue(common.Uint8)
	d := &DirectoryInformationBase{
		BuiltIns: make(map[string]*DirectoryInformationTree),
		Loglevel: &ll,
	}
	d.Loglevel.SetNamesMap(map[int]string{
		logError:   "ERROR",   // 1
		logWarning: "WARNING", // 2
		logTrace:   "TRACE",   // 4
	})
	d.emitT("DIB init")
	return d
}

/*
CreateOrOpenDIT registers and returns a root instance of *[DirectoryInformationTree].

To extend the return value to include glued subordinate contexts, see the
[DirectoryInformationTree.CreateOrOpenGlueContext] method.
*/
func (r *DirectoryInformationBase) CreateOrOpenDIT(cfg Config) (dit *DirectoryInformationTree) {
	if _, found := r.BuiltIns[lc(cfg.Suffix)]; found {
		r.emitE("CreateOrOpenDIT: suffix already exists: %q", cfg.Suffix)
		return
	}

	err := cfg.Verify()
	if err == nil {
		bdn := []byte(cfg.Suffix)

		var base Database
		if base, err = NewDatabase(cfg.Type); err == nil {
			base.SetLoglevel(r.Loglevel)
			offset := bytes.Count(bdn, []byte(`,`)) // TODO: skip escaped commas
			var dn syntax.DistinguishedName
			if dn, err = syntax.NewDistinguishedName(bdn, true); err == nil {
				dit = &DirectoryInformationTree{
					db:  base,
					dib: r,
					dir: cfg.DataDir,
					typ: cfg.Type,
					root: contextProfile{
						dn:  dn,
						off: offset,
					},
				}
				if err = dit.open(cfg); err == nil {
					r.emitT("CreateOrOpenDIT: dit %q (%s): open %q",
						dit.root.dn.Normal, cfg.Name, dit.dir)
					r.DITs = append(r.DITs, dit)
				}
			}
		}
	}

	if err != nil {
		r.emitE("CreateOrOpenDIT: failed to create or open %q (%s): %v",
			cfg.Suffix, cfg.Name, err.Error())
	}

	return
}

/*
DIT returns an instance of *[DirectoryInformationTree] associated with the input
suffix string argument.

This method is mainly used to access a specific [DirectoryInformationTree] based
on a particular subordinate entry residing within.  Depending on the depth of
the suffix argument, this method may be costly to use in high volume.

If the specific root suffix is already known, simply call it from the underlying
"DITs" field map residing within the receiver instance as opposed to calling this
method.
*/
func (r *DirectoryInformationBase) DIT(suffix string) (dit *DirectoryInformationTree) {
	dn, err := syntax.NewDistinguishedName([]byte(suffix), true)
	if err != nil {
		r.emitE("DIT: invalid DN %q: %v", suffix, err.Error())
		return
	}

	lsfx := lc(suffix)
	if bi, found := r.BuiltIns[lsfx]; found {
		r.emitT("DIT: Matched built-in %q", lsfx)
		dit = bi
		return
	}

	for k, v := range r.BuiltIns {
		r.emitT("DIT: scanning built-in %q", k)
		if k != "" {
			if bytes.HasSuffix(dn.Normal, v.root.dn.Normal) {
				r.emitT("DIT: matched built-in %q to %q", k, v.root.dn.Normal)
				dit = v
				return
			}
		}
	}

	var recurse func(*DirectoryInformationTree) *DirectoryInformationTree
	recurse = func(tree *DirectoryInformationTree) (out *DirectoryInformationTree) {
		// if this tree’s root is not an ancestor of the
		// requested DN, nothing under it can match.
		if !bytes.HasSuffix(dn.Normal, tree.root.dn.Normal) {
			return nil
		}

		// try glued children first: deepest match wins.
		for _, glue := range tree.glue {
			r.emitT("DIT: recursing glue %q", glue.root.dn.Normal)
			if out = recurse(glue); out != nil {
				return out
			}
		}

		r.emitT("DIT: no glue context matched; fallback to %q",
			tree.root.dn.Normal)

		return tree
	}

	// Iterate through all DITs within the receiver DIB,
	// call recurse on each one of them.
	for i := 0; i < len(r.DITs) && dit == nil; i++ {
		r.emitT("DIT: recursing %q", r.DITs[i].root.dn.Normal)
		dit = recurse(r.DITs[i])
	}

	if dit == nil {
		r.emitW("DIT: found no appropriate match for %q", dn.Normal)
	}

	return
}

func (r DirectoryInformationBase) emitW(m string, v ...any) { emitW(r.Loglevel, m, v...) }
func (r DirectoryInformationBase) emitE(m string, v ...any) { emitE(r.Loglevel, m, v...) }
func (r DirectoryInformationBase) emitT(m string, v ...any) { emitT(r.Loglevel, m, v...) }
