#!/usr/bin/env bash
set -euo pipefail
mkdir -p keys

# JWT RS256 keypair
if [ -f keys/jwtRS256.key ] && [ -f keys/jwtRS256.key.pub ]; then
  echo "JWT keys already exist, skipping"
else
  openssl genrsa -out keys/jwtRS256.key 2048 2>/dev/null || { echo "ERROR: failed to generate JWT private key"; exit 1; }
  openssl rsa -in keys/jwtRS256.key -pubout -out keys/jwtRS256.key.pub 2>/dev/null || { echo "ERROR: failed to generate JWT public key"; exit 1; }
  echo "JWT keys generated"
fi

# TLS certificate (self-signed, for nginx reverse proxy)
if [ -f keys/sentinel.crt ] && [ -f keys/sentinel.key ]; then
  echo "TLS cert already exists, skipping"
else
  openssl req -x509 -nodes -newkey rsa:2048 \
    -keyout keys/sentinel.key \
    -out keys/sentinel.crt \
    -days 3650 \
    -subj "/CN=sentinel.local/O=Sentinel/C=CA" \
    -addext "subjectAltName=DNS:localhost,DNS:sentinel.local,IP:127.0.0.1" \
    2>/dev/null || { echo "ERROR: failed to generate TLS cert"; exit 1; }
  chmod 600 keys/sentinel.key
  echo "TLS cert generated (self-signed, valid 10 years)"
fi
