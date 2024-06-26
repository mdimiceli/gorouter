package proxy

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mdimiceli/gorouter/common/health"

	"github.com/cloudfoundry/dropsonde"
	"go.uber.org/zap"
	"github.com/urfave/negroni/v3"

	"github.com/mdimiceli/gorouter/accesslog"
	router_http "github.com/mdimiceli/gorouter/common/http"
	"github.com/mdimiceli/gorouter/config"
	"github.com/mdimiceli/gorouter/errorwriter"
	"github.com/mdimiceli/gorouter/handlers"
	"github.com/mdimiceli/gorouter/logger"
	"github.com/mdimiceli/gorouter/metrics"
	"github.com/mdimiceli/gorouter/proxy/fails"
	"github.com/mdimiceli/gorouter/proxy/round_tripper"
	"github.com/mdimiceli/gorouter/proxy/utils"
	"github.com/mdimiceli/gorouter/registry"
	"github.com/mdimiceli/gorouter/routeservice"
)

var (
	headersToAlwaysRemove = []string{"X-CF-Proxy-Signature"}
)

type proxy struct {
	logger                logger.Logger
	errorWriter           errorwriter.ErrorWriter
	reporter              metrics.ProxyReporter
	accessLogger          accesslog.AccessLogger
	promRegistry          handlers.Registry
	health                *health.Health
	routeServiceConfig    *routeservice.RouteServiceConfig
	bufferPool            httputil.BufferPool
	backendTLSConfig      *tls.Config
	routeServiceTLSConfig *tls.Config
	config                *config.Config
}

