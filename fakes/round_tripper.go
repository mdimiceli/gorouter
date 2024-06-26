// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"net/http"
	"sync"
)

type RoundTripper struct {
	RoundTripStub        func(*http.Request) (*http.Response, error)
	roundTripMutex       sync.RWMutex
	roundTripArgsForCall []struct {
		arg1 *http.Request
	}
	roundTripReturns struct {
		result1 *http.Response
		result2 error
	}
	roundTripReturnsOnCall map[int]struct {
		result1 *http.Response
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *RoundTripper) RoundTrip(arg1 *http.Request) (*http.Response, error) {
	fake.roundTripMutex.Lock()
	ret, specificReturn := fake.roundTripReturnsOnCall[len(fake.roundTripArgsForCall)]
	fake.roundTripArgsForCall = append(fake.roundTripArgsForCall, struct {
		arg1 *http.Request
	}{arg1})
	stub := fake.RoundTripStub
	fakeReturns := fake.roundTripReturns
	fake.recordInvocation("RoundTrip", []interface{}{arg1})
	fake.roundTripMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *RoundTripper) RoundTripCallCount() int {
	fake.roundTripMutex.RLock()
	defer fake.roundTripMutex.RUnlock()
	return len(fake.roundTripArgsForCall)
}

func (fake *RoundTripper) RoundTripCalls(stub func(*http.Request) (*http.Response, error)) {
	fake.roundTripMutex.Lock()
	defer fake.roundTripMutex.Unlock()
	fake.RoundTripStub = stub
}

func (fake *RoundTripper) RoundTripArgsForCall(i int) *http.Request {
	fake.roundTripMutex.RLock()
	defer fake.roundTripMutex.RUnlock()
	argsForCall := fake.roundTripArgsForCall[i]
	return argsForCall.arg1
}

func (fake *RoundTripper) RoundTripReturns(result1 *http.Response, result2 error) {
	fake.roundTripMutex.Lock()
	defer fake.roundTripMutex.Unlock()
	fake.RoundTripStub = nil
	fake.roundTripReturns = struct {
		result1 *http.Response
		result2 error
	}{result1, result2}
}

func (fake *RoundTripper) RoundTripReturnsOnCall(i int, result1 *http.Response, result2 error) {
	fake.roundTripMutex.Lock()
	defer fake.roundTripMutex.Unlock()
	fake.RoundTripStub = nil
	if fake.roundTripReturnsOnCall == nil {
		fake.roundTripReturnsOnCall = make(map[int]struct {
			result1 *http.Response
			result2 error
		})
	}
	fake.roundTripReturnsOnCall[i] = struct {
		result1 *http.Response
		result2 error
	}{result1, result2}
}

func (fake *RoundTripper) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.roundTripMutex.RLock()
	defer fake.roundTripMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *RoundTripper) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ http.RoundTripper = new(RoundTripper)
