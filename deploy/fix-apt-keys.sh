#!/bin/bash
# fix-apt-keys.sh — repair the Netdata packagecloud APT GPG key on Ubuntu 24.04
# Run with: sudo bash sentinelcore/deploy/fix-apt-keys.sh
set -euo pipefail

[[ $EUID -ne 0 ]] && { echo "Run as root: sudo bash fix-apt-keys.sh"; exit 1; }

NC_KEYRING=/usr/share/keyrings/netdata-archive-keyring.gpg

# ── Step 1: detect which keys the repos actually need ─────────────────────────
echo "[1] Detecting missing GPG keys..."
MISSING=$(apt-get update 2>&1 \
    | grep -oP "(?<=NO_PUBKEY )[0-9A-Fa-f]+" \
    | sort -u || true)

if [[ -z "$MISSING" ]]; then
    echo "    No missing keys found — apt sources are already clean."
    exit 0
fi
echo "    Missing: $MISSING"

# ── Step 2: import each missing key directly into the netdata keyring ─────────
# We use a temp GNUPGHOME so we don't pollute root's personal keyring, then
# export just the fetched key in binary format and append it to the keyring file.
echo "[2] Importing key(s) into $NC_KEYRING..."

GNUPGTMP=$(mktemp -d)
trap 'rm -rf "$GNUPGTMP"' EXIT

for KEY_ID in $MISSING; do
    echo "    Fetching $KEY_ID from keyserver.ubuntu.com..."
    if gpg --homedir "$GNUPGTMP" \
            --keyserver hkps://keyserver.ubuntu.com \
            --recv-keys "$KEY_ID" 2>/dev/null; then
        # Export in binary (not armored) format and append to the keyring file.
        # GPG keyring files are just concatenated packets — appending is safe.
        gpg --homedir "$GNUPGTMP" --export "$KEY_ID" >> "$NC_KEYRING"
        chmod 644 "$NC_KEYRING"
        echo "    Key $KEY_ID appended to keyring."
    else
        echo "    WARNING: could not fetch $KEY_ID from keyserver — trying pool.sks-keyservers.net..."
        gpg --homedir "$GNUPGTMP" \
            --keyserver hkp://pool.sks-keyservers.net \
            --recv-keys "$KEY_ID" 2>/dev/null && \
        gpg --homedir "$GNUPGTMP" --export "$KEY_ID" >> "$NC_KEYRING" && \
        chmod 644 "$NC_KEYRING" && \
        echo "    Key $KEY_ID appended (via fallback keyserver)." || \
        echo "    ERROR: could not fetch key $KEY_ID from any keyserver."
    fi
done

# ── Step 3: verify ────────────────────────────────────────────────────────────
echo ""
echo "[3] Keys now in keyring:"
gpg --no-default-keyring --keyring "$NC_KEYRING" --list-keys 2>/dev/null || true

echo ""
echo "[4] Running apt-get update to verify..."
if apt-get update 2>&1 | grep -q "NO_PUBKEY"; then
    echo "STILL FAILING — debug info:"
    echo "  Keyring: $NC_KEYRING ($(wc -c < "$NC_KEYRING") bytes)"
    gpg --no-default-keyring --keyring "$NC_KEYRING" --list-keys 2>/dev/null || true
else
    echo "SUCCESS — all repository GPG keys are valid."
fi
