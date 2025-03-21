package recovery

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// Go wraps a `go func()` with recover()
func Go(handler func(), recoverHandler func(r interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "%s goroutine panic: %v\n%s\n", time.Now(), r, string(debug.Stack()))

				if recoverHandler != nil {
					go func() {
						defer func() {
							if p := recover(); p != nil {
								fmt.Fprintf(os.Stderr, "recover goroutine panic:%v\n%s\n", p, string(debug.Stack()))
							}
						}()

						recoverHandler(r)
					}()
				}
			}
		}()
		handler()
	}()
}
