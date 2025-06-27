#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")

kubectl create ns k8s-event-logger-operator || true

kubectl apply -f  "${SCRIPT_DIR}/secret-webhook-cert.yaml"

helm upgrade --install k8s-event-logger-operator helm \
  --namespace k8s-event-logger-operator \
  -f "${SCRIPT_DIR}/e2e-values.yaml" \
  --atomic

