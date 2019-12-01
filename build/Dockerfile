FROM golang:1.13 as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx
COPY . .

ENV GOPROXY=https://goproxy.io \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
RUN go test ./...
RUN go build -a -installsuffix cgo -ldflags="-w -s" -o k8s-event-logger-operator cmd/manager/main.go && \
    upx --ultra-brute -q k8s-event-logger-operator

# application image

FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080
USER 1001
ENTRYPOINT ["/opt/go/k8s-event-logger-operator"]

COPY --from=builder /build/k8s-event-logger-operator /opt/go/k8s-event-logger-operator