// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	resource "github.com/vmware-tanzu/observability-event-resource"
)

func main() {
	fmt.Println(resource.AppVersion)
}
