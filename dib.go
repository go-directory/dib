package dib

/*
dib.go contains types and methods pertaining to the
X.501 Directory Information Base.
*/

import (
	"bytes"

	"github.com/go-directory/config"
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
	Loglevel *BitValue
	Config   *config.Config
}

/*
New returns a freshly initialized instance of *[DirectoryInformationBase].
*/
func New(cfg *config.Config) *DirectoryInformationBase {
	ll := new8BitValue()
	d := &DirectoryInformationBase{
		BuiltIns: make(map[string]*DirectoryInformationTree),
		Config:   cfg,
		Loglevel: &ll,
	}
	d.Loglevel.SetNamesMap(map[int]string{
		logError:   "ERROR",   // 1
		logWarning: "WARNING", // 2
		logTrace:   "TRACE",   // 4
	})

	if err := verifyRuntimeConfig(cfg); err != nil {
		d.emitE(err.Error())
		panic(err)
	}

	d.emitT("DIB init")
	return d
}

/*
CreateOrOpenDIT registers and returns a root instance of *[DirectoryInformationTree].

To extend the return value to include glued subordinate contexts, see the
[DirectoryInformationTree.CreateOrOpenGlueContext] method.
*/
func (r *DirectoryInformationBase) CreateOrOpenDIT(suffix LDAPDN) (dit *DirectoryInformationTree) {
	if _, found := r.BuiltIns[string(blc(suffix))]; found {
		r.emitE("CreateOrOpenDIT: suffix already exists: %q", suffix)
		return
	}

	entry, ok := r.Config.DIB.DIT(suffix)
	if !ok {
		r.emitE("CreateOrOpenDIT: suffix '", string(suffix), "' not found within cn=dib,cn=config")
		return
	}

	var base Database
	var err error
	if base, err = NewDatabase(entry.Type); err == nil {
		base.SetLoglevel(r.Loglevel)
		var dn DistinguishedName
		if dn, err = newDN(suffix, true); err == nil {
			dn.Boundary = uint(bytes.Count(suffix, []byte(`,`))) // TODO: skip escaped commas
			dit = &DirectoryInformationTree{
				db:   base,
				dib:  r,
				dir:  string(entry.DataDir),
				typ:  entry.Type,
				root: contextProfile{dn: dn},
			}

			if err = dit.open(entry); err == nil {
				r.emitT("CreateOrOpenDIT: dit %q (%s): open %q",
					dit.root.dn.Normal, entry.Name, dit.dir)
				r.DITs = append(r.DITs, dit)
			}
		}
	}

	if err != nil {
		r.emitE("CreateOrOpenDIT: failed to create or open %q (%s): %v",
			entry.Suffix, entry.Name, err.Error())
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
func (r *DirectoryInformationBase) DIT(suffix LDAPDN) (dit *DirectoryInformationTree) {
	dn, err := newDN(suffix, true)
	if err != nil {
		r.emitE("DIT: invalid DN %q: %v", suffix, err.Error())
		return
	}

	lsfx := lc(suffix.String())
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
