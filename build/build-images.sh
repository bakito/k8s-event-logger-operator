#!/bin/sh -e

set -e

if [[ $# -ne 1 ]] ; then
    echo 'please use version as argument'
    exit 1
fi

RELEASE=v${1}

git checkout tags/${RELEASE} -b ${RELEASE}

podman rmi -f golang:1.13

podman build -t quay.io/bakito/k8s-event-logger-operator:${RELEASE} --no-cache  -f ./build/Dockerfile .
podman push quay.io/bakito/k8s-event-logger-operator:${RELEASE}

podman build -t quay.io/bakito/k8s-event-logger:${RELEASE} --no-cache  -f ./build/logger.Dockerfile .
podman push quay.io/bakito/k8s-event-logger:${RELEASE}

git checkout master
git branch -d ${RELEASE} -f