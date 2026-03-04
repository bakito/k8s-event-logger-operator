#!/bin/bash
set -e
docker build -f Dockerfile --build-arg VERSION=e2e-tests -t k8s-event-logger:e2e .
