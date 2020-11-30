FROM quay.io/bitnami/golang:1.15 as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx

ENV GOPROXY=https://goproxy.io \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
COPY . .

RUN make test
RUN ./hack/build.sh k8s-event-logger .

# application image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest 
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080
USER 1001
ENTRYPOINT ["/opt/go/k8s-event-logger"]

COPY --from=builder /build/k8s-event-logger /opt/go/k8s-event-logger
