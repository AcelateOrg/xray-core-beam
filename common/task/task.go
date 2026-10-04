package task

import (
	"context"
	goerrors "errors"

	"github.com/xtls/xray-core/common/crash"
	"github.com/xtls/xray-core/common/signal/semaphore"
)

// errTaskPanicked fails the task group when one of its tasks panicked: the
// connection it carries ends instead of the process (see common/crash).
var errTaskPanicked = goerrors.New("task panicked")

func runContained(f func() error) (err error) {
	defer crash.RecoverThen("task", func() { err = errTaskPanicked })
	return f()
}

// OnSuccess executes g() after f() returns nil.
func OnSuccess(f func() error, g func() error) func() error {
	return func() error {
		if err := f(); err != nil {
			return err
		}
		return g()
	}
}

// Run executes a list of tasks in parallel, returns the first error encountered or nil if all tasks pass.
func Run(ctx context.Context, tasks ...func() error) error {
	n := len(tasks)
	s := semaphore.New(n)
	done := make(chan error, 1)

	for _, task := range tasks {
		<-s.Wait()
		go func(f func() error) {
			err := runContained(f)
			if err == nil {
				s.Signal()
				return
			}

			select {
			case done <- err:
			default:
			}
		}(task)
	}

	/*
		if altctx := ctx.Value("altctx"); altctx != nil {
			ctx = altctx.(context.Context)
		}
	*/

	for i := 0; i < n; i++ {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-s.Wait():
		}
	}

	/*
		if cancel := ctx.Value("cancel"); cancel != nil {
			cancel.(context.CancelFunc)()
		}
	*/

	return nil
}
