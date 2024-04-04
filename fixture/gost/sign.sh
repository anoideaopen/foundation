#!/bin/bash

# Specify file paths
PRIVATE_KEY="gost_private.key.example"
CSR="gost.csr.example"
CERTIFICATE="gost_certificate.crt.example"
CERTIFICATE_TEXT="gost_certificate.txt"
DATA="data.txt"
SIGNATURE_RAW="signature.bin"
SIGNATURE_BASE64="signature_base64.txt"
PUBLIC_KEY="public_key.pem.example"

# Private key creation using GOST 2012 algorithm
openssl genpkey -algorithm gost2012_256 -pkeyopt paramset:A -out "$PRIVATE_KEY"

# Create a certificate request (CSR) using this private key
openssl req -new -key "$PRIVATE_KEY" -out "$CSR" -subj "/C=RU/ST=Moscow/L=Moscow/O=Example/OU=IT/CN=example.com"

# Creating a self-signed certificate based on CSR
openssl x509 -req -days 365 -in "$CSR" -signkey "$PRIVATE_KEY" -out "$CERTIFICATE" -extfile <(printf "basicConstraints=CA:TRUE\nkeyUsage=nonRepudiation,digitalSignature,keyEncipherment")

# Create a file with data for signing
echo "Hello World" > "$DATA"

# Computation of message digest using GOST 2012 algorithm
DIGEST_RAW="digest.bin"
openssl dgst -md_gost12_256 -binary "$DATA" > "$DIGEST_RAW"

# Convert digest to Base64 format
DIGEST_BASE64=$(base64 -w 0 "$DIGEST_RAW")

# Output the digest in Base64 format to the screen
echo "Message digest in Base64 format: $DIGEST_BASE64"

# Signing data using a private key
openssl dgst -md_gost12_256 -sign "$PRIVATE_KEY" -out "$SIGNATURE_RAW" "$DATA"

# Convert signature to Base64 format without metadata
base64 -w 0 "$SIGNATURE_RAW" > "$SIGNATURE_BASE64"

# Exporting a public key
openssl pkey -pubout -in "$PRIVATE_KEY" -out "$PUBLIC_KEY"

# Saving certificate information to a text file
openssl x509 -in "$CERTIFICATE" -text -noout > "$CERTIFICATE_TEXT"

# Display a message about script completion
echo "The creation of the GOST certificate and signature is complete."
