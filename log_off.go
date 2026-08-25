//go:build !dib_debug

package dib

import (
	"github.com/go-directory/common"
)

type NullLogger struct{}

var EventLogger Logger = NullLogger{}

func (_ NullLogger) Printf(_ string, _ ...any) {}
func (_ NullLogger) Fatalf(_ string, _ ...any) {}

func emitW(_ *common.BitValue, _ string, _ ...any) {}
func emitE(_ *common.BitValue, _ string, _ ...any) {}
func emitT(_ *common.BitValue, _ string, _ ...any) {}
func emitErr(_ *common.BitValue, _ error)          {}
