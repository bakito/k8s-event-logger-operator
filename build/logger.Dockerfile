FROM golang:1.13 as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx
COPY . .

ENV GOPROXY=https://goproxy.io \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
RUN go build -a -installsuffix cgo -ldflags="-w -s" -o event-logger-operator cmd/logger/main.go && \
    upx --ultra-brute -q event-logger-operator

# application image

FROM scratch

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080
USER 1001
ENTRYPOINT ["/go/bin/event-logger-operator"]

COPY --from=builder /build/event-logger-operator /go/bin/event-logger-operator