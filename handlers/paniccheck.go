package handlers

import (
	"code.cloudfoundry.org/gorouter/common/health"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/gorouter/logger"
	"github.com/uber-go/zap"
	"github.com/urfave/negroni"
)

type panicCheck struct {
	health *health.Health
	logger logger.Logger
}

// NewPanicCheck creates a handler responsible for checking for panics and setting the Healthcheck to fail.
func NewPanicCheck(health *health.Health, logger logger.Logger) negroni.Handler {
	return &panicCheck{
		health: health,
		logger: logger,
	}
}

func (p *panicCheck) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer func() {
		if rec := recover(); rec != nil {
			switch rec {
			case http.ErrAbortHandler:
				// The ErrAbortHandler panic occurs when a client goes away in the middle of a request
				// this is a panic we expect to see in normal operation and is safe to allow the panic
				// http.Server will handle it appropriately
				panic(http.ErrAbortHandler)
			default:
				err, ok := rec.(error)
				if !ok {
					err = fmt.Errorf("%v", rec)
				}
				p.logger.Error("panic-check", zap.Nest("error", zap.Error(err), zap.Stack()))

				p.health.SetHealth(health.Degraded)

				rw.WriteHeader(http.StatusServiceUnavailable)
				r.Close = true
			}
		}
	}()

	next(rw, r)
}
