FROM golang:1.13 as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx
COPY . .

ENV GOPROXY=https://goproxy.io \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
RUN go build -a -installsuffix cgo -ldflags="-w -s" -o k8s-event-logger cmd/logger/main.go && \
    upx --ultra-brute -q k8s-event-logger

# application image

FROM scratch

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080
USER 1001
ENTRYPOINT ["/go/bin/k8s-event-logger"]

COPY --from=builder /build/k8s-event-logger /go/bin/k8s-event-logger
