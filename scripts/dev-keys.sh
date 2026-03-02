#!/usr/bin/env bash
set -euo pipefail
if [ -f keys/jwtRS256.key ] && [ -f keys/jwtRS256.key.pub ]; then
  echo "keys already exist, skipping generation"
  exit 0
fi
mkdir -p keys
openssl genrsa -out keys/jwtRS256.key 2048 2>/dev/null || { echo "ERROR: failed to generate private key"; exit 1; }
openssl rsa -in keys/jwtRS256.key -pubout -out keys/jwtRS256.key.pub 2>/dev/null || { echo "ERROR: failed to generate public key"; exit 1; }
echo "keys generated"
