// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package in

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"

	resource "github.com/vmware-tanzu/observability-event-resource"
	"github.com/vmware-tanzu/observability-event-resource/wavefront"
)

// RunCommand will attempt to read the requested event, and will
// output the following files to the specified outputDirectory:
// * id - contains the event ID
// * event.json - contains the whole of the event JSON
func RunCommand(stdin io.Reader, outputDirectory string, hc *http.Client) (Response, error) {
	var s Request

	if err := json.NewDecoder(stdin).Decode(&s); err != nil {
		return Response{}, err
	}

	if err := s.Source.Validate(); err != nil {
		return Response{}, err
	}

	client := wavefront.NewAPIClient(s.Source, hc)

	eventJSON, err := client.GetEventJSON(s.Version.ID)
	if err != nil {
		return Response{}, fmt.Errorf("error getting event data: %w", err)
	}

	if err = ioutil.WriteFile(filepath.Join(outputDirectory, "id"), []byte(s.Version.ID), 0644); err != nil {
		return Response{}, fmt.Errorf("error writing event id: %w", err)
	}

	if err = ioutil.WriteFile(filepath.Join(outputDirectory, "event.json"), eventJSON, 0644); err != nil {
		return Response{}, fmt.Errorf("error writing event data: %w", err)
	}

	var event interface{}
	if err = json.NewDecoder(bytes.NewBuffer(eventJSON)).Decode(&event); err != nil {
		return Response{}, fmt.Errorf("error parsing event from response: %w", err)
	}

	metadata, err := wavefront.GetConcourseMetadata(event)
	if err != nil {
		return Response{}, fmt.Errorf("error calculating resource metadata: %w", err)
	}

	return Response{
		Version:  resource.Version{ID: s.Version.ID},
		Metadata: metadata,
	}, nil
}
