# syntax=docker/dockerfile:1@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89
FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

WORKDIR /build

# Install build dependencies
RUN apk update && apk add --no-cache upx git

ARG VERSION=main
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GO111MODULE=on \
    CGO_ENABLED=0

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
      -a \
      -installsuffix cgo \
      -ldflags="-w -s -X github.com/bakito/k8s-event-logger-operator/version.Version=${VERSION}" \
      -o k8s-event-logger . && \
    upx -q k8s-event-logger

# application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080
USER 1001
ENTRYPOINT ["/opt/go/k8s-event-logger"]

COPY --from=builder /build/k8s-event-logger /opt/go/k8s-event-logger
