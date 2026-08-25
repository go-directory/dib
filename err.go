package dib

import (
	"errors"
	"os"
)

var (
	ErrAttrSingleValue error = errors.New("Attribute is SINGLE-VALUE")
	ErrBadDN           error = errors.New("Bad DN")
	ErrBadType         error = errors.New("Inappropriate input type")
	ErrBucketExists    error = errors.New("Bucket already exists")
	ErrBucketNotFound  error = errors.New("Bucket not found")
	ErrConsViolation   error = errors.New("constraintViolation")
	ErrDupAddValue     error = errors.New("Attribute value specified more than once")
	ErrEntryExists     error = errors.New("Entry already exists")
	ErrInvalidSyntax   error = errors.New("Invalid value per syntax")
	ErrLockFailed      error = errors.New("Unable to acquire lock")
	ErrModsNotAllowed  error = errors.New("Modifications not allowed")
	ErrNamingViolation error = errors.New("namingViolation")
	ErrNilBackend      error = errors.New("Nil backend")
	ErrNilFn           error = errors.New("Nil func/meth")
	ErrNilInput        error = errors.New("Nil input")
	ErrNilReceiver     error = errors.New("Nil receiver")
	ErrNilSchema       error = errors.New("Nil or zero schema")
	ErrNilTx           error = errors.New("tx closed")
	ErrNoSuchObject    error = errors.New("No such object")
	ErrNoSuchValue     error = errors.New("No such value")
	ErrNoSupEntry      error = errors.New("No superior entry")
	ErrNonLeafEntry    error = errors.New("Non-leaf entry")
	ErrNotExist        error = os.ErrNotExist
	ErrNotImplemented  error = errors.New("Function not implemented")
	ErrNotInitialized  error = errors.New("Instance Not initialized -- hint: new()?")
	ErrNotWritable     error = errors.New("Bucket not writable")
	ErrOCViolation     error = errors.New("objectClassViolation")
	ErrUnknownAttr     error = errors.New("Undefined attribute type")
	ErrUnknownChOp     error = errors.New("Unknown change operation")
	ErrUnknownClass    error = errors.New("Undefined object class")
	ErrUnknownOID      error = errors.New("Unknown numeric OID")
	ErrValueExists     error = errors.New("Value already exists")
	ErrZeroKey         error = errors.New("Zero key")
)

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

	return errors.New(uerr.Error() + n)
}
