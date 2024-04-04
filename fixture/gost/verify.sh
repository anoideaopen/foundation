#!/bin/bash

# Specify file paths
PUBLIC_KEY="public_key.pem.example"
SIGNATURE_BASE64="signature_base64.txt"
DATA="data.txt"
SIGNATURE_RAW="signature.bin"

# Decoding signature from Base64 to raw
echo "Decoding signature from Base64..."
base64 -d "$SIGNATURE_BASE64" > "$SIGNATURE_RAW"
if [ $? -ne 0 ]; then
    echo "Error during signature decoding."
    exit 1
fi
echo "The signature has been successfully decoded to raw format."

# Signature verification
echo "Signature verification..."
openssl dgst -md_gost12_256 -verify "$PUBLIC_KEY" -signature "$SIGNATURE_RAW" "$DATA"
if [ $? -eq 0 ]; then
    echo "The signature is correct."
else
    echo "The signature is wrong."
    exit 1
fi
