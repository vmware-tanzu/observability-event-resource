// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package wavefront

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mitchellh/pointerstructure"
	resource "github.com/vmware-tanzu/observability-event-resource"
)

func (a *APIClient) GetEventJSON(eventID string) ([]byte, error) {
	uri := fmt.Sprintf("/api/v2/event/%s", url.PathEscape(eventID))

	req, err := a.newRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	return a.doEventRequest(req)
}

func (a *APIClient) CreateInstantEvent(name string, annotations map[string]string, tags []string) ([]byte, error) {
	start := time.Now().UnixNano() / int64(time.Millisecond)
	end := start + 1

	return a.createEvent(name, annotations, tags, start, end)
}

func (a *APIClient) StartOngoingEvent(name string, annotations map[string]string, tags []string) ([]byte, error) {
	return a.createEvent(name, annotations, tags, 0, 0)
}

func (a *APIClient) createEvent(name string, annotations map[string]string, tags []string, startTimeMillis int64, endTimeMillis int64) ([]byte, error) {
	requestBody := map[string]interface{}{
		"name":        name,
		"annotations": annotations,
		"tags":        tags,
	}

	if startTimeMillis > 0 {
		requestBody["startTime"] = startTimeMillis
	}

	if endTimeMillis > 0 {
		requestBody["endTime"] = endTimeMillis
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := a.newRequest(http.MethodPost, "/api/v2/event", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	return a.doEventRequest(req)
}

func (a *APIClient) EndOngoingEvent(eventID string) ([]byte, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("/api/v2/event/%s/close", url.PathEscape(eventID)), nil)
	if err != nil {
		return nil, err
	}

	return a.doEventRequest(req)
}

func (a *APIClient) doEventRequest(req *http.Request) ([]byte, error) {
	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: expected 200, got %d", ErrBadResponseStatus, response.StatusCode)
	}

	var resp singleItemResponse
	if err = json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return json.Marshal(resp.Response)
}

// GetEventID simply returns the ID of the event as specified in an event JSON block
func GetEventID(event interface{}) (string, error) {
	return getStr(event, "/id")
}

// GetConcourseMetadata will return the following key-value pairs:
//
//		key: name, value: <event name>
//		key: state, value: <event state>
func GetConcourseMetadata(event interface{}) (resource.Metadata, error) {
	name, err := getStr(event, "/name")
	if err != nil {
		return nil, err
	}

	state, err := getStr(event, "/runningState")
	if err != nil {
		return nil, err
	}

	return resource.Metadata{
		resource.Metadatum{
			Name:  "name",
			Value: name,
		},
		resource.Metadatum{
			Name:  "state",
			Value: state,
		},
	}, nil
}

func getStr(event interface{}, query string) (string, error) {
	obj, err := pointerstructure.Get(event, query)
	if err != nil {
		return "", err
	}

	str, ok := obj.(string)
	if !ok {
		return "", fmt.Errorf("expected %s to be a string, but it was %T", query, obj)
	}

	return str, nil
}