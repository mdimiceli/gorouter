package router_test

import (
	"github.com/mdimiceli/gorouter/registry"
	"github.com/mdimiceli/gorouter/test/common"
)

func appRegistered(registry *registry.RouteRegistry, app *common.TestApp) bool {
	for _, url := range app.Urls() {
		pool := registry.Lookup(url)
		if pool == nil {
			return false
		}
	}

	return true
}
