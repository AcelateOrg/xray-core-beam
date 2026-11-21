// Package crash contains panics raised on the core's connection goroutines.
//
// Beam runs the core inside the app process. In Go a panic that is not
// recovered on its own goroutine ends the whole process, so a nil dereference
// in one protocol handler used to take the window, the tunnel and the kill
// switch down with it. The goroutines that carry traffic defer Recover, which
// turns such a panic into a failed connection and reports it through the hook.
//
// It must not import anything from the core: low-level packages (buf, mux,
// task) depend on it.
package crash

import (
	"runtime/debug"
	"sync/atomic"
)

// Hook receives every contained panic: where it happened, the value and the
// goroutine's stack. It runs on the goroutine that panicked and must not block.
type Hook func(where string, value any, stack []byte)

var (
	hook  atomic.Pointer[Hook]
	count atomic.Int64
)

// SetHook installs the reporter; nil removes it.
func SetHook(h Hook) {
	if h == nil {
		hook.Store(nil)
		return
	}
	hook.Store(&h)
}

// Count is the number of panics contained since the process started.
func Count() int64 { return count.Load() }

// Recover contains a panic on the current goroutine. Defer it directly:
//
//	defer crash.Recover("mux client")
func Recover(where string) {
	if r := recover(); r != nil {
		report(where, r)
	}
}

// RecoverThen is Recover that also runs cleanup (closing the connection's
// links, setting an error result) after a panic. Defer it directly.
func RecoverThen(where string, cleanup func()) {
	if r := recover(); r != nil {
		report(where, r)
		if cleanup != nil {
			func() {
				// A cleanup that panics on a half-built connection must not
				// undo the containment.
				defer func() { _ = recover() }()
				cleanup()
			}()
		}
	}
}

// Go runs f on a new goroutine with Recover deferred.
func Go(where string, f func()) {
	go func() {
		defer Recover(where)
		f()
	}()
}

func report(where string, value any) {
	count.Add(1)
	h := hook.Load()
	if h == nil {
		return
	}
	stack := debug.Stack()
	func() {
		defer func() { _ = recover() }()
		(*h)(where, value, stack)
	}()
}
