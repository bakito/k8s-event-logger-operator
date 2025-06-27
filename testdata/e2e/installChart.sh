#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
WEBHOOK_CERT_NAME=k8s-event-logger-operator-webhook
NAMESPACE=k8s-event-logger-operator

${SCRIPT_DIR}/create-cert.sh

kubectl create ns "${NAMESPACE}" || true

kubectl create secret tls "${WEBHOOK_CERT_NAME}" -n "${NAMESPACE}" \
  --cert="${SCRIPT_DIR}/certificates/certificate.crt" \
  --key="${SCRIPT_DIR}/certificates/private.key"

WEBHOOK_CERT=$(cat "${SCRIPT_DIR}/certificates/certificate.crt" | base64 | tr -d '\n')

helm upgrade --install k8s-event-logger-operator helm \
  --namespace "${NAMESPACE}" \
  -f "${SCRIPT_DIR}/e2e-values.yaml" \
  --set "webhook.caBundle=${WEBHOOK_CERT}" \
  --set "webhook.certsSecret.name=${WEBHOOK_CERT_NAME}" \
  --atomic

