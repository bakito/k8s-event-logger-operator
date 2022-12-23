#!/bin/bash
set -e

helm upgrade --install k8s-event-logger-operator helm \
  --namespace k8s-event-logger-operator \
  --create-namespace \
  -f testdata/e2e/e2e-values.yaml \
  --atomic

