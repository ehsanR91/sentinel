#!/bin/bash
# =============================================================================
# fix-repo-keys.sh — Cross-distro repository GPG / signing key repair
# Detects the package manager family and fixes missing or broken signing keys.
#
# Supported families:
#   apt     (Debian, Ubuntu, Mint, Pop!_OS, …)
#   dnf     (Fedora, RHEL 8+, Rocky, AlmaLinux, CentOS Stream, …)
#   yum     (CentOS 7, older RHEL, Amazon Linux 2)
#   pacman  (Arch, Manjaro, EndeavourOS, …)
#   zypper  (openSUSE Leap/Tumbleweed, SLES)
#
# Usage: sudo bash sentinelcore/deploy/fix-repo-keys.sh
# =============================================================================
set -euo pipefail

[[ $EUID -ne 0 ]] && { echo "Run as root: sudo bash fix-repo-keys.sh"; exit 1; }

# ── Colour helpers ────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC}  $*"; }
info() { echo -e "      $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERR]${NC}  $*"; }

# ── Distro + package-manager detection ───────────────────────────────────────
DISTRO_ID="unknown"; DISTRO_NAME="Unknown Linux"; PKG_FAMILY="unknown"

if [[ -f /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    DISTRO_ID="${ID:-unknown}"
    DISTRO_NAME="${PRETTY_NAME:-$ID}"
fi

if   command -v apt-get &>/dev/null; then PKG_FAMILY="apt"
elif command -v dnf     &>/dev/null; then PKG_FAMILY="dnf"
elif command -v yum     &>/dev/null; then PKG_FAMILY="yum"
elif command -v pacman  &>/dev/null; then PKG_FAMILY="pacman"
elif command -v zypper  &>/dev/null; then PKG_FAMILY="zypper"
fi

echo "======================================================================"
echo "  Repository Key Repair"
echo "  Distro : $DISTRO_NAME"
echo "  Family : $PKG_FAMILY"
echo "======================================================================"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# APT  (Debian / Ubuntu / Mint / …)
# ─────────────────────────────────────────────────────────────────────────────
fix_apt() {
    local keyring_dir="/usr/share/keyrings"
    mkdir -p "$keyring_dir"

    info "Running apt-get update to detect missing keys..."
    local raw
    raw=$(apt-get update 2>&1 || true)

    local missing
    missing=$(echo "$raw" | grep -oP "(?<=NO_PUBKEY )[0-9A-Fa-f]+" | sort -u || true)

    if [[ -z "$missing" ]]; then
        ok "No missing keys — apt sources are clean."
        return 0
    fi
    warn "Missing key(s): $missing"

    local gnupgtmp
    gnupgtmp=$(mktemp -d)
    # shellcheck disable=SC2064
    trap "rm -rf '$gnupgtmp'" RETURN

    for key_id in $missing; do
        info "Fetching $key_id..."
        local fetched=0

        for ks in \
            "hkps://keyserver.ubuntu.com" \
            "hkps://keys.openpgp.org" \
            "hkp://pool.sks-keyservers.net:80"; do
            if gpg --homedir "$gnupgtmp" \
                    --keyserver "$ks" \
                    --recv-keys "$key_id" 2>/dev/null; then
                fetched=1
                info "  ← $ks"
                break
            fi
        done

        if [[ $fetched -eq 0 ]]; then
            warn "Could not fetch $key_id from any keyserver — skipping."
            continue
        fi

        # Find the keyring file this repo's sources.list already references,
        # or derive one from the key ID.
        local keyring
        keyring=$(grep -rh "signed-by=" /etc/apt/sources.list.d/ 2>/dev/null \
                    | grep -oP "(?<=signed-by=)[^] ]+" | head -1 || true)
        [[ -z "$keyring" ]] && keyring="${keyring_dir}/auto-${key_id,,}.gpg"

        # Append binary key to the keyring (GPG keyring = concatenated packets).
        gpg --homedir "$gnupgtmp" --export "$key_id" >> "$keyring"
        chmod 644 "$keyring"
        ok "Key $key_id → $keyring"

        # If no sources.list entry has a signed-by yet, patch them.
        local needs_patch
        needs_patch=$(grep -rl "packagecloud.io" /etc/apt/sources.list.d/ 2>/dev/null \
                        | xargs grep -L "signed-by" 2>/dev/null || true)
        for src in $needs_patch; do
            sed -i -E \
                "s|^(deb(-src)?[[:space:]]+)(https?://packagecloud\.io)|\1[signed-by=${keyring}] \3|" \
                "$src"
            info "  Patched: $(basename "$src")"
        done
    done

    info "Re-running apt-get update to verify..."
    if apt-get update 2>&1 | grep -q "NO_PUBKEY"; then
        err "Still failing — remaining debug info:"
        info "  Keyring: $(wc -c < "$keyring" 2>/dev/null || echo '?') bytes"
        gpg --no-default-keyring --keyring "$keyring" --list-keys 2>/dev/null || true
        return 1
    fi
    ok "All apt repository keys are valid."
}

# ─────────────────────────────────────────────────────────────────────────────
# DNF / YUM  (RHEL, Fedora, Rocky, Alma, CentOS, Amazon Linux…)
# ─────────────────────────────────────────────────────────────────────────────
fix_rpm() {
    local pm="$1"   # "dnf" or "yum"
    local rpm_gpg_dir="/etc/pki/rpm-gpg"

    info "Running $pm check-update to detect key issues..."
    local raw
    raw=$($pm check-update 2>&1 || true)

    if ! echo "$raw" | grep -qiE "gpg|public key|key not installed|not trusted"; then
        # check-update may not show key errors; try a safer probe
        raw=$(LANG=C $pm makecache 2>&1 || true)
    fi

    if ! echo "$raw" | grep -qiE "gpg|public key|key not installed|not trusted"; then
        ok "No GPG key issues detected for $pm repositories."
        return 0
    fi

    warn "GPG key issues detected."

    # ── Netdata (packagecloud.io) ─────────────────────────────────────────
    if ls /etc/yum.repos.d/netdata* &>/dev/null 2>/dev/null || \
       $pm repolist 2>/dev/null | grep -q "netdata"; then
        info "Netdata repository detected — importing Netdata GPG key..."
        # Try the packagecloud key URL first, fall back to keyserver
        if ! rpm --import "https://packagecloud.io/netdata/netdata/gpgkey" 2>/dev/null; then
            local gnupgtmp; gnupgtmp=$(mktemp -d)
            # shellcheck disable=SC2064
            trap "rm -rf '$gnupgtmp'" RETURN
            gpg --homedir "$gnupgtmp" \
                --keyserver hkps://keyserver.ubuntu.com \
                --recv-keys 54832F89F09FED90 2>/dev/null && \
            gpg --homedir "$gnupgtmp" --export 54832F89F09FED90 \
                > "${rpm_gpg_dir}/RPM-GPG-KEY-netdata" && \
            rpm --import "${rpm_gpg_dir}/RPM-GPG-KEY-netdata" 2>/dev/null || \
            warn "Could not import Netdata GPG key — try: rpm --import https://packagecloud.io/netdata/netdata/gpgkey"
        else
            ok "Netdata GPG key imported via packagecloud.io"
        fi
    fi

    # ── Generic: collect key URLs from repo files and import them ─────────
    local key_urls
    key_urls=$(grep -rh "^gpgkey=" /etc/yum.repos.d/ 2>/dev/null \
                | sed 's/^gpgkey=//' | sort -u || true)
    for url in $key_urls; do
        info "Importing key from repo file: $url"
        rpm --import "$url" 2>/dev/null && \
            ok "  Imported: $url" || \
            warn "  Failed to import: $url"
    done

    info "Re-running $pm makecache to verify..."
    if $pm makecache 2>&1 | grep -qiE "gpg|public key"; then
        warn "GPG issues may remain — run: $pm update --nogpgcheck  (only to diagnose; re-enable checks after)"
    else
        ok "All $pm repository keys appear valid."
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# PACMAN  (Arch, Manjaro, EndeavourOS, …)
# ─────────────────────────────────────────────────────────────────────────────
fix_pacman() {
    info "Initialising / refreshing pacman keyring..."
    pacman-key --init 2>/dev/null || true
    pacman-key --populate archlinux 2>/dev/null || true

    info "Refreshing all keys from keyserver..."
    pacman-key --refresh-keys 2>/dev/null || \
        warn "Key refresh had errors — some keys may not have been updated."

    # Detect untrusted keys from a dry-run upgrade
    local untrusted
    untrusted=$(pacman -Sy 2>&1 | grep -oP "(?<=key unknown: )\S+" | sort -u || true)
    for key_id in $untrusted; do
        info "Signing key $key_id..."
        pacman-key --recv-keys "$key_id" 2>/dev/null && \
        pacman-key --lsign-key "$key_id" 2>/dev/null && \
            ok "Signed: $key_id" || \
            warn "Could not sign key $key_id"
    done

    if [[ -z "$untrusted" ]]; then
        ok "pacman keyring is up to date."
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# ZYPPER  (openSUSE, SLES)
# ─────────────────────────────────────────────────────────────────────────────
fix_zypper() {
    info "Refreshing zypper repositories and auto-importing keys..."
    if zypper --gpg-auto-import-keys refresh 2>&1; then
        ok "zypper repositories refreshed and keys imported."
    else
        warn "zypper refresh had errors — try: zypper --gpg-auto-import-keys dup"
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Dispatch
# ─────────────────────────────────────────────────────────────────────────────
case "$PKG_FAMILY" in
    apt)    fix_apt ;;
    dnf)    fix_rpm "dnf" ;;
    yum)    fix_rpm "yum" ;;
    pacman) fix_pacman ;;
    zypper) fix_zypper ;;
    *)
        err "Unsupported package manager. Supported: apt, dnf, yum, pacman, zypper"
        exit 1
        ;;
esac
