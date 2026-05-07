#!/bin/bash
# =============================================================================
# fix-all-repo-keys.sh — Comprehensive fix for apt/dpkg GPG key and filesystem issues
# 
# This script addresses:
# 1. Read-only filesystem detection and remount attempts
# 2. GPG key import failures for common repositories (Netdata, Docker, GitHub CLI)
# 3. dpkg lock file cleanup
# 4. Repository source list validation
#
# Usage: sudo bash sentinelcore/deploy/fix-all-repo-keys.sh
# =============================================================================
set -euo pipefail

[[ $EUID -ne 0 ]] && { echo "Run as root: sudo bash fix-all-repo-keys.sh"; exit 1; }

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

ok() { echo -e "${GREEN}[OK]${NC} $*"; }
info() { echo -e " $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err() { echo -e "${RED}[ERR]${NC} $*"; }

echo "======================================================================"
echo " Repository & Package Installation Fix Script"
echo "======================================================================"
echo ""

# ── Step 1: Check filesystem writability ────────────────────────────────────
info "Step 1: Checking filesystem writability..."
FS_WRITABLE=true
TEST_FILE="/tmp/.write_test_$$"

if ! touch "$TEST_FILE" 2>/dev/null; then
    warn("Cannot write to /tmp - critical filesystem issue")
    FS_WRITABLE=false
else
    rm -f "$TEST_FILE"
fi

# Check if /usr is writable (needed for keyrings)
if ! touch /usr/share/keyrings/.write_test_$$ 2>/dev/null; then
    warn "/usr/share/keyrings/ appears read-only"
    info "Attempting to remount /usr as read-write..."
    mount -o remount,rw /usr 2>/dev/null && ok "Remounted /usr as rw" || warn "Could not remount /usr"
fi

# Check if /etc is writable
if ! touch /etc/apt/keyrings/.write_test_$$ 2>/dev/null; then
    warn "/etc/apt/keyrings/ appears read-only"
    info "Attempting to remount /etc as read-write..."
    mount -o remount,rw /etc 2>/dev/null && ok "Remounted /etc as rw" || warn "Could not remount /etc"
fi
rm -f /usr/share/keyrings/.write_test_$$ /etc/apt/keyrings/.write_test_$$ 2>/dev/null

# ── Step 2: Clean up any stale dpkg/apt locks ───────────────────────────────
info "Step 2: Checking for stale dpkg/apt locks..."
for lock in /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock /var/cache/apt/archives/lock; do
    if [ -f "$lock" ]; then
        if fuser "$lock" >/dev/null 2>&1; then
            warn "Process holding $lock - waiting..."
            sleep 2
        fi
        # Only remove if no process is holding it
        if ! fuser "$lock" >/dev/null 2>&1; then
            rm -f "$lock" && ok "Removed stale lock: $lock" || true
        fi
    fi
done

# Fix any interrupted dpkg state
if [ -f /var/lib/dpkg/lock-frontend ]; then
    dpkg --configure -a 2>/dev/null || true
fi

# ── Step 3: Create necessary directories ────────────────────────────────────
info "Step 3: Ensuring keyring directories exist..."
mkdir -p /usr/share/keyrings 2>/dev/null || warn "Could not create /usr/share/keyrings"
mkdir -p /etc/apt/keyrings 2>/dev/null || warn "Could not create /etc/apt/keyrings"
chmod 755 /usr/share/keyrings /etc/apt/keyrings 2>/dev/null || true

# ── Step 4: Fix common repository keys ──────────────────────────────────────
info "Step 4: Fetching and importing repository GPG keys..."

fix_keyring() {
    local name="$1"
    local url="$2"
    local keyring="$3"
    
    info "Importing $name key..."
    GNUPG_TMP=$(mktemp -d)
    
    if curl -fsSL "$url" --connect-timeout 10 \
    | gpg --homedir "$GNUPG_TMP" --dearmor -o "$keyring" 2>/dev/null; then
        chmod 644 "$keyring"
        ok "$name key imported to $keyring"
        rm -rf "$GNUPG_TMP"
        return 0
    else
        warn "Failed to import $name key"
        rm -rf "$GNUPG_TMP"
        return 1
    fi
}

# Docker key
if [ -f /etc/apt/sources.list.d/docker.list ] || grep -q "download.docker.com" /etc/apt/sources.list.d/* 2>/dev/null; then
    fix_keyring "Docker" "https://download.docker.com/linux/ubuntu/gpg" "/etc/apt/keyrings/docker.gpg" || true
fi

# Netdata key
if [ -f /etc/apt/sources.list.d/netdata.list ] || grep -q "packagecloud.io/netdata" /etc/apt/sources.list.d/* 2>/dev/null; then
    fix_keyring "Netdata" "https://packagecloud.io/netdata/netdata/gpgkey" "/usr/share/keyrings/netdata-archive-keyring.gpg" || true
fi

# GitHub CLI key
if [ -f /etc/apt/sources.list.d/github-cli.list ] || grep -q "cli.github.com" /etc/apt/sources.list.d/* 2>/dev/null; then
    fix_keyring "GitHub CLI" "https://cli.github.com/packages/githubcli-archive-keyring.gpg" "/usr/share/keyrings/githubcli-archive-keyring.gpg" || true
fi

# NodeSource key
if [ -f /etc/apt/sources.list.d/nodesource.list ] || grep -q "deb.nodesource.com" /etc/apt/sources.list.d/* 2>/dev/null; then
    fix_keyring "NodeSource" "https://deb.nodesource.com/gpgkey/nodesource.gpg.key" "/usr/share/keyrings/nodesource.gpg" || true
fi

# ── Step 5: Fix source list files with signed-by directives ─────────────────
info "Step 5: Validating source list files..."

fix_source_list() {
    local file="$1"
    local keyring="$2"
    local repo_url="$3"
    
    if [ -f "$file" ] && grep -q "$repo_url" "$file" && ! grep -q "signed-by=" "$file"; then
        info "Adding signed-by directive to $file..."
        sed -i "s|deb https|$repo_url|deb [signed-by=$keyring] https|$repo_url|g" "$file" 2>/dev/null || true
        ok "Updated $file"
    fi
}

# Fix source lists if needed
for src in /etc/apt/sources.list.d/*.list; do
    [ -f "$src" ] || continue
    case "$src" in
        *docker*) fix_source_list "$src" "/etc/apt/keyrings/docker.gpg" "download.docker.com" ;;
        *netdata*) fix_source_list "$src" "/usr/share/keyrings/netdata-archive-keyring.gpg" "packagecloud.io/netdata" ;;
        *github-cli*) fix_source_list "$src" "/usr/share/keyrings/githubcli-archive-keyring.gpg" "cli.github.com" ;;
        *nodesource*) fix_source_list "$src" "/usr/share/keyrings/nodesource.gpg" "deb.nodesource.com" ;;
    esac
done

# ── Step 6: Update apt cache ────────────────────────────────────────────────
info "Step 6: Updating apt cache..."
apt-get clean
apt-get update 2>&1 | tee /tmp/apt-update.log

if grep -qi "NO_PUBKEY\|gpg\|key" /tmp/apt-update.log; then
    warn "GPG key issues detected in update output"
    info "Run 'sudo bash sentinelcore/deploy/fix-repo-keys.sh' for detailed key repair"
else
    ok "apt update completed successfully"
fi

# ── Step 7: Fix any broken packages ─────────────────────────────────────────
info "Step 7: Checking for broken packages..."
if dpkg --audit 2>/dev/null; then
    ok "No broken packages detected"
else
    warn "Broken packages detected, attempting repair..."
    apt-get install -f -y 2>/dev/null || warn "Automatic repair failed"
fi

# ── Summary ─────────────────────────────────────────────────────────────────
echo ""
echo "======================================================================"
echo " Fix Complete"
echo "======================================================================"
echo ""
ok "Repository keys and sources have been processed."
info "Next steps:"
echo "  1. Try installing packages again: sudo apt install <package>"
echo "  2. If issues persist, run: sudo bash sentinelcore/deploy/fix-repo-keys.sh"
echo "  3. For ClamAV specifically: sudo apt install clamav-daemon"
echo ""
