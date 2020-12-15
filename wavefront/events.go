// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package wavefront

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
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

func (a *APIClient) EndOngoingEvent(eventID string, eventJSON []byte, newAnnotations map[string]string) ([]byte, error) {
	if eventJSON != nil && newAnnotations != nil {
		var event interface{}

		err := json.NewDecoder(bytes.NewBuffer(eventJSON)).Decode(&event)
		if err != nil {
			return nil, fmt.Errorf("could not parse event json: %w", err)
		}

		newEvent, err := mergeAnnotationsIntoEvent(event, newAnnotations)
		if err != nil {
			return nil, fmt.Errorf("error merging new annotations with existing annotations: %w", err)
		}

		if newEvent != nil { // newEvent will be nil if no changes were made, so if it's not nil, we need to update
			if err = a.updateExistingEvent(eventID, newEvent); err != nil {
				return nil, fmt.Errorf("could not update event: %w", err)
			}
		}
	}

	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("/api/v2/event/%s/close", url.PathEscape(eventID)), nil)
	if err != nil {
		return nil, err
	}

	return a.doEventRequest(req)
}

func (a *APIClient) updateExistingEvent(eventID string, event interface{}) error {
	bodyBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("could not serialize to json: %w", err)
	}

	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("/api/v2/event/%s", url.PathEscape(eventID)), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("error generating HTTP request: %w", err)
	}

	_, err = a.doEventRequest(req)
	return err
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

func mergeAnnotationsIntoEvent(event interface{}, newAnnotations map[string]string) (interface{}, error) {
	var (
		existingAnnotations interface{}
		err                 error
	)

	if existingAnnotations, err = pointerstructure.Get(event, "/response/annotations"); err != nil {
		return nil, fmt.Errorf("could not retrieve required annotations field: %w", err)
	}

	finalMap := map[string]interface{}{}
	existingAnnotationMap, ok := existingAnnotations.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected existing annotations to be map[string]interface{} but it was %T", existingAnnotations)
	}

	for k, v := range existingAnnotationMap {
		finalMap[k] = v
	}

	for k, v := range newAnnotations {
		if v == "" {
			if _, ok = finalMap[k]; ok {
				str, ok := finalMap[k].(string)
				if !ok {
					return nil, fmt.Errorf("expected value at %s to be a string, but it was %T", k, v)
				}
				if str != "" {
					// if the existing annotation is not "" and the new one is, delete the key from finalMap
					delete(finalMap, k)
					continue
				}
			}

			continue
		}

		finalMap[k] = v
	}

	if !reflect.DeepEqual(existingAnnotations, finalMap) {
		if event, err = pointerstructure.Set(event, "/response/annotations", newAnnotations); err != nil {
			return nil, fmt.Errorf("could not modify annotations: %w", err)
		}

		newEvent, err := pointerstructure.Get(event, "/response")
		if err != nil {
			return nil, fmt.Errorf("could not get response JSON: %w", err)
		}

		return newEvent, nil
	}

	return nil, nil
}
