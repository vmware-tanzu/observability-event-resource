// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/vmware-tanzu/observability-event-resource/wavefront"
)

type request struct {
	method        string
	token         string
	response      string
	requestString string
}

type fakeRoundTripper struct {
	allowedURLs map[string]*request
	urlCounts   map[string]int
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.ResponseRecorder{}

	if f.urlCounts == nil {
		f.urlCounts = map[string]int{}
	}

	if _, ok := f.urlCounts[req.URL.Path]; !ok {
		f.urlCounts[req.URL.Path] = 0
	}

	f.urlCounts[req.URL.Path]++

	r, ok := f.allowedURLs[req.URL.Path]
	if !ok {
		recorder.Code = http.StatusNotFound
		return recorder.Result(), nil
	}

	if r.method != req.Method {
		recorder.Code = http.StatusMethodNotAllowed
		return recorder.Result(), nil
	}

	token := req.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if r.token != token {
		recorder.Code = http.StatusUnauthorized
		return recorder.Result(), nil
	}

	if req.Body != nil {
		b := &bytes.Buffer{}
		io.Copy(b, req.Body)

		r.requestString = b.String()
	}

	recorder.Code = http.StatusOK
	recorder.Body = bytes.NewBufferString(r.response)
	return recorder.Result(), nil
}

func GetFakeHTTPClient(method, path, token, response string) *http.Client {
	f := &fakeRoundTripper{
		allowedURLs: map[string]*request{},
		urlCounts:   map[string]int{},
	}

	f.addSubRequest(method, path, token, "", response)

	hc := http.DefaultClient
	hc.Transport = f

	return hc
}

func GetSentRequest(hc *http.Client, url string) string {
	f := getRoundTripperFromClient(hc)

	r, ok := f.allowedURLs[url]
	if !ok {
		return ""
	}

	return r.requestString
}

func GetURLHitCount(hc *http.Client, url string) int {
	f := getRoundTripperFromClient(hc)
	if v, ok := f.urlCounts[url]; ok {
		return v
	}

	return 0
}

func AddSubRequest(hc *http.Client, method, path, token, response string) {
	f := getRoundTripperFromClient(hc)
	f.addSubRequest(method, path, token, "", response)
}

func (f *fakeRoundTripper) addSubRequest(method, path, token, requestString, response string) {
	f.allowedURLs[path] = &request{
		method:        method,
		token:         token,
		requestString: requestString,
		response:      response,
	}
}

func getRoundTripperFromClient(hc *http.Client) *fakeRoundTripper {
	var f *fakeRoundTripper

	switch hc.Transport.(type) {
	case *fakeRoundTripper:
		f = hc.Transport.(*fakeRoundTripper)
	case *wavefront.AuthRoundTripper:
		a := hc.Transport.(*wavefront.AuthRoundTripper)
		f = a.GetDelegate().(*fakeRoundTripper)
	}

	return f
}
