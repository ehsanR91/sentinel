#!/usr/bin/env bash
# SentinelCore — Development Environment Setup
# Installs sudoers entries so the backend can run privileged commands
# without being deployed as a systemd service.
#
# Usage:
#   sudo bash deploy-dev.sh
#   sudo bash deploy-dev.sh --user myuser   # explicit user override

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ── Resolve script directory (works even when called from another dir) ─────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUDOERS_SRC="${SCRIPT_DIR}/deploy/sentinelcore.sudoers"
SUDOERS_DEST="/etc/sudoers.d/sentinelcore"

# ── Must run as root ───────────────────────────────────────────────────────────
[[ "$EUID" -eq 0 ]] || error "Please run with sudo: sudo bash deploy-dev.sh"

# ── Source file must exist ─────────────────────────────────────────────────────
[[ -f "$SUDOERS_SRC" ]] || error "Source sudoers file not found: $SUDOERS_SRC"

# ── Resolve target user ────────────────────────────────────────────────────────
TARGET_USER=""

# Allow --user flag
while [[ $# -gt 0 ]]; do
  case "$1" in
    --user) TARGET_USER="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# If not passed explicitly, use SUDO_USER (the user who called sudo)
if [[ -z "$TARGET_USER" ]]; then
  TARGET_USER="${SUDO_USER:-}"
fi

# Final fallback: ask
if [[ -z "$TARGET_USER" ]]; then
  read -r -p "$(echo -e "${BOLD}Username to grant sudo access to${NC}: ")" TARGET_USER
fi

[[ -n "$TARGET_USER" ]] || error "No username provided."
id "$TARGET_USER" &>/dev/null || error "User '$TARGET_USER' does not exist on this system."

echo -e "\n${BOLD}SentinelCore — Dev sudoers setup${NC}"
echo -e "  User   : ${CYAN}${TARGET_USER}${NC}"
echo -e "  Source : ${DIM}${SUDOERS_SRC}${NC}"
echo -e "  Target : ${DIM}${SUDOERS_DEST}${NC}\n"

# ── Write sudoers file with <USER> substituted ─────────────────────────────────
info "Writing sudoers file..."
sed "s/<USER>/${TARGET_USER}/g" "$SUDOERS_SRC" | tee "$SUDOERS_DEST" > /dev/null
chmod 440 "$SUDOERS_DEST"
success "Written: $SUDOERS_DEST"

# ── Validate with visudo ───────────────────────────────────────────────────────
info "Validating sudoers syntax..."
if visudo -c -f "$SUDOERS_DEST" &>/dev/null; then
  success "Syntax OK"
else
  rm -f "$SUDOERS_DEST"
  error "Sudoers syntax check failed — file removed. Check $SUDOERS_SRC for errors."
fi

# ── Quick smoke-test: can the user run sudo -n true? ──────────────────────────
info "Smoke-testing passwordless sudo for '${TARGET_USER}'..."
if sudo -u "$TARGET_USER" sudo -n true 2>/dev/null; then
  success "Passwordless sudo is working for '${TARGET_USER}'"
else
  warn "Smoke-test could not verify (may need a new shell). Try: sudo -n true"
fi

# ── UFW sudoers (separate file referenced by health check) ─────────────────────
UFW_SUDOERS_DEST="/etc/sudoers.d/sentinelcore-ufw"
if [[ ! -f "$UFW_SUDOERS_DEST" ]]; then
  info "Creating minimal sentinelcore-ufw sudoers entry..."
  cat > "$UFW_SUDOERS_DEST" <<EOF
# SentinelCore UFW sudoers (dev)
${TARGET_USER} ALL=(root) NOPASSWD: /usr/sbin/ufw *, /usr/bin/ufw *
EOF
  chmod 440 "$UFW_SUDOERS_DEST"
  if visudo -c -f "$UFW_SUDOERS_DEST" &>/dev/null; then
    success "Written: $UFW_SUDOERS_DEST"
  else
    rm -f "$UFW_SUDOERS_DEST"
    warn "UFW sudoers syntax check failed — skipped."
  fi
else
  info "UFW sudoers already exists at $UFW_SUDOERS_DEST — skipped."
fi

echo -e "\n${GREEN}${BOLD}Dev environment sudoers setup complete.${NC}"
echo -e "  You can now run the backend as ${CYAN}${TARGET_USER}${NC} without a password prompt."
echo -e "  To remove:  ${DIM}sudo rm ${SUDOERS_DEST} ${UFW_SUDOERS_DEST}${NC}\n"
