package dib

import (
	"errors"
	"os"

	"github.com/go-directory/common"
)

var (
	ErrBucketExists   = backendError("Bucket already exists")
	ErrBucketNotFound = backendError("Bucket not found")
	ErrLockFailed     = backendError("Unable to acquire lock")
	ErrNilBackend     = backendError("Nil backend")
	ErrNilInput       = backendError("Nil input")
	ErrNilReceiver    = backendError("Nil receiver")
	ErrNilTx          = backendError("tx closed")
	ErrNotExist       = os.ErrNotExist
	ErrNotImplemented = backendError("Function not implemented")
	ErrNotInitialized = backendError("Instance Not initialized -- hint: new()?")
	ErrNotWritable    = backendError("Bucket not writable")
)

var errorIs = errors.Is

func errBucketName(uerr error, name any) (err error) {
	var n string = `: `
	var def string = `<unspecified>`
	switch tv := name.(type) {
	case string:
		if len(tv) == 0 {
			tv = def
		}
		n += tv
	case []byte:
		if len(tv) == 0 {
			tv = []byte(def)
		}
		n += string(tv)
	default:
		err = uerr
		return
	}

	return backendError(uerr.Error(), n)
}

func backendError(msg ...string) error {
	m := append([]string{"Backend error: "}, msg...)
	return common.ErrorBackend.New(m...)
}
