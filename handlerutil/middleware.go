package handlerutil

import (
	"fmt"
	"net/http"
	"runtime"

	"go.uber.org/zap"
)

type Middleware struct {
	logger *zap.Logger
	debug  bool
}

func NewMiddleware(logger *zap.Logger, debug bool) *Middleware {
	return &Middleware{
		logger: logger,
		debug:  debug,
	}
}

func (m Middleware) RecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			needRecovery, errString, caller := PanicRecoveryError(recover())
			if needRecovery {
				m.logger.Error("Recovered from panic", zap.Any("error", errString), zap.String("trace", fmt.Sprintf("%s", caller)))
				if m.debug {
					for _, line := range caller {
						fmt.Printf("\t%s\n", line)
					}
				}

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next(w, r)
	}
}

func PanicRecoveryError(err any) (bool, string, []string) {
	if err == nil {
		return false, "", nil
	}

	var callers []string
	for i := 2; ; /* 1 for New() 2 for NewPanicRecoveryError */ i++ {
		_, file, line, got := runtime.Caller(i)
		if !got {
			break
		}

		callers = append(callers, fmt.Sprintf("%s:%d", file, line))
	}

	if parseErr, ok := err.(error); ok {
		return true, parseErr.Error(), callers
	}

	return true, fmt.Sprintf("%v", err), callers
}
