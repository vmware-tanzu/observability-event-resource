FROM golang:1 AS builder

ARG VERSION="0.0.0-dev"

ENV CGO_ENABLED 0

COPY . /builddir
WORKDIR  /builddir

RUN go mod download
RUN go build -trimpath -ldflags "-X github.com/vmware-tanzu/observability-event-resource.AppVersion=${VERSION}" -o /assets/out ./cmd/out
RUN go build -trimpath -ldflags "-X github.com/vmware-tanzu/observability-event-resource.AppVersion=${VERSION}" -o /assets/in ./cmd/in
RUN go build -trimpath -ldflags "-X github.com/vmware-tanzu/observability-event-resource.AppVersion=${VERSION}" -o /assets/check ./cmd/check
RUN go build -trimpath -ldflags "-X github.com/vmware-tanzu/observability-event-resource.AppVersion=${VERSION}" -o /assets/version ./cmd/version

RUN set -e; for pkg in $(go list ./...); do \
		go test -o "/tests/$(basename $pkg).test" -c $pkg; \
	done

FROM ubuntu:bionic AS resource
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates 

FROM resource AS tests
COPY --from=builder /tests /tests
RUN set -e; for test in /tests/*.test; do \
		$test; \
	done

FROM scratch
COPY --from=builder /assets /opt/resource
COPY --from=resource /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/