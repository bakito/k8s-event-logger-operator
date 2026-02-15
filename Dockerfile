FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk update && apk add upx

ARG VERSION=main
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux
COPY . .

RUN go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/k8s-event-logger-operator/version.Version=${VERSION}" -o k8s-event-logger && \
  upx -q k8s-event-logger

# application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080
USER 1001
ENTRYPOINT ["/opt/go/k8s-event-logger"]

COPY --from=builder /build/k8s-event-logger /opt/go/k8s-event-logger