func NewProxy(
	logger logger.Logger,
	accessLogger accesslog.AccessLogger,
	promRegistry handlers.Registry,
	errorWriter errorwriter.ErrorWriter,
	cfg *config.Config,
	registry registry.Registry,
	reporter metrics.ProxyReporter,
	routeServiceConfig *routeservice.RouteServiceConfig,
	backendTLSConfig *tls.Config,
	routeServiceTLSConfig *tls.Config,
	health *health.Health,
	routeServicesTransport http.RoundTripper,
) http.Handler {

	p := &proxy{
		accessLogger:          accessLogger,
		promRegistry:          promRegistry,
		logger:                logger,
		errorWriter:           errorWriter,
		reporter:              reporter,
		health:                health,
		routeServiceConfig:    routeServiceConfig,
		bufferPool:            NewBufferPool(),
		backendTLSConfig:      backendTLSConfig,
		routeServiceTLSConfig: routeServiceTLSConfig,
		config:                cfg,
	}

	dialer := &net.Dialer{
		Timeout:   cfg.EndpointDialTimeout,
		KeepAlive: cfg.EndpointKeepAliveProbeInterval,
	}

	roundTripperFactory := &round_tripper.FactoryImpl{
		BackendTemplate: &http.Transport{
			DialContext:           dialer.DialContext,
			DisableKeepAlives:     cfg.DisableKeepAlives,
			MaxIdleConns:          cfg.MaxIdleConns,
			IdleConnTimeout:       90 * time.Second, // setting the value to golang default transport
			MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
			DisableCompression:    true,
			TLSClientConfig:       backendTLSConfig,
			TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
		RouteServiceTemplate: &http.Transport{
			DialContext:           dialer.DialContext,
			DisableKeepAlives:     cfg.DisableKeepAlives,
			MaxIdleConns:          cfg.MaxIdleConns,
			IdleConnTimeout:       90 * time.Second, // setting the value to golang default transport
			MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
			DisableCompression:    true,
			TLSClientConfig:       routeServiceTLSConfig,
			ExpectContinueTimeout: 1 * time.Second,
		},
		IsInstrumented: cfg.SendHttpStartStopClientEvent,
	}

	prt := round_tripper.NewProxyRoundTripper(
		roundTripperFactory,
		fails.RetriableClassifiers,
		logger,
		reporter,
		&round_tripper.ErrorHandler{
			MetricReporter: reporter,
			ErrorSpecs:     round_tripper.DefaultErrorSpecs,
		},
		routeServicesTransport,
		cfg,
	)

	rproxy := &httputil.ReverseProxy{
		Director:       p.setupProxyRequest,
		Transport:      prt,
		FlushInterval:  50 * time.Millisecond,
		BufferPool:     p.bufferPool,
		ModifyResponse: p.modifyResponse,
	}

	routeServiceHandler := handlers.NewRouteService(routeServiceConfig, registry, logger, errorWriter)

	zipkinHandler := handlers.NewZipkin(cfg.Tracing.EnableZipkin, logger)
	w3cHandler := handlers.NewW3C(cfg.Tracing.EnableW3C, cfg.Tracing.W3CTenantID, logger)

	headersToLog := utils.CollectHeadersToLog(
		cfg.ExtraHeadersToLog,
		zipkinHandler.HeadersToLog(),
		w3cHandler.HeadersToLog(),
	)

	n := negroni.New()
	n.Use(handlers.NewPanicCheck(p.health, logger))
	n.Use(handlers.NewRequestInfo())
	n.Use(handlers.NewProxyWriter(logger))
	n.Use(zipkinHandler)
	n.Use(w3cHandler)
	n.Use(handlers.NewVcapRequestIdHeader(logger))
	if cfg.SendHttpStartStopServerEvent {
		n.Use(handlers.NewHTTPStartStop(dropsonde.DefaultEmitter, logger))
	}
	if p.promRegistry != nil {
		if cfg.PerAppPrometheusHttpMetricsReporting {
			n.Use(handlers.NewHTTPLatencyPrometheus(p.promRegistry))
		}
	}
	n.Use(handlers.NewAccessLog(accessLogger, headersToLog, cfg.Logging.EnableAttemptsDetails, logger))
	n.Use(handlers.NewQueryParam(logger))
	n.Use(handlers.NewReporter(reporter, logger))
	n.Use(handlers.NewHTTPRewriteHandler(cfg.HTTPRewrite, headersToAlwaysRemove))
	n.Use(handlers.NewProxyHealthcheck(cfg.HealthCheckUserAgent, p.health))
	n.Use(handlers.NewProtocolCheck(logger, errorWriter, cfg.EnableHTTP2))
	n.Use(handlers.NewLookup(registry, reporter, logger, errorWriter, cfg.EmptyPoolResponseCode503))
	n.Use(handlers.NewMaxRequestSize(cfg, logger))
	n.Use(handlers.NewClientCert(
		SkipSanitize(routeServiceHandler.(*handlers.RouteService)),
		ForceDeleteXFCCHeader(routeServiceHandler.(*handlers.RouteService), cfg.ForwardedClientCert, logger),
		cfg.ForwardedClientCert,
		logger,
		errorWriter,
	))
	n.Use(handlers.NewHopByHop(cfg, logger))
	n.Use(&handlers.XForwardedProto{
		SkipSanitization:         SkipSanitizeXFP(routeServiceHandler.(*handlers.RouteService)),
		ForceForwardedProtoHttps: p.config.ForceForwardedProtoHttps,
		SanitizeForwardedProto:   p.config.SanitizeForwardedProto,
	})
	n.Use(routeServiceHandler)
	n.Use(p)
	n.UseHandler(rproxy)

	return n
}

type RouteServiceValidator interface {
	ArrivedViaRouteService(req *http.Request, logger logger.Logger) (bool, error)
	IsRouteServiceTraffic(req *http.Request) bool
}

func SkipSanitizeXFP(routeServiceValidator RouteServiceValidator) func(*http.Request) bool {
	return func(req *http.Request) bool {
		return routeServiceValidator.IsRouteServiceTraffic(req)
	}
}

func SkipSanitize(routeServiceValidator RouteServiceValidator) func(*http.Request) bool {
	return func(req *http.Request) bool {
		return routeServiceValidator.IsRouteServiceTraffic(req) && (req.TLS != nil)
	}
}

func ForceDeleteXFCCHeader(routeServiceValidator RouteServiceValidator, forwardedClientCert string, logger logger.Logger) func(*http.Request) (bool, error) {
	return func(req *http.Request) (bool, error) {
		valid, err := routeServiceValidator.ArrivedViaRouteService(req, logger)
		if err != nil {
			return false, err
		}
		return valid && forwardedClientCert != config.SANITIZE_SET && forwardedClientCert != config.ALWAYS_FORWARD, nil
	}
}

func (p *proxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request, next http.HandlerFunc) {
	logger := handlers.LoggerWithTraceInfo(p.logger, request)
	proxyWriter := responseWriter.(utils.ProxyResponseWriter)

	if p.config.EnableHTTP1ConcurrentReadWrite && request.ProtoMajor == 1 {
		rc := http.NewResponseController(proxyWriter)

		err := rc.EnableFullDuplex()
		if err != nil {
			logger.Panic("enable-full-duplex-err", zap.Error(err))
		}
	}

	reqInfo, err := handlers.ContextRequestInfo(request)
	if err != nil {
		logger.Panic("request-info-err", zap.Error(err))
	}

	if reqInfo.RoutePool == nil {
		logger.Panic("request-info-err", zap.Error(errors.New("failed-to-access-RoutePool")))
	}

	reqInfo.AppRequestStartedAt = time.Now()
	next(responseWriter, request)
	reqInfo.AppRequestFinishedAt = time.Now()
}

func (p *proxy) setupProxyRequest(target *http.Request) {
	reqInfo, err := handlers.ContextRequestInfo(target)
	if err != nil {
		p.logger.Panic("request-info-err", zap.Error(err))
		return
	}
	reqInfo.BackendReqHeaders = target.Header

	target.URL.Scheme = "http"
	target.URL.Host = target.Host
	target.URL.ForceQuery = false
	target.URL.Opaque = target.RequestURI

	if strings.HasPrefix(target.RequestURI, "//") {
		path := escapePathAndPreserveSlashes(target.URL.Path)
		target.URL.Opaque = "//" + target.Host + path

		if len(target.URL.Query()) > 0 {
			target.URL.Opaque = target.URL.Opaque + "?" + target.URL.Query().Encode()
		}
	}
	target.URL.RawQuery = ""

	setRequestXRequestStart(target)
	target.Header.Del(router_http.CfAppInstance)
}

func setRequestXRequestStart(request *http.Request) {
	if _, ok := request.Header[http.CanonicalHeaderKey("X-Request-Start")]; !ok {
		request.Header.Set("X-Request-Start", strconv.FormatInt(time.Now().UnixNano()/1e6, 10))
	}
}

func escapePathAndPreserveSlashes(unescaped string) string {
	parts := strings.Split(unescaped, "/")
	escapedPath := ""
	for _, part := range parts {
		escapedPart := url.PathEscape(part)
		escapedPath = escapedPath + escapedPart + "/"
	}
	escapedPath = strings.TrimSuffix(escapedPath, "/")

	return escapedPath
}
