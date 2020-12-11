// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"errors"
	"fmt"
)

// AppVersion will be specified by the build
var AppVersion = "0.0.0-dev"

// Source is what is configured for the concourse resource
//
// 		resources:
//		- name: events
// 		  type: wavefront
//		  source:
//			tenant_url: http://<mywavefronttenant>.wavefront.com
//			api_token: ((my-secret-token))
type Source struct {
	WavefrontURL   string `json:"tenant_url"`
	WavefrontToken string `json:"api_token"`
}

// Validate ensures that the source's required properties are set
func (s Source) Validate() error {
	if s.WavefrontURL == "" {
		return fmt.Errorf("could not validate source configuration: %w", ErrMissingWavefrontURL)
	}

	if s.WavefrontToken == "" {
		return fmt.Errorf("could not validate source configuration: %w", ErrMissingWavefrontToken)
	}

	return nil
}

// Version is used by the in and out script and represents an event's ID
type Version struct {
	ID string `json:"id"`
}

// Metadatum is a key value pair
type Metadatum struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Metadata is a slice of Metadatum
type Metadata []Metadatum

// ERRORS

// ErrMissingWavefrontURL will be emitted or wrapped when the source is missing the wavefront URL
var ErrMissingWavefrontURL = errors.New("wavefront url is missing")

// ErrMissingWavefrontToken will be emitted or wrapped when the source is missing the wavefront token
var ErrMissingWavefrontToken = errors.New("wavefront token is missing")
