#!/bin/sh -e

set -e

if [[ $# -ne 1 ]] ; then
    echo 'please use version as argument'
    exit 1
fi

RELEASE=v${1}
git pull
git checkout tags/${RELEASE} -b ${RELEASE}

podman rmi -f golang:1.13 || true

podman build -t quay.io/bakito/k8s-event-logger:${RELEASE} --no-cache  -f ./build/Dockerfile .
podman push quay.io/bakito/k8s-event-logger:${RELEASE}

git checkout master
git branch -d ${RELEASE} -f