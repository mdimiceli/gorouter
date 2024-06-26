package handlers

import (
	"net/http"
	"time"

	"github.com/mdimiceli/gorouter/metrics"
	"github.com/mdimiceli/gorouter/proxy/utils"

	"github.com/mdimiceli/gorouter/logger"
	"go.uber.org/zap"
	"github.com/urfave/negroni/v3"
)

type reporterHandler struct {
	reporter metrics.ProxyReporter
	logger   logger.Logger
}

// NewReporter creates a new handler that handles reporting backend
// responses to metrics and missing Content-Length header
func NewReporter(reporter metrics.ProxyReporter, logger logger.Logger) negroni.Handler {
	return &reporterHandler{
		reporter: reporter,
		logger:   logger,
	}
}

// ServeHTTP handles reporting the response after the request has been completed
func (rh *reporterHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	logger := LoggerWithTraceInfo(rh.logger, r)
	requestInfo, err := ContextRequestInfo(r)
	// logger.Panic does not cause gorouter to exit 1 but rather throw panic with
	// stacktrace in error log
	if err != nil {
		logger.Panic("request-info-err", zap.Error(err))
		return
	}
	if r.Header.Get("Content-Length") == "" {
		rh.reporter.CaptureMissingContentLengthHeader()
	}

	next(rw, r)

	if requestInfo.RouteEndpoint == nil {
		return
	}

	proxyWriter := rw.(utils.ProxyResponseWriter)
	rh.reporter.CaptureRoutingResponse(proxyWriter.Status())

	if requestInfo.AppRequestFinishedAt.Equal(time.Time{}) {
		return
	}
	rh.reporter.CaptureRoutingResponseLatency(
		requestInfo.RouteEndpoint, proxyWriter.Status(),
		requestInfo.ReceivedAt, requestInfo.AppRequestFinishedAt.Sub(requestInfo.ReceivedAt),
	)
}
