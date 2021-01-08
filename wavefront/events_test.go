package wavefront_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	resource "github.com/vmware-tanzu/observability-event-resource"
	"github.com/vmware-tanzu/observability-event-resource/wavefront"
)

func TestRetryLogic(t *testing.T) {
	handler := &testServerHandler{
		retryCount: 2,
	}

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	source := resource.Source{
		WavefrontURL:   server.URL,
		WavefrontToken: "retry",
	}

	hc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := wavefront.NewAPIClient(source, hc)
	_, err := client.StartOngoingEvent("My event", map[string]string{"foo": "bar"}, []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("unexpected error occured: %v", err)
	}

	if handler.retryCount != 0 {
		t.Fatalf("expected the client to retry twice, but retried %d times", 2-handler.retryCount)
	}
}

type testServerHandler struct {
	retryCount int
}

func (s *testServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var token string
	authHeader := r.Header.Get("authorization")

	fmt.Sscanf(strings.ToLower(authHeader), "bearer %s", &token)
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if s.retryCount > 0 {
		s.retryCount--
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, httpOKResponse)
}

const (
	httpOKResponse = `
	{
		"status": {},
		"response": {
			"id": "12345",
			"name": "My event",
			"runningState": "ONGOING",
			"annotations": {
				"foo": "bar",
				"concourse-job": "",
				"concourse-team": "",
				"concourse-pipeline": "test-pipeline",
				"concourse-build-url": "/builds/",
				"severity": "info",
				"details": "Created by concourse observability-event-resource version 0.0.0-dev"
			},
			"tags": ["tag1", "tag2"]
		}
	}
	`
)
