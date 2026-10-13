//go:build !dib_debug

package dib

type NullLogger struct{}

var EventLogger Logger = NullLogger{}

func (_ NullLogger) Printf(_ string, _ ...any) {}
func (_ NullLogger) Fatalf(_ string, _ ...any) {}

func emitW(_ *BitValue, _ string, _ ...any) {}
func emitE(_ *BitValue, _ string, _ ...any) {}
func emitT(_ *BitValue, _ string, _ ...any) {}
func emitErr(_ *BitValue, _ error)          {}
