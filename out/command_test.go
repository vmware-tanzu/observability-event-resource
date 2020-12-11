// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package out_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/vmware-tanzu/observability-event-resource/internal"
	"github.com/vmware-tanzu/observability-event-resource/out"
)

var envMap = map[string]string{
	"BUILD_JOB_NAME":      "test-job",
	"BUILD_PIPELINE_NAME": "test-pipeline",
}

func TestParamValidation(t *testing.T) {
	p := out.Params{
		Action: "foo",
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("an expected error did not occur")
	}

	p.Action = out.END
	err = p.Validate()
	if err == nil {
		t.Fatal("an expected error did not occur")
	}

	p.Event = "some-event"
	err = p.Validate()
	if err != nil {
		t.Fatalf("an unexpected error occured: %v", err)
	}

	p = out.Params{
		Action: out.START,
	}

	err = p.Validate()
	if err == nil {
		t.Fatal("an expected error did not occur")
	}

	p.Name = "foo"
	err = p.Validate()
	if err != nil {
		t.Fatalf("an unexpected error occured: %v", err)
	}
}

func TestStartEvent(t *testing.T) {
	stdin := strings.NewReader(startEventRequest)

	hc := internal.GetFakeHTTPClient(http.MethodPost, "/api/v2/event", "asdf", startEventResponse)

	resp, err := out.RunCommand(stdin, "", hc, os.Getenv)
	if err != nil {
		t.Fatalf("an unexpected error occured: %v", err)
	}

	if resp.Version.ID != "12345" {
		t.Fatalf("expected output version ID to be 12345, but it was %s", resp.Version.ID)
	}

	if len(resp.Metadata) != 2 {
		t.Fatalf("expected 2 metadata but found %d", len(resp.Metadata))
	}

	if resp.Metadata[0].Value != "My event" {
		t.Fatalf(`expected name to be "My event" but it was %q`, resp.Metadata[0].Value)
	}

	if resp.Metadata[1].Value != "ONGOING" {
		t.Fatalf("expected state to be ONGOING, but it was %s", resp.Metadata[1].Value)
	}
}

func TestEndEvent(t *testing.T) {
	stdin := strings.NewReader(endEventRequest)

	baseDir := t.TempDir()
	if err := os.MkdirAll(path.Join(baseDir, "some-event"), 0777); err != nil {
		t.Fatalf("an unexpected error occured: %v", err)
	}

	if err := ioutil.WriteFile(path.Join(baseDir, "some-event", "id"), []byte("12345"), 0666); err != nil {
		t.Fatalf("an unexpected error occured: %v", err)
	}

	hc := internal.GetFakeHTTPClient(http.MethodPost, "/api/v2/event/12345/close", "asdf", endEventResponse)

	resp, err := out.RunCommand(stdin, baseDir, hc, os.Getenv)
	if err != nil {
		t.Fatalf("an unexpected error occurred: %v", err)
	}

	if resp.Version.ID != "12345" {
		t.Fatalf("expected output version ID to be 12345, but it was %s", resp.Version.ID)
	}

	if len(resp.Metadata) != 2 {
		t.Fatalf("expected 2 metadata but found %d", len(resp.Metadata))
	}

	if resp.Metadata[0].Value != "My event" {
		t.Fatalf(`expected name to be "My event" but it was %q`, resp.Metadata[0].Value)
	}

	if resp.Metadata[1].Value != "ENDED" {
		t.Fatalf("expected state to be ENDED, but it was %s", resp.Metadata[1].Value)
	}
}

func TestVariablizedEvent(t *testing.T) {
	stdin := strings.NewReader(variablizedEventRequest)

	hc := internal.GetFakeHTTPClient(http.MethodPost, "/api/v2/event", "asdf", variablizedEventResponse)

	resp, err := out.RunCommand(stdin, "", hc, envFunc)
	if err != nil {
		t.Fatalf("an unexpected error occured: %v", err)
	}

	if resp.Version.ID != "12345" {
		t.Fatalf("expected output version ID to be 12345, but it was %s", resp.Version.ID)
	}

	if len(resp.Metadata) != 2 {
		t.Fatalf("expected 2 metadata but found %d", len(resp.Metadata))
	}

	if resp.Metadata[0].Value != "My event" {
		t.Fatalf(`expected name to be "My event" but it was %q`, resp.Metadata[0].Value)
	}

	if resp.Metadata[1].Value != "ONGOING" {
		t.Fatalf("expected state to be ONGOING, but it was %s", resp.Metadata[1].Value)
	}

	requestBody := internal.GetSentRequest(hc)
	if !strings.Contains(requestBody, "test-job") {
		t.Fatal("expected the ${BUILD_JOB_NAME} parameter to be resolved, but it wasn't")
	}

	if !strings.Contains(requestBody, "test-pipeline") {
		t.Fatal("expected the ${BUILD_PIPELINE_NAME} parameter to be resolved, but it wasn't")
	}
}

func envFunc(str string) string {
	if s, ok := envMap[str]; ok {
		return s
	}

	return ""
}

const (
	startEventRequest = `
	{
		"source": {
			"tenant_url": "https://foo.com",
			"api_token": "asdf"
		},
		"params": {
			"action": "start",
			"event_name": "My event",
			"annotations": {
				"foo": "bar",
				"concourse-job": ""
			},
			"tags": ["tag1", "tag2"]
		}
	}
	`

	startEventResponse = `
	{
		"status": {},
		"response": {
			"id": "12345",
			"name": "My event",
			"runningState": "ONGOING",
			"annotations": {
				"foo": "bar",
				"concourse-job": ""
			},
			"tags": ["tag1", "tag2"]
		}
	}
	`

	endEventRequest = `
	{
		"source": {
			"tenant_url": "https://foo.com",
			"api_token": "asdf"
		},
		"params": {
			"action": "end",
			"event": "some-event"
		}
	}
	`

	endEventResponse = `
	{
		"status": {},
		"response": {
			"id": "12345",
			"name": "My event",
			"runningState": "ENDED",
			"annotations": {
				"foo": "bar",
				"concourse-job": ""
			},
			"tags": ["tag1", "tag2"]
		}
	}
	`

	variablizedEventRequest = `
	{
		"source": {
			"tenant_url": "https://foo.com",
			"api_token": "asdf"
		},
		"params": {
			"action": "start",
			"event_name": "My event",
			"annotations": {
				"foo": "bar",
				"concourse-job": "${BUILD_JOB_NAME} starting"
			},
			"tags": ["tag1", "tag2", "${BUILD_PIPELINE_NAME}"]
		}
	}`

	variablizedEventResponse = `
	{
		"status": {},
		"response": {
			"id": "12345",
			"name": "My event",
			"runningState": "ONGOING",
			"annotations": {
				"foo": "bar",
				"concourse-job": "test-job starting"
			},
			"tags": ["tag1", "tag2", "test-pipeline"]
		}
	}
	`
)
