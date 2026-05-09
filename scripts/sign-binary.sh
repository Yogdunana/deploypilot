#!/usr/bin/env bash
# sign-binary.sh - CI helper script to sign a binary with Ed25519
#
# Usage: ./scripts/sign-binary.sh <binary-path> <signature-output-path>
#
# This script generates an Ed25519 key pair (or reuses existing keys),
# signs the specified binary, and writes the signature to the output path.
#
# Environment variables:
#   DEPLOYSIGN_PUBLIC_KEY  - Path to existing public key (optional)
#   DEPLOYSIGN_PRIVATE_KEY - Path to existing private key (optional)
#
# If key paths are not provided, a new key pair will be generated
# in the same directory as the binary.

set -euo pipefail

BINARY_PATH="${1:?Usage: sign-binary.sh <binary-path> <signature-output-path>}"
SIGNATURE_PATH="${2:?Usage: sign-binary.sh <binary-path> <signature-output-path>}"

if [ ! -f "$BINARY_PATH" ]; then
    echo "error: binary file not found: $BINARY_PATH" >&2
    exit 1
fi

# Determine key directory
KEY_DIR="$(dirname "$BINARY_PATH")"
PUBLIC_KEY_PATH="${DEPLOYSIGN_PUBLIC_KEY:-$KEY_DIR/signing_public.pem}"
PRIVATE_KEY_PATH="${DEPLOYSIGN_PRIVATE_KEY:-$KEY_DIR/signing_private.pem}"

# Generate key pair if not already present
if [ ! -f "$PUBLIC_KEY_PATH" ] || [ ! -f "$PRIVATE_KEY_PATH" ]; then
    echo "generating Ed25519 key pair..."
    # Use Go to generate keys and sign the binary
    SIGNER_DIR="$(cd "$(dirname "$0")/.." && pwd)"
    go run "$SIGNER_DIR/cmd/sign-tool/main.go" generate \
        --public-key "$PUBLIC_KEY_PATH" \
        --private-key "$PRIVATE_KEY_PATH"
    echo "keys generated:"
    echo "  public:  $PUBLIC_KEY_PATH"
    echo "  private: $PRIVATE_KEY_PATH"
fi

# Sign the binary
echo "signing binary: $BINARY_PATH"
SIGNER_DIR="$(cd "$(dirname "$0")/.." && pwd)"
go run "$SIGNER_DIR/cmd/sign-tool/main.go" sign \
    --binary "$BINARY_PATH" \
    --signature "$SIGNATURE_PATH" \
    --private-key "$PRIVATE_KEY_PATH"

echo "signature written to: $SIGNATURE_PATH"
echo "done."
