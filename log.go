package dib

/*
Logger is qualified through facilities bearing the methos below.
A given qualifier of this interface can be made active via the
[EventLogger] global variable.

By default, the [NullLogger] is set. This means that no events
will be emitted to the console.
*/
type Logger interface {
	Printf(string, ...any)
	Fatalf(string, ...any)
}

const (
	// not to exceed uint8
	logError   int = 1 << iota // 1
	logWarning                 // 2
	logTrace                   // 4
)
