# Tanzu Observability Event Resource

A [concourse](https://concourse-ci.org) resource to send pipeline events to [VMware Tanzu Observability by Wavefront](https://tanzu.vmware.com/observability)

## Source Configuration

* `tenant_url`: The URL to your tenant, for example 
   `https://longboard.wavefront.com`
* `api_token`: A REST API token. More information on generating
   an API token [here](https://docs.wavefront.com/wavefront_api.html)

## Behavior

### `check`: No-op

Currently, this resource does not support monitoring for new events.

### `in`: Fetch information about an event

Fetches the given event, and creates the following files:
* `id`: contains the event's ID
* `event.json`: represents the event object as returned by the API

**Note**: Because this resource does not support monitoring, the `in` script is
really only used in a `get-after-put` context. It is used to pass the event 
between jobs in a pipeline so that it may be started in one job and ended in a
subsequent job.

### `out`: Start or end an event

Depending on the `action` parameter, `out` will either create a new event with
a running state of `ONGOING`, or close an event with the given ID.

#### Parameters

* `action`: *Required*. One of `create`, `start`, or `end`.
* `event`: *Required if action is `end`, ignored if action is `start` or `create`*. The path 
  to a previous event's `get` step, containing its `id` file.
* `event_name`: *Required if action is `start` or `create`, ignored if action is `end`*. The name of
  the event to be created
* `annotations`: *Optional, ignored if action is `end`*. A map of key-value pairs
  that will be added as annotations to the event. Values MUST be strings. In addition 
  to any annotations specified here, the following annotations will be added:
  ```
  concourse-team: ${BUILD_TEAM_NAME}
  concourse-pipeline: ${BUILD_PIPELINE_NAME}
  concourse-job: ${BUILD_JOB_NAME}
  concourse-build-url: ${ATC_EXTERNAL_URL}/builds/${BUILD_ID}
  ```
  Learn more about environment variables available to the resource type [here](https://concourse-ci.org/implementing-resource-types.html#resource-metadata)

  If you do not want one of those annotations on your event, add it as a custom 
  annotation with a value of `""`.

  Note that custom annotations currently do not support interpolating environment 
  variables.
* `tags`: *Optional, ignored if action is `end`*. A list of strings to be added as
  tags on the event.
   
**Note**: Both `annotations` and `tags` support very simple variable interpolation. For the list of
allowed variables, see [here](https://concourse-ci.org/implementing-resource-types.html#resource-metadata) 
and for a list of substitution patterns, see [here](https://github.com/drone/envsubst/blob/v1.0.2/README).

## Example

```yaml
resource_types:
- name: observability-events
  type: registry-image
  source:
    repository: projects.registry.vmware.com/tanzu/observability-event-resource
    tag: "1"

resources:
- name: observability
  type: observability-events
  source:
    tenant_url: https://longboard.wavefront.com
    api_token: ((my-secret-wavefront-token))

jobs:
- name: start-event
  plan:
  - put: observability
    params:
      action: start
      event_name: Pipeline Started
      tags: ["${BUILD_PIPELINE_NAME}"]
- name: do-a-thing
  plan:
  - get: observability
    passed: [start-event]
    trigger: true
  - task: do-some-stuff
    config: ...
- name: end-event
  plan:
  - get: observability
    passed: [do-a-thing]
    trigger: true
  - put: observability
    params:
      action: end
      event: observability
```