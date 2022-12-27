#!/bin/bash
set -e
docker build -f Dockerfile --build-arg VERSION=e2e-tests -t localhost:5001/k8s-event-logger:e2e .
docker push localhost:5001/k8s-event-logger:e2e
