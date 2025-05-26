#!/bin/bash

mkdir -p certs

# Generate CA key and cert
openssl genrsa -out certs/ca.key 4096
openssl req -x509 -new -nodes -key certs/ca.key -sha256 -days 3650 -out certs/ca.crt -subj "/C=IN/ST=Telangana/L=Hyderabad/O=MyOrg/OU=MyUnit/CN=MyCA"

# Generate server key and CSR
openssl genrsa -out certs/server.key 4096
openssl req -new -key certs/server.key -out certs/server.csr -subj "/C=IN/ST=Telangana/L=Hyderabad/O=MyOrg/OU=MyUnit/CN=localhost"

# Create server certificate with SAN
openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 3650 -sha256 -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1")

# Generate client key and CSR
openssl genrsa -out certs/client.key 4096
openssl req -new -key certs/client.key -out certs/client.csr -subj "/C=IN/ST=Telangana/L=Hyderabad/O=MyOrg/OU=MyUnit/CN=client"

# Create client certificate
openssl x509 -req -in certs/client.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/client.crt -days 3650 -sha256

echo "Certificates generated successfully!"