// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package in_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchellh/pointerstructure"
	resource "github.com/vmware-tanzu/observability-event-resource"
	"github.com/vmware-tanzu/observability-event-resource/in"
	"github.com/vmware-tanzu/observability-event-resource/internal"
)

func TestInvalidSource(t *testing.T) {
	stdin := strings.NewReader("{}")
	outputDirectory := "unused"

	_, err := in.RunCommand(stdin, outputDirectory, nil)
	if err == nil {
		t.Fatal("Expected an error to occur that never did")
	}

	if !errors.Is(err, resource.ErrMissingWavefrontURL) {
		t.Fatalf("Expected to get %v as an error but got %v", resource.ErrMissingWavefrontURL, err)
	}

	stdin = strings.NewReader(`{"source":{"tenant_url":"http://foo.com"}}`)
	_, err = in.RunCommand(stdin, outputDirectory, nil)
	if err == nil {
		t.Fatal("Expected an error to occur that never did")
	}

	if !errors.Is(err, resource.ErrMissingWavefrontToken) {
		t.Fatalf("Expected to get %v as an error but got %v", resource.ErrMissingWavefrontToken, err)
	}
}

func TestIn(t *testing.T) {
	stdin := strings.NewReader(`{"source": {"tenant_url": "https://foo", "api_token": "bar"}, "version": {"id": "1234"}}`)

	hc := internal.GetFakeHTTPClient(http.MethodGet, "/api/v2/event/1234", "bar", fakeOngoingEventJSON)

	tmpDir := t.TempDir()
	resp, err := in.RunCommand(stdin, tmpDir, hc)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if resp.Version.ID != "1234" {
		t.Fatalf("expected output version ID to be 1234, but it was %s", resp.Version.ID)
	}

	if len(resp.Metadata) != 2 {
		t.Fatalf("expected 2 metadata but found %d", len(resp.Metadata))
	}

	if resp.Metadata[0].Value != "some fake event" {
		t.Fatalf(`expected name to be "some fake event" but it was %q`, resp.Metadata[0].Value)
	}

	if resp.Metadata[1].Value != "ONGOING" {
		t.Fatalf("expected state to be ONGOING, but it was %s", resp.Metadata[1].Value)
	}

	id, err := ioutil.ReadFile(path.Join(tmpDir, "id"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if strings.TrimSpace(string(id)) != "1234" {
		t.Fatalf("expected id file to contain 1234, but it contained %q", strings.TrimSpace(string(id)))
	}

	jsonBytes, err := ioutil.ReadFile(path.Join(tmpDir, "event.json"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var (
		actualEvent   interface{}
		expectedEvent interface{}
	)

	if err = json.NewDecoder(bytes.NewBuffer(jsonBytes)).Decode(&actualEvent); err != nil {
		t.Fatal(err)
	}

	if err = json.NewDecoder(strings.NewReader(fakeOngoingEventJSON)).Decode(&expectedEvent); err != nil {
		t.Fatal(err)
	}

	respObj, err := pointerstructure.Get(expectedEvent, "/response")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(respObj, actualEvent) {
		t.Fatalf("unexpected value in event.json: %q", string(jsonBytes))
	}
}

const fakeOngoingEventJSON = `
{
	"status": {},
	"response": {
		"id": "1234",
		"name": "some fake event",
		"runningState": "ONGOING"
	}
}
`
