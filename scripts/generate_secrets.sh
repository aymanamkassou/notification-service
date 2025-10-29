#!/usr/bin/env bash
set -euo pipefail

# Generates VAPID keys and an HMAC secret for local development
# Usage: ./scripts/generate_secrets.sh

have_cmd() { command -v "$1" >/dev/null 2>&1; }

info() { echo "[info] $*"; }

# Generate VAPID keys
if have_cmd web-push; then
  info "Generating VAPID keys with local web-push CLI"
  KEYS_JSON=$(web-push generate-vapid-keys --json)
elif have_cmd docker; then
  info "Generating VAPID keys using Docker (node:20-alpine + web-push)"
  KEYS_JSON=$(docker run --rm node:20-alpine sh -lc "npm -g i web-push >/dev/null 2>&1; web-push generate-vapid-keys --json")
else
  echo "web-push CLI or docker is required to generate VAPID keys. Install one and retry." >&2
  exit 1
fi

VAPID_PUBLIC_KEY=$(echo "$KEYS_JSON" | grep -o '"publicKey":"[^"]*' | cut -d '"' -f4)
VAPID_PRIVATE_KEY=$(echo "$KEYS_JSON" | grep -o '"privateKey":"[^"]*' | cut -d '"' -f4)

# Generate HMAC secret
if have_cmd openssl; then
  HMAC_SECRET=$(openssl rand -base64 32)
else
  info "openssl not found; generating HMAC secret from /dev/urandom"
  HMAC_SECRET=$(head -c 32 /dev/urandom | base64)
fi

cat <<EOF
# --- Add these to your .env ---
VAPID_PUBLIC_KEY=${VAPID_PUBLIC_KEY}
VAPID_PRIVATE_KEY=${VAPID_PRIVATE_KEY}
HMAC_SECRET=${HMAC_SECRET}
# ------------------------------
EOF
