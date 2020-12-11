// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/vmware-tanzu/observability-event-resource/wavefront"
)

type fakeRoundTripper struct {
	method   string
	path     string
	token    string
	request  string
	response string
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.ResponseRecorder{}

	if f.method != req.Method {
		recorder.Code = http.StatusMethodNotAllowed
		return recorder.Result(), nil
	}

	if f.path != req.URL.Path {
		recorder.Code = http.StatusNotFound
		return recorder.Result(), nil
	}

	token := req.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if f.token != token {
		recorder.Code = http.StatusUnauthorized
		return recorder.Result(), nil
	}

	if req.Body != nil {
		b := &bytes.Buffer{}
		io.Copy(b, req.Body)

		f.request = b.String()
	}

	recorder.Code = http.StatusOK
	recorder.Body = bytes.NewBufferString(f.response)
	return recorder.Result(), nil
}

func GetFakeHTTPClient(method, path, token, response string) *http.Client {
	f := &fakeRoundTripper{
		method,
		path,
		token,
		"",
		response,
	}

	hc := http.DefaultClient
	hc.Transport = f

	return hc
}

func GetSentRequest(hc *http.Client) string {
	switch hc.Transport.(type) {
	case *fakeRoundTripper:
		f := hc.Transport.(*fakeRoundTripper)
		return f.request
	case *wavefront.AuthRoundTripper:
		a := hc.Transport.(*wavefront.AuthRoundTripper)
		delegate := a.GetDelegate()
		if f, ok := delegate.(*fakeRoundTripper); ok {
			return f.request
		}

		return ""
	default:
		return ""
	}
}
