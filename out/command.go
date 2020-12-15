// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package out

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/drone/envsubst"
	resource "github.com/vmware-tanzu/observability-event-resource"
	"github.com/vmware-tanzu/observability-event-resource/wavefront"
)

var safeEnvVars = map[string]bool{
	"ATC_EXTERNAL_URL":    true,
	"BUILD_ID":            true,
	"BUILD_JOB_NAME":      true,
	"BUILD_PIPELINE_NAME": true,
	"BUILD_TEAM_NAME":     true,
}

func safeEnvSubst(envFunc func(string) string) func(string) string {
	return func(str string) string {
		if _, ok := safeEnvVars[str]; ok {
			return envFunc(str)
		}

		return "INVALID ENV VAR " + str
	}
}

// RunCommand will either create an ongoing event (if params.action == "start")
// or close an existing ongoing event (if params.action == "end")
func RunCommand(stdin io.Reader, baseDir string, hc *http.Client, envFunc func(string) string) (Response, error) {
	var (
		s         Request
		err       error
		eventJSON []byte
	)

	if err = json.NewDecoder(stdin).Decode(&s); err != nil {
		return Response{}, err
	}

	if err = s.Source.Validate(); err != nil {
		return Response{}, err
	}

	if err = s.Params.Validate(); err != nil {
		return Response{}, err
	}

	client := wavefront.NewAPIClient(s.Source, hc)

	annotations, err := buildAnnotationsMap(s.Params.Annotations, envFunc)
	if err != nil {
		return Response{}, err
	}

	name, err := interpolateString(s.Params.Name, envFunc)
	if err != nil {
		return Response{}, err
	}

	tags, err := expandTags(s.Params.Tags, envFunc)
	if err != nil {
		return Response{}, err
	}

	switch s.Params.Action {
	case CREATE:
		eventJSON, err = client.CreateInstantEvent(name, annotations, tags)
	case START:
		eventJSON, err = client.StartOngoingEvent(name, annotations, tags)
	case END:
		idFilePath := filepath.Join(baseDir, s.Params.Event, "id")
		idBytes, ferr := ioutil.ReadFile(idFilePath)
		if ferr != nil {
			return Response{}, fmt.Errorf("could not read event ID to close: %w", ferr)
		}
		id := strings.TrimSpace(string(idBytes))

		jsonFilePath := filepath.Join(baseDir, s.Params.Event, "event.json")
		jsonBytes, ferr := ioutil.ReadFile(jsonFilePath)
		if ferr != nil {
			return Response{}, fmt.Errorf("could not parse event json: %w", ferr)
		}

		// if there are no annotations here, we want to do nothing to them in the end event
		if s.Params.Annotations == nil {
			annotations = nil
		}

		eventJSON, err = client.EndOngoingEvent(id, jsonBytes, annotations)
	}
	if err != nil {
		return Response{}, fmt.Errorf("could not complete API call: %w", err)
	}

	var event interface{}
	if err = json.NewDecoder(bytes.NewBuffer(eventJSON)).Decode(&event); err != nil {
		return Response{}, fmt.Errorf("could not parse response: %w", err)
	}

	id, err := wavefront.GetEventID(event)
	if err != nil {
		return Response{}, fmt.Errorf("could not determine event ID from response: %w", err)
	}

	metadata, err := wavefront.GetConcourseMetadata(event)
	if err != nil {
		return Response{}, fmt.Errorf("could not determine event state from response: %w", err)
	}
	return Response{
		Version:  resource.Version{ID: id},
		Metadata: metadata,
	}, nil
}

func buildAnnotationsMap(custom map[string]string, envFunc func(string) string) (map[string]string, error) {
	annotations := make(map[string]string)

	annotations["concourse-team"] = envFunc("BUILD_TEAM_NAME")
	annotations["concourse-pipeline"] = envFunc("BUILD_PIPELINE_NAME")
	annotations["concourse-job"] = envFunc("BUILD_JOB_NAME")
	annotations["concourse-build-url"] = fmt.Sprintf("%s/builds/%s", envFunc("ATC_EXTERNAL_URL"), envFunc("BUILD_ID"))
	annotations["severity"] = "info"
	annotations["details"] = fmt.Sprintf("Created by Concourse observability-event-resource version %s", resource.AppVersion)

	var err error
	for k, v := range custom {
		if v == "" {
			delete(annotations, k)
			continue
		}

		if annotations[k], err = interpolateString(v, safeEnvSubst(envFunc)); err != nil {
			return nil, err
		}
	}

	return annotations, nil
}

func expandTags(tags []string, envFunc func(string) string) ([]string, error) {
	var err error

	newTags := make([]string, len(tags))
	for i, tag := range tags {
		if newTags[i], err = interpolateString(tag, safeEnvSubst(envFunc)); err != nil {
			return nil, err
		}
	}

	return newTags, nil
}

func interpolateString(s string, envFunc func(string) string) (string, error) {
	return envsubst.Eval(s, safeEnvSubst(envFunc))
}
