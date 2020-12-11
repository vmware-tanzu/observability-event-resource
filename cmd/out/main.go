// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	resource "github.com/vmware-tanzu/observability-event-resource"
	"github.com/vmware-tanzu/observability-event-resource/out"
)

func main() {
	baseDirectory := os.Args[1]

	fmt.Fprintln(os.Stderr, resource.AppVersion)

	resp, err := out.RunCommand(os.Stdin, baseDirectory, http.DefaultClient, os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		log.Fatal(err)
	}
}
