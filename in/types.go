// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package in

import resource "github.com/vmware-tanzu/observability-event-resource"

// Params is empty because an observability event get takes no parameters
type Params struct{}

// Request is what is received on stdin from the pipeline
type Request struct {
	Source  resource.Source  `json:"source"`
	Params  Params           `json:"params"`
	Version resource.Version `json:"version"`
}

// Response is a version and a set of metadata
type Response struct {
	Version  resource.Version  `json:"version"`
	Metadata resource.Metadata `json:"metadata"`
}
