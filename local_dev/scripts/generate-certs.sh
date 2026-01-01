#!/bin/bash
# Generate TLS certificates for local development
# Run this once: ./generate-certs.sh

set -euo pipefail

CERT_DIR="$(dirname "$0")/../certs"
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Clean old certs
rm -f *.pem *.crt *.key *.csr *.srl

echo "==> Generating CA..."
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
    -subj "/CN=pg-inventory-local-ca"

echo "==> Generating server certificate..."
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr \
    -subj "/CN=postgres"

# Server cert needs SANs for all hostnames
cat > server-ext.cnf <<EOF
subjectAltName = DNS:localhost,DNS:pg-primary,DNS:pg-replica1,DNS:pg-replica2,IP:127.0.0.1
EOF

openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days 365 -extfile server-ext.cnf

echo "==> Generating client certificate (for extractor)..."
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr \
    -subj "/CN=pgmonitor"

openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -days 365

# PostgreSQL requires specific permissions
chmod 600 server.key client.key
chmod 644 ca.crt server.crt client.crt

# Cleanup CSRs
rm -f *.csr *.cnf *.srl

echo "==> Certificates generated in $(pwd)"
echo ""
echo "Files:"
ls -la
echo ""
echo "To connect with psql using client cert:"
echo "  psql \"host=localhost port=5433 dbname=postgres user=pgmonitor sslmode=verify-full sslcert=local_dev/certs/client.crt sslkey=local_dev/certs/client.key sslrootcert=local_dev/certs/ca.crt\""
