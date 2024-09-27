FROM golang:1.23-bullseye as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx

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
