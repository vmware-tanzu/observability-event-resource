// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package wavefront

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	resource "github.com/vmware-tanzu/observability-event-resource"
)

type APIClient struct {
	client  *http.Client
	baseURL string
}

func NewAPIClient(source resource.Source, client *http.Client) *APIClient {
	if client == nil {
		client = http.DefaultClient
	}

	rt := client.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}

	newRT := &AuthRoundTripper{
		delegate: rt,
		token:    source.WavefrontToken,
	}

	client.Transport = newRT
	return &APIClient{
		client:  client,
		baseURL: strings.TrimSuffix(source.WavefrontURL, "/"),
	}
}

func (a *APIClient) newRequest(method string, uri string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, fmt.Sprintf("%s%s", a.baseURL, uri), body)
}

type AuthRoundTripper struct {
	delegate http.RoundTripper
	token    string
}

func (a *AuthRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	request.Header.Add("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	return a.delegate.RoundTrip(request)
}

func (a *AuthRoundTripper) GetDelegate() http.RoundTripper {
	return a.delegate
}

type singleItemResponse struct {
	Status   interface{} `json:"status"`
	Response interface{} `json:"response"`
}

type multiItemResponse struct {
	Status   interface{} `json:"status"`
	Response struct {
		Items []interface{} `json:"items"`
	} `json:"response"`
}

// ErrBadResponseStatus will be returned when a response code doesn't match the API specification
var ErrBadResponseStatus = errors.New("invalid response status code")
