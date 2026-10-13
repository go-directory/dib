package dib

import (
	"github.com/go-directory/syntax"
)

type (
	DistinguishedName = syntax.DistinguishedName
	LDAPDN            = syntax.LDAPDN
)

var newDN = syntax.NewDistinguishedName
