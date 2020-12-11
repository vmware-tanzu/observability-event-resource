// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package out

import (
	"errors"
	"fmt"

	resource "github.com/vmware-tanzu/observability-event-resource"
)

// Params indicates what should be done
type Params struct {
	Action      EventAction       `json:"action"`
	Name        string            `json:"event_name"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Tags        []string          `json:"tags"`
	Event       string            `json:"event"`
}

// Validate will ensure that all required properties are set in a put's "params" block
func (p Params) Validate() error {
	if p.Action != START &&
		p.Action != END &&
		p.Action != CREATE {
		return fmt.Errorf("invalid action %s", p.Action)
	}

	if p.Action == END && p.Event == "" {
		return errors.New(`the "event" parameter must be set when "action" is "end"`)
	}

	if (p.Action == START || p.Action == CREATE) && p.Name == "" {
		return errors.New(`the "event_name" parameter must be set when "action" is "start" or "create"`)
	}

	return nil
}

// Request is what is received on stdin from the pipeline
type Request struct {
	Source resource.Source `json:"source"`
	Params Params          `json:"params"`
}

// Response is a version and a set of metadata
type Response struct {
	Version  resource.Version  `json:"version"`
	Metadata resource.Metadata `json:"metadata"`
}

// EventAction represents an action the resource can take
type EventAction string

const (
	// START will create an event with running state ONGOING
	START EventAction = "start"

	// END will close an ONGOING event
	END EventAction = "end"

	// CREATE will create an instantaneous event
	CREATE EventAction = "create"
)
