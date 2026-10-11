package dib

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/*
gtime returns the []byte representation of a generalized time value.
*/
func gtime() []byte {
	return []byte(time.Now().Format(`20060102150405`) + `Z`)
}

func t2gt(t time.Time) []byte {
	return []byte(t.Format(`20060102150405`) + `Z`)
}

/*
mCondA is a cheap means to reduce cyclomatic complexity
where two bool comparisons are to be performed under an
"and" context.
*/
func mCondA(a, b bool) bool { return a && b }

/*
mCondO is a cheap means to reduce cyclomatic complexity
where two bool comparisons are to be performed under an
"or" context.
*/
func mCondO(a, b bool) bool { return a || b }

func isUpper(r rune) bool { return 'A' <= r && r <= 'Z' }

/*
blc returns a fast ASCII case-normalized string based
on the in input argument.
*/
func lc(in string) string {
	bld := &strings.Builder{}
	for i := 0; i < len(in); i++ {
		if c := rune(in[i]); isUpper(c) {
			c = rune(in[i]) + 32
			bld.WriteRune(c)
		} else {
			bld.WriteRune(c)
		}
	}

	return bld.String()
}

/*
blc returns a fast ASCII case-normalized []byte slice
based on the in input argument.
*/
func blc(in []byte) []byte {
	bld := &bytes.Buffer{}
	for i := 0; i < len(in); i++ {
		if c := rune(in[i]); isUpper(c) {
			c = rune(in[i]) + 32
			bld.WriteRune(c)
		} else {
			bld.WriteRune(c)
		}
	}

	return bld.Bytes()
}

func getFlockPath(path string) string {
	path = strings.TrimSpace(path)
	dir := getPathDir(path)
	chop := chopExtension(getPathBase(path))

	return dir + "/" + chop + ".lock"
}

func getPathDir(path string) (dir string) {
	return filepath.Dir(path)
}

func getPathBase(path string) (base string) {
	return filepath.Base(path)
}

func chopExtension(base string) (chopped string) {
	// default: if no extension, just hand it back.
	chopped = base

	if sp := strings.Split(base, `.`); len(sp) > 1 {
		// there was an extension, so
		// return everything but.
		chopped = strings.Join(sp[:len(sp)-1], `.`)
	}

	return
}

func fileExists(name string, mustExist ...bool) (exists bool, err error) {
	var must bool
	if len(mustExist) > 0 {
		must = mustExist[0]
	}

	if _, err = os.Stat(name); err == nil {
		exists = true
	} else if errorIs(err, ErrNotExist) {
		if !must {
			// nullify error, since it isn't
			// a "problem" the file does not
			// exist.
			err = nil
		}
	}

	return
}

func keyString(k []byte) string {
	b := make([]byte, len(k))
	copy(b, k)
	return string(b)
}

/*
EncodeKey returns the big endian form of eid, attr and sub
concatenation (in that order).

The return value shall always be no less than eight (8)
bytes in length.

The structure of a key is as follows:

	 BE-ID   BE-AT-LEN  AT-OID   [VALUE]
	 -----   ---------  ------  ----------
	8 bytes   4 bytes   AT-LEN  0 <= bytes

If attr is nil, only eid is returned as BE bytes (8).

See also [DecodeKey].
*/
func EncodeKey(eid uint64, attr, val []byte) []byte {
	if attr == nil {
		return itob(eid)
	}
	atl := len(attr)
	alloc := 12 + atl

	end := make([]byte, 4)
	k := make([]byte, 8+4+atl+len(val))

	binary.BigEndian.PutUint32(end, uint32(atl))
	binary.BigEndian.PutUint64(k[0:8], eid)

	copy(k[8:12], end)
	copy(k[12:alloc], attr)
	copy(k[alloc:], val)
	return k
}

/*
DecodeKey returns eid, aid and sub following an attempt to
decode the input big endian b.

See also [EncodeKey].
*/
func DecodeKey(b []byte) (eid uint64, attr, val []byte) {
	if len(b) == 8 {
		// ID only
		eid = btoi(b[0:8])
	} else if len(b) >= 12 {
		eid = btoi(b[0:8])
		mark := int(binary.BigEndian.Uint32(b[8:12])) + 12
		attr = b[12:mark]
		val = b[mark:]
	}
	return
}

func encodeByteSlices(ss [][]byte) []byte {
	total := 4 // count
	for _, s := range ss {
		total += 4 + len(s)
	}

	buf := make([]byte, 0, total)
	tmp := make([]byte, 4)

	binary.BigEndian.PutUint32(tmp, uint32(len(ss)))
	buf = append(buf, tmp...)

	for _, b := range ss {
		binary.BigEndian.PutUint32(tmp, uint32(len(b)))
		buf = append(buf, tmp...)
		buf = append(buf, b...)
	}
	return buf
}

func decodeByteSlices(b []byte) (out [][]byte, err error) {
	var pos int
	if len(b) < 4 || pos+4 > len(b) {
		return
	}
	count := int(binary.BigEndian.Uint32(b[pos : pos+4]))
	pos += 4

	out = make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		if pos+4 > len(b) {
			err = backendError("decode error: truncated slice")
			break
		}
		l := int(binary.BigEndian.Uint32(b[pos : pos+4]))
		pos += 4
		if mCondO(l < 0, pos+l > len(b)) {
			err = backendError("decode error: truncated slice")
			break
		}
		out = append(out, b[pos:pos+l])
		pos += l
	}

	return
}

/*
itob returns an 8-byte Big-Endian representation of v.
*/
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

/*
btoi returns the Big-Endian uint64 form of v.
*/
func btoi(v []byte) uint64 {
	return binary.BigEndian.Uint64(v[:])
}
