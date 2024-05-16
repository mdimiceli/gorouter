package http_test

import (
	"net/http"

	commonhttp "github.com/mdimiceli/gorouter/common/http"
	httpfakes "github.com/mdimiceli/gorouter/common/http/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Headers", func() {
	Describe("SetTraceHeaders", func() {
		var respWriter http.ResponseWriter

		BeforeEach(func() {
			respWriter = httpfakes.NewFakeResponseWriter()
		})

		JustBeforeEach(func() {
			commonhttp.SetTraceHeaders(respWriter, "1.1.1.1", "example.com")
		})

		It("sets the trace headers on the response", func() {
			Expect(respWriter.Header().Get(commonhttp.VcapRouterHeader)).To(Equal("1.1.1.1"))
			Expect(respWriter.Header().Get(commonhttp.VcapBackendHeader)).To(Equal("example.com"))
			Expect(respWriter.Header().Get(commonhttp.CfRouteEndpointHeader)).To(Equal("example.com"))
		})
	})
})
