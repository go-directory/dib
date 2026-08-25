//go:build dib_debug

package dib

import (
	"log"

	"github.com/go-directory/common"
)

type ConsoleLogger struct{}

var EventLogger Logger = ConsoleLogger{}

func (_ ConsoleLogger) Printf(s string, args ...any) { log.Printf(s, args...) }
func (_ ConsoleLogger) Fatalf(s string, args ...any) { log.Fatalf(s, args...) }

func emitW(l *common.BitValue, m string, v ...any) {
	if l.Positive(logWarning) {
		EventLogger.Printf("[WARN] "+m, v...)
	}
}

func emitE(l *common.BitValue, m string, v ...any) {
	if l.Positive(logError) {
		EventLogger.Printf("[ERROR] "+m, v...)
	}
}

func emitT(l *common.BitValue, m string, v ...any) {
	if l.Positive(logTrace) {
		EventLogger.Printf("[TRACE] "+m, v...)
	}
}

// for pre-crafted errors
func emitErr(l *common.BitValue, err error) {
	if l.Positive(logError) && err != nil {
		EventLogger.Printf("[ERROR] " + err.Error())
	}
}
