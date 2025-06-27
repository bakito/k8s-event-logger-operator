#!/bin/bash

# Set default values
DAYS=365
KEY_SIZE=2048
COUNTRY="US"
STATE="State"
LOCALITY="City"
ORGANIZATION="Organization"
ORGANIZATIONAL_UNIT="IT"
COMMON_NAME="k8s-event-logger-operator.k8s-event-logger-operator.svc"
OUTPUT_DIR="$(dirname "$0")/certificates"

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Generate private key
openssl genrsa -out "$OUTPUT_DIR/private.key" $KEY_SIZE

# Generate Certificate Signing Request (CSR)
openssl req -new \
    -key "$OUTPUT_DIR/private.key" \
    -out "$OUTPUT_DIR/request.csr" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORGANIZATIONAL_UNIT/CN=$COMMON_NAME"

# Generate self-signed certificate
openssl x509 -req \
    -days $DAYS \
    -in "$OUTPUT_DIR/request.csr" \
    -signkey "$OUTPUT_DIR/private.key" \
    -out "$OUTPUT_DIR/certificate.crt" \
    -extensions v3_req \
    -extfile <(printf "[v3_req]\nsubjectAltName=DNS:$COMMON_NAME")

# Set appropriate permissions
chmod 600 "$OUTPUT_DIR/private.key"
chmod 644 "$OUTPUT_DIR/certificate.crt"

echo "Self-signed certificate has been generated successfully!"
echo "Private key: $OUTPUT_DIR/private.key"
echo "Certificate: $OUTPUT_DIR/certificate.crt"
echo "CSR: $OUTPUT_DIR/request.csr"