package test_helpers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mdimiceli/gorouter/metrics"
	"github.com/mdimiceli/gorouter/route"
	"github.com/mdimiceli/gorouter/stats"
)

type NullVarz struct{}

func (NullVarz) MarshalJSON() ([]byte, error)            { return json.Marshal(nil) }
func (NullVarz) ActiveApps() *stats.ActiveApps           { return stats.NewActiveApps() }
func (NullVarz) CaptureBadRequest()                      {}
func (NullVarz) CaptureBadGateway()                      {}
func (NullVarz) CaptureRoutingRequest(b *route.Endpoint) {}
func (NullVarz) CaptureRoutingResponse(int)              {}
func (NullVarz) CaptureRoutingResponseLatency(*route.Endpoint, int, time.Time, time.Duration) {
}
func (NullVarz) CaptureRouteServiceResponse(*http.Response)         {}
func (NullVarz) CaptureRegistryMessage(msg metrics.ComponentTagged) {}
