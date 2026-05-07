#!/usr/bin/env bash
# SentinelCore Deployment & Maintenance Script
# Usage: sudo bash deploy-sentinel.sh

set -euo pipefail

###############################################################################
# Colour helpers
###############################################################################
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# Visual helpers
section()  { echo -e "\n${BOLD}── $* $(printf '─%.0s' $(seq 1 $((55 - ${#1}))))${NC}"; }
ok_item()  { echo -e "  ${GREEN}+${NC} $*"; }
warn_item(){ echo -e "  ${YELLOW}!${NC} $*"; }
info_item(){ echo -e "  ${CYAN}.${NC} $*"; }

SCRIPT_VERSION="2.1"

prompt() {
  local var="$1" msg="$2" default="$3"
  read -r -p "$(echo -e "${BOLD}${msg}${NC} [default: ${CYAN}${default}${NC}]: ")" input
  eval "$var=\"${input:-$default}\""
}

promptpw() {
  local var="$1" msg="$2"
  while true; do
    read -r -s -p "$(echo -e "${BOLD}${msg}${NC}: ")" pw; echo
    read -r -s -p "$(echo -e "${BOLD}Confirm password${NC}: ")" pw2; echo
    [[ "$pw" == "$pw2" ]] || { warn "Passwords do not match, try again."; continue; }
    [[ ${#pw} -ge 12 ]]   || { warn "Password must be at least 12 characters."; continue; }
    break
  done
  eval "$var=\"$pw\""
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/sentinelcore"

###############################################################################
# .env reader  (read_env KEY)
###############################################################################
read_env() {
  grep "^${1}=" "${INSTALL_DIR}/.env" 2>/dev/null | cut -d'=' -f2-
}

###############################################################################
# Distro / firewall detection helpers
###############################################################################

# Returns the distro ID from /etc/os-release (e.g. ubuntu, debian, fedora, arch)
detect_distro() {
  if [[ -f /etc/os-release ]]; then
    # shellcheck source=/dev/null
    local ID; ID=$(. /etc/os-release && echo "${ID:-unknown}")
    echo "$ID"
  elif [[ -f /etc/debian_version ]]; then
    echo "debian"
  elif [[ -f /etc/redhat-release ]]; then
    echo "rhel"
  elif [[ -f /etc/arch-release ]]; then
    echo "arch"
  else
    echo "unknown"
  fi
}

# Returns the distro-family: debian | rhel | arch | unknown
detect_distro_family() {
  local id; id=$(detect_distro)
  case "$id" in
    ubuntu|debian|linuxmint|pop|kali|raspbian) echo "debian" ;;
    fedora|rhel|centos|rocky|almalinux|ol)     echo "rhel"   ;;
    arch|manjaro|endeavouros)                  echo "arch"   ;;
    opensuse*|sles)                            echo "suse"   ;;
    *)                                         echo "unknown";;
  esac
}

# Finds the ufw binary across common install locations.
# Prints the full path, or empty string if not found.
find_ufw_bin() {
  local bin
  # 1. Try the current PATH first (handles custom installs)
  bin=$(command -v ufw 2>/dev/null)
  [[ -x "$bin" ]] && { echo "$bin"; return; }
  # 2. Search known locations (covers both old /usr/sbin and new /usr/bin placement)
  for p in /usr/sbin/ufw /usr/bin/ufw /sbin/ufw /bin/ufw; do
    [[ -x "$p" ]] && { echo "$p"; return; }
  done
  echo ""
}

# Detects the active firewall tool: ufw | firewalld | iptables | nftables | none
detect_firewall() {
  [[ -n "$(find_ufw_bin)" ]]                        && { echo "ufw";       return; }
  command -v firewall-cmd &>/dev/null                && { echo "firewalld"; return; }
  command -v nft          &>/dev/null                && { echo "nftables";  return; }
  command -v iptables     &>/dev/null                && { echo "iptables";  return; }
  echo "none"
}

###############################################################################
# nftables helpers (best-effort)
###############################################################################
nft_has_default_input_chain() {
  command -v nft &>/dev/null || return 1
  nft list chain inet filter input &>/dev/null
}

nft_port_open() {
  local port=$1
  nft_has_default_input_chain || return 1
  nft list chain inet filter input 2>/dev/null | grep -qE "tcp dport ${port} .* accept"
}

nft_open() {
  local port=$1
  if ! nft_has_default_input_chain; then
    warn "nftables detected but default chain 'inet filter input' not found."
    warn "Open port ${port}/tcp in your nft ruleset manually (table/chain names may differ)."
    warn "Hint: nft list ruleset | sed -n '1,120p'"
    return 1
  fi

  if nft_port_open "$port"; then
    success "nftables: port ${port}/tcp already allowed (inet filter input)"
    return 0
  fi

  nft add rule inet filter input tcp dport "$port" accept
  success "nftables: opened port ${port}/tcp (runtime rule)"
  warn "This rule may not persist after reboot. Persist it in /etc/nftables.conf (or your distro's nftables config) and reload nftables."
}

###############################################################################
# UFW helpers
###############################################################################
ufw_open() {
  local port=$1
  local UFW_BIN; UFW_BIN=$(find_ufw_bin)
  if [[ -z "$UFW_BIN" ]]; then
    warn "UFW not found — skipping firewall rule for port ${port}."
    return
  fi
  "$UFW_BIN" allow "${port}/tcp"
  "$UFW_BIN" reload
  success "UFW: opened port ${port}/tcp"
}

ufw_close() {
  local port=$1
  local UFW_BIN; UFW_BIN=$(find_ufw_bin)
  [[ -z "$UFW_BIN" ]] && return
  "$UFW_BIN" delete allow "${port}/tcp" 2>/dev/null || true
  "$UFW_BIN" reload
  info "UFW: closed port ${port}/tcp"
}

ufw_port_open() {
  local port=$1
  local UFW_BIN; UFW_BIN=$(find_ufw_bin)
  [[ -z "$UFW_BIN" ]] && return 1
  "$UFW_BIN" status 2>/dev/null | grep -qE "^${port}[/ ].*ALLOW"
}

###############################################################################
# Admin binary runner  (calls the binary's built-in admin subcommand)
###############################################################################
run_admin() {
  local db_path; db_path=$(read_env DB_PATH)
  local secrets_key_path; secrets_key_path=$(read_env SECRETS_KEY_PATH)
  if [[ -z "$db_path" ]]; then
    db_path="${INSTALL_DIR}/data/app.db"
  fi
  DB_PATH="$db_path" SECRETS_KEY_PATH="$secrets_key_path" "${INSTALL_DIR}/sentinelcore" admin "$@"
}

###############################################################################
# Maintenance actions
###############################################################################
do_reset_password() {
  echo -e "\n${BOLD}─── Reset User Password ────────────────────────────────────────────${NC}\n"
  info "Available users:"
  run_admin list-users 2>/dev/null | sed 's/^/    /'
  echo ""
  local username newpass
  prompt username "Username to reset" "admin"
  promptpw newpass "New password (min 12 chars)"
  info "Updating password for '${username}'..."
  if run_admin reset-password "$username" "$newpass"; then
    success "Password updated."
    if systemctl is-active sentinelcore &>/dev/null 2>&1; then
      info "Restarting service to invalidate existing sessions..."
      systemctl restart sentinelcore
      success "Service restarted."
    fi
  fi
}

do_reset_2fa() {
  echo -e "\n${BOLD}─── Remove 2FA From User ───────────────────────────────────────────${NC}\n"
  info "Available users:"
  run_admin list-users 2>/dev/null | sed 's/^/    /'
  echo ""
  local username
  prompt username "Username to remove 2FA from" "admin"
  info "Removing 2FA for '${username}'..."
  if run_admin reset-2fa "$username"; then
    success "2FA disabled for '${username}'. They can re-enable it from Settings."
  fi
}

do_manage_ufw() {
  echo -e "\n${BOLD}─── UFW Firewall Management ────────────────────────────────────────${NC}\n"
  local UFW_BIN; UFW_BIN=$(find_ufw_bin)
  if [[ -z "$UFW_BIN" ]]; then
    warn "UFW is not installed on this system."
    return
  fi
  local listen_addr port
  listen_addr=$(read_env LISTEN_ADDR)
  port="${listen_addr##*:}"
  local status_label
  if ufw_port_open "$port"; then
    status_label="${GREEN}OPEN${NC}"
  else
    status_label="${RED}CLOSED${NC}"
  fi
  echo -e "  Configured port : ${CYAN}${port}${NC}"
  echo -e "  UFW status      : $(echo -e "$status_label")\n"
  echo -e "  ${GREEN}1)${NC} Open port ${port}  (allow public access)"
  echo -e "  ${GREEN}2)${NC} Close port ${port} (block public access)"
  echo -e "  ${GREEN}3)${NC} Open a different port"
  echo -e "  ${GREEN}4)${NC} Close a different port"
  echo -e "  ${GREEN}0)${NC} Back\n"
  local choice
  read -r -p "$(echo -e "${BOLD}Choice [0-4]${NC}: ")" choice
  case "$choice" in
    1) ufw_open "$port" ;;
    2) ufw_close "$port" ;;
    3) local p; prompt p "Port to open" ""; [[ -n "$p" ]] && ufw_open "$p" ;;
    4) local p; prompt p "Port to close" ""; [[ -n "$p" ]] && ufw_close "$p" ;;
    0) return ;;
    *) warn "Invalid choice." ;;
  esac
}

do_uninstall() {
  echo -e "\n${BOLD}─── Uninstall SentinelCore ─────────────────────────────────────────${NC}\n"
  warn "This will stop and remove the SentinelCore systemd service."
  echo ""
  local confirm
  read -r -p "$(echo -e "${BOLD}Are you sure you want to uninstall?${NC} (y/${BOLD}N${NC}): ")" confirm
  [[ "${confirm,,}" == "y" || "${confirm,,}" == "yes" ]] || { info "Uninstall cancelled."; return; }

  # Close UFW port if open
  if [[ -n "$(find_ufw_bin)" ]]; then
    local port; port=$(read_env LISTEN_ADDR | cut -d':' -f2)
    if [[ -n "$port" ]] && ufw_port_open "$port"; then
      ufw_close "$port"
    fi
  fi

  # Stop and remove systemd service
  if systemctl is-active sentinelcore &>/dev/null 2>&1; then
    systemctl stop sentinelcore
    info "Service stopped."
  fi
  if systemctl is-enabled sentinelcore &>/dev/null 2>&1; then
    systemctl disable sentinelcore
  fi
  if [[ -f /etc/systemd/system/sentinelcore.service ]]; then
    rm -f /etc/systemd/system/sentinelcore.service
    systemctl daemon-reload
    success "Systemd service removed."
  fi

  # Optionally delete data
  echo ""
  local del_data
  read -r -p "$(echo -e "${BOLD}Delete ${INSTALL_DIR} and all data?${NC} (y/${BOLD}N${NC}): ")" del_data
  if [[ "${del_data,,}" == "y" || "${del_data,,}" == "yes" ]]; then
    rm -rf "$INSTALL_DIR"
    success "Deleted ${INSTALL_DIR}."
  else
    info "Data preserved at ${INSTALL_DIR} — you can delete it manually when ready."
  fi

  echo ""
  success "SentinelCore has been uninstalled."
}

do_rebuild() {
  # Just jump to the build+install section without changing any config
  # The trick: set IS_REDEPLOY=true, keep existing JWT + .env, skip prompts
  info "Rebuilding SentinelCore (keeping all settings and data)..."
  IS_REDEPLOY=true
  # Ensure sudoers is always refreshed on rebuild
  setup_all_sudoers "$(grep '^User=' /etc/systemd/system/sentinelcore.service 2>/dev/null | cut -d= -f2 | xargs || echo deploy)"
  # fall through to build section
}

do_export_backup() {
  local TIMESTAMP; TIMESTAMP=$(date +%Y%m%d_%H%M%S)
  local BACKUP_DIR="/opt/sentinelcore-backup"
  local BACKUP_FILE="${BACKUP_DIR}/sentinelcore_backup_${TIMESTAMP}.tar.gz"

  mkdir -p "$BACKUP_DIR"

  # Stop service briefly for DB consistency
  local WAS_RUNNING=false
  if systemctl is-active sentinelcore &>/dev/null 2>&1; then
    WAS_RUNNING=true
    systemctl stop sentinelcore
    info "Service stopped for backup."
  fi

  tar czf "$BACKUP_FILE" -C /opt \
    --exclude='sentinelcore/sentinelcore' \
    --exclude='sentinelcore/frontend' \
    sentinelcore 2>/dev/null || true

  if [[ "$WAS_RUNNING" == "true" ]]; then
    systemctl start sentinelcore
    info "Service restarted."
  fi

  chmod 600 "$BACKUP_FILE"
  success "Backup saved: ${BACKUP_FILE}"
  echo -e "  ${CYAN}Size: $(du -sh "$BACKUP_FILE" | cut -f1)${NC}"
}

do_import_backup() {
  echo -e "\n${BOLD}─── Import Backup ──────────────────────────────────────────────────${NC}\n"

  # List available backups
  local BACKUP_DIR="/opt/sentinelcore-backup"
  local BACKUP_FILE=""
  if [[ ! -d "$BACKUP_DIR" ]] || [[ -z "$(ls "$BACKUP_DIR"/*.tar.gz 2>/dev/null)" ]]; then
    warn "No backups found in ${BACKUP_DIR}."
    local CUSTOM_PATH
    prompt CUSTOM_PATH "Enter full path to backup file" ""
    [[ -n "$CUSTOM_PATH" ]] && BACKUP_FILE="$CUSTOM_PATH" || return
  else
    echo -e "Available backups:\n"
    local i=1
    declare -A BACKUP_MAP
    for f in "$BACKUP_DIR"/*.tar.gz; do
      echo -e "  ${GREEN}${i})${NC} $(basename "$f")  ${CYAN}($(du -sh "$f" | cut -f1))${NC}"
      BACKUP_MAP[$i]="$f"
      ((i++))
    done
    echo ""
    local choice
    read -r -p "$(echo -e "${BOLD}Choose backup number${NC}: ")" choice
    BACKUP_FILE="${BACKUP_MAP[$choice]:-}"
    [[ -z "$BACKUP_FILE" ]] && { warn "Invalid choice."; return; }
  fi

  [[ -f "$BACKUP_FILE" ]] || { error "File not found: $BACKUP_FILE"; return; }

  read -r -p "$(echo -e "${BOLD}Restore from ${BACKUP_FILE}? This will overwrite current data.${NC} (y/${BOLD}N${NC}): ")" confirm
  [[ "${confirm,,}" == "y" || "${confirm,,}" == "yes" ]] || { info "Import cancelled."; return; }

  # Stop service
  local WAS_RUNNING=false
  if systemctl is-active sentinelcore &>/dev/null 2>&1; then
    WAS_RUNNING=true
    systemctl stop sentinelcore
  fi

  tar xzf "$BACKUP_FILE" -C /opt
  chmod 600 "${INSTALL_DIR}/.env" 2>/dev/null || true

  if [[ "$WAS_RUNNING" == "true" ]]; then
    systemctl start sentinelcore
    success "Service restarted with restored data."
  fi

  success "Backup restored from: $BACKUP_FILE"
}

do_rotate_master_key() {
  echo -e "\n${BOLD}─── Rotate Master Key ──────────────────────────────────────────────${NC}\n"
  local key_path
  key_path=$(read_env SECRETS_KEY_PATH)
  [[ -z "$key_path" ]] && key_path="${INSTALL_DIR}/data/.master.key"

  warn "This will generate a fresh master key and re-encrypt protected values in the database."
  echo -e "  Protected values currently include encrypted TOTP secrets and SMTP credentials."
  echo -e "  Key file: ${CYAN}${key_path}${NC}\n"

  local backup_first
  read -r -p "$(echo -e "${BOLD}Create a backup before rotating the master key?${NC} (${BOLD}Y${NC}/n): ")" backup_first
  if [[ "${backup_first,,}" != "n" ]]; then
    do_export_backup || { warn "Backup failed; master key rotation aborted."; return; }
    echo ""
  fi

  read -r -p "$(echo -e "${BOLD}Rotate the master key now?${NC} (y/${BOLD}N${NC}): ")" confirm
  [[ "${confirm,,}" == "y" || "${confirm,,}" == "yes" ]] || { info "Master key rotation cancelled."; return; }

  if run_admin rotate-master-key; then
    chown "root:${SERVICE_USER:-$(grep '^User=' /etc/systemd/system/sentinelcore.service 2>/dev/null | cut -d= -f2 | xargs || echo deploy)}" "$key_path" 2>/dev/null || true
    chmod 640 "$key_path" 2>/dev/null || true
    success "Master key rotation completed successfully."
  fi
}

###############################################################################
# Main sudoers installation — comprehensive app installation permissions
# Usage: install_main_sudoers [user]
# Installs/updates sentinelcore.sudoers with latest permissions for app installations
###############################################################################
install_main_sudoers() {
  local TARGET_USER="${1:-$SVC_USER}"
  local SUDOERS_FILE="/etc/sudoers.d/sentinelcore"
  local SRC_SUDOERS="${SCRIPT_DIR}/deploy/sentinelcore.sudoers"
  
  if [[ -z "$TARGET_USER" ]]; then
    warn "No service user specified — skipping main sudoers installation"
    return 1
  fi
  
  if [[ ! -f "$SRC_SUDOERS" ]]; then
    warn "sentinelcore.sudoers not found at $SRC_SUDOERS"
    return 1
  fi
  
  info "Installing main sudoers for user: ${CYAN}${TARGET_USER}${NC}"
  
  # Backup existing sudoers if it exists
  if [[ -f "$SUDOERS_FILE" ]]; then
    cp "$SUDOERS_FILE" "${SUDOERS_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    info "Backed up existing sudoers to ${SUDOERS_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
  fi
  
  # Replace <USER> placeholder with actual service user
  sed "s/<USER>/$TARGET_USER/g" "$SRC_SUDOERS" > "$SUDOERS_FILE"
  chmod 440 "$SUDOERS_FILE"
  
  if visudo -c -f "$SUDOERS_FILE" &>/dev/null; then
    success "Installed: $SUDOERS_FILE with latest app installation permissions"
    info "Sudoers updated — if you changed sentinelcore.sudoers, restart the service so new sudo rules are applied."
    return 0
  else
    rm -f "$SUDOERS_FILE"
    # Restore backup if it existed
    if [[ -f "${SUDOERS_FILE}.backup.$(date +%Y%m%d_%H%M%S)" ]]; then
      mv "${SUDOERS_FILE}.backup.$(date +%Y%m%d_%H%M%S)" "$SUDOERS_FILE"
    fi
    warn "visudo validation failed — main sudoers installation skipped"
    return 1
  fi
}

###############################################################################
# Sudoers helper — grant NOPASSWD ufw to one or more users
# Usage: configure_sudoers_ufw [user1 user2 ...]
# With no args: uses $ADMIN_USER + service user from existing systemd unit.
###############################################################################
configure_sudoers_ufw() {
  local SUDOERS_FILE="/etc/sudoers.d/sentinelcore-ufw"
  local UFW_BIN; UFW_BIN=$(find_ufw_bin)

  if [[ -z "$UFW_BIN" ]]; then
    warn "ufw binary not found — skipping sudoers entry."
    return
  fi

  # Build user list from args or fall back to ADMIN_USER + service unit
  local RAW_USERS=()
  if [[ $# -gt 0 ]]; then
    RAW_USERS=("$@")
  else
    [[ -n "${ADMIN_USER:-}" ]] && RAW_USERS+=("${ADMIN_USER}")
    local _svc_file="${SERVICE_FILE:-/etc/systemd/system/sentinelcore.service}"
    local _svc_user
    _svc_user=$(grep "^User=" "$_svc_file" 2>/dev/null | cut -d= -f2 | xargs || true)
    if [[ -n "$_svc_user" && "$_svc_user" != "root" && "$_svc_user" != "${ADMIN_USER:-}" ]]; then
      RAW_USERS+=("$_svc_user")
    fi
  fi

  # Deduplicate, drop root
  local USERS=()
  declare -A _SEEN=()
  for U in "${RAW_USERS[@]}"; do
    [[ "$U" == "root" ]] && continue
    [[ -n "${_SEEN[$U]:-}" ]] && continue
    _SEEN[$U]=1
    USERS+=("$U")
  done

  if [[ ${#USERS[@]} -eq 0 ]]; then
    warn "No non-root users specified — sudoers entry skipped."
    return
  fi

  {
    echo "# SentinelCore — password-less ufw for firewall-rule reads"
    echo "# Written by deploy-sentinel.sh on $(date +%Y-%m-%d)"
    for U in "${USERS[@]}"; do
      echo "${U} ALL=(ALL) NOPASSWD: ${UFW_BIN}"
    done
  } > "$SUDOERS_FILE"
  chmod 440 "$SUDOERS_FILE"

  if visudo -c -f "$SUDOERS_FILE" &>/dev/null; then
    for U in "${USERS[@]}"; do
      success "Sudoers: ${U} ALL=(ALL) NOPASSWD: ${UFW_BIN}"
    done
  else
    rm -f "$SUDOERS_FILE"
    warn "visudo validation failed — sudoers entry skipped (ufw will fall back to direct call)."
  fi
}

###############################################################################
# Complete sudoers setup — installs both main and ufw sudoers
# Usage: setup_all_sudoers [user]
###############################################################################
setup_all_sudoers() {
  local TARGET_USER="${1:-$SVC_USER}"
  section "Sudoers Configuration"
  
  # Install main sudoers first (app installation permissions)
  install_main_sudoers "$TARGET_USER"
  
  # Install UFW sudoers (firewall permissions)
  configure_sudoers_ufw "$TARGET_USER"
  
  echo ""
}

###############################################################################
# Fix Sentinel Permissions
# Detects service user, adds groups, fixes ownership, updates systemd unit.
###############################################################################
do_fix_permissions() {
  section "Fix Sentinel Permissions"

  [[ -f "${INSTALL_DIR}/sentinelcore" ]] || {
    warn "SentinelCore is not installed at ${INSTALL_DIR}."
    return 1
  }

  # ── Detect current service user ─────────────────────────────────────────────
  local SVC_FILE="/etc/systemd/system/sentinelcore.service"
  local SVC_USER=""
  if [[ -f "$SVC_FILE" ]]; then
    SVC_USER=$(grep "^User=" "$SVC_FILE" 2>/dev/null | cut -d= -f2 | xargs || true)
  fi

  local DEFAULT_USER="deploy"

  if [[ -z "$SVC_USER" || "$SVC_USER" == "root" ]]; then
    info "Service is currently running as root (or unit not found)."
    prompt SVC_USER "Service user to run the SentinelCore binary as" "$DEFAULT_USER"
  else
    echo -e "  Detected service user: ${CYAN}${SVC_USER}${NC}"
    read -r -p "$(echo -e "${BOLD}Use this user?${NC} (${BOLD}Y${NC}/n): ")" _USE
    if [[ "${_USE,,}" == "n" ]]; then
      prompt SVC_USER "Service user" "$SVC_USER"
    fi
  fi

  if ! id "$SVC_USER" &>/dev/null; then
    warn "User '${SVC_USER}' does not exist."
    warn "Create it first:  useradd -m -s /bin/bash -G sudo ${SVC_USER}"
    return 1
  fi

  info "Applying permissions for user: ${CYAN}${SVC_USER}${NC}"
  echo ""

  # ── Add to required groups ───────────────────────────────────────────────────
  section "Group membership"
  local GROUPS_TO_ADD=("adm" "systemd-journal")
  getent group docker &>/dev/null && GROUPS_TO_ADD+=("docker")

  for grp in "${GROUPS_TO_ADD[@]}"; do
    if getent group "$grp" &>/dev/null; then
      if usermod -aG "$grp" "$SVC_USER" 2>/dev/null; then
        success "Added ${SVC_USER} → group ${grp}"
      else
        warn "Could not add ${SVC_USER} to group ${grp}"
      fi
    else
      info "Group '${grp}' not present — skipped."
    fi
  done
  echo ""

  # ── Directory ownership & permissions ───────────────────────────────────────
  section "Ownership & permissions"
  chown -R "${SVC_USER}:${SVC_USER}" "$INSTALL_DIR"
  success "chown -R ${SVC_USER}:${SVC_USER} ${INSTALL_DIR}"

  chmod 700 "${INSTALL_DIR}/data"
  success "chmod 700 ${INSTALL_DIR}/data"

  if [[ -f "${INSTALL_DIR}/.env" ]]; then
    chmod 600 "${INSTALL_DIR}/.env"
    success "chmod 600 ${INSTALL_DIR}/.env"
  fi

  if [[ -f "${INSTALL_DIR}/data/.master.key" ]]; then
    chown "root:${SVC_USER}" "${INSTALL_DIR}/data/.master.key" 2>/dev/null || true
    chmod 640 "${INSTALL_DIR}/data/.master.key"
    success "secured ${INSTALL_DIR}/data/.master.key (root:${SVC_USER}, 640)"
  fi
  echo ""

  # ── Sudoers ─────────────────────────────────────────────────────────────────
  setup_all_sudoers "$SVC_USER"

  # ── Update systemd unit ─────────────────────────────────────────────────────
  section "Systemd unit"
  if [[ -f "$SVC_FILE" ]]; then
    local SUPP_GROUPS="adm systemd-journal"
    getent group docker &>/dev/null && SUPP_GROUPS="adm systemd-journal docker"

    # User=
    sed -i "s/^User=.*/User=${SVC_USER}/" "$SVC_FILE"
    success "Updated: User=${SVC_USER}"

    # SupplementaryGroups=
    if grep -q "^SupplementaryGroups=" "$SVC_FILE"; then
      sed -i "s|^SupplementaryGroups=.*|SupplementaryGroups=${SUPP_GROUPS}|" "$SVC_FILE"
    else
      sed -i "/^User=.*/a SupplementaryGroups=${SUPP_GROUPS}" "$SVC_FILE"
    fi
    success "Updated: SupplementaryGroups=${SUPP_GROUPS}"

    # Ensure explicit PATH is present
    if ! grep -q "^Environment=PATH=" "$SVC_FILE"; then
      sed -i "/^EnvironmentFile=.*/a Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" "$SVC_FILE"
      success "Added: explicit PATH to systemd unit"
    fi

    systemctl daemon-reload
    success "systemctl daemon-reload"
    echo ""

    read -r -p "$(echo -e "${BOLD}Restart sentinelcore service now?${NC} (${BOLD}Y${NC}/n): ")" _RST
    if [[ "${_RST,,}" != "n" ]]; then
      systemctl restart sentinelcore
      sleep 2
      if systemctl is-active sentinelcore &>/dev/null; then
        success "Service restarted and is running."
      else
        warn "Service may have failed. Check: journalctl -u sentinelcore -n 30"
      fi
    fi
  else
    warn "Systemd unit not found at ${SVC_FILE} — unit not updated."
  fi

  echo ""
  warn "Group changes for interactive shells take effect on next login."
  warn "The service uses new groups immediately after restart."
  success "Permissions fix complete."
}

###############################################################################
# Banner
###############################################################################
_SC_HOST="$(hostname 2>/dev/null || echo unknown)"
_SC_DATE="$(date '+%Y-%m-%d %H:%M')"
echo -e "
${CYAN}╔══════════════════════════════════════════════════════════════╗
║      SENTINELCORE  ─  Deployment & Maintenance               ║
║      Self-hosted Linux Security Dashboard  ·  v${SCRIPT_VERSION}          ║
╠══════════════════════════════════════════════════════════════╣
║  Host  : $(printf '%-51s' "${_SC_HOST}")║
║  Date  : $(printf '%-51s' "${_SC_DATE}")║
╚══════════════════════════════════════════════════════════════╝${NC}
"

###############################################################################
# OS + root check
###############################################################################
[[ "$OSTYPE" == "linux-gnu"* ]] || error "This script requires Linux."
[[ "$EUID" -eq 0 ]] || error "Run as root (use sudo)."

###############################################################################
# Existing installation detection
###############################################################################
IS_REDEPLOY=false
MENU_CHOICE=""

if [[ -f "${INSTALL_DIR}/sentinelcore" && -f "${INSTALL_DIR}/.env" ]]; then
  IS_REDEPLOY=true
  SVC_STATUS=$(systemctl is-active sentinelcore 2>/dev/null || echo "not installed")
  CURRENT_LISTEN=$(read_env LISTEN_ADDR)
  CURRENT_PORT="${CURRENT_LISTEN##*:}"
  SVC_USER_DISPLAY=$(grep "^User=" /etc/systemd/system/sentinelcore.service 2>/dev/null | cut -d= -f2 || echo "root")

  if [[ "$SVC_STATUS" == "active" ]]; then
    _SVC_COLOR="${GREEN}"
  else
    _SVC_COLOR="${YELLOW}"
  fi

  echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗
║      SentinelCore is already installed                       ║
╚══════════════════════════════════════════════════════════════╝${NC}

  ${BOLD}Install dir${NC}   : ${CYAN}${INSTALL_DIR}${NC}
  ${BOLD}Service user${NC}  : ${CYAN}${SVC_USER_DISPLAY}${NC}
  ${BOLD}Service${NC}       : ${_SVC_COLOR}${SVC_STATUS}${NC}
  ${BOLD}Listen${NC}        : ${CYAN}${CURRENT_LISTEN}${NC}
"
  echo -e "${BOLD}What would you like to do?${NC}\n"
  echo -e "  ${GREEN}1)${NC} Redeploy / Upgrade    ${DIM}(rebuild + reinstall, keeps DB & settings)${NC}"
  echo -e "  ${GREEN}2)${NC} Reset a user password"
  echo -e "  ${GREEN}3)${NC} Remove 2FA from a user"
  echo -e "  ${GREEN}4)${NC} Manage UFW port        ${DIM}(current: ${CURRENT_PORT})${NC}"
  echo -e "  ${GREEN}5)${NC} Uninstall SentinelCore"
  echo -e "  ${GREEN}6)${NC} Rebuild binary & UI   ${DIM}(same config, no prompts)${NC}"
  echo -e "  ${GREEN}7)${NC} Export backup          ${DIM}(saves .env + DB → /opt/sentinelcore-backup/)${NC}"
  echo -e "  ${GREEN}8)${NC} Import backup          ${DIM}(restores .env + DB from a backup file)${NC}"
  echo -e "  ${GREEN}9)${NC} Fix permissions        ${DIM}(groups, ownership, sudoers, systemd user, app install perms)${NC}"
  echo -e "  ${GREEN}10)${NC} Rotate master key      ${DIM}(re-encrypt protected DB values safely)${NC}"
  echo -e "  ${GREEN}0)${NC} Exit\n"

  read -r -p "$(echo -e "${BOLD}Choice [0-10]${NC}: ")" MENU_CHOICE
  case "$MENU_CHOICE" in
    1) info "Proceeding with redeploy..." ;;
    2) do_reset_password;  exit 0 ;;
    3) do_reset_2fa;       exit 0 ;;
    4) do_manage_ufw;      exit 0 ;;
    5) do_uninstall;       exit 0 ;;
    6) do_rebuild ;;
    7) do_export_backup;   exit 0 ;;
    8) do_import_backup;   exit 0 ;;
    9) do_fix_permissions; exit 0 ;;
    10) do_rotate_master_key; exit 0 ;;
    0) info "Exiting.";    exit 0 ;;
    *) error "Invalid choice." ;;
  esac
fi

# If rebuild was chosen, skip all prompts — read everything from existing .env
if [[ "$MENU_CHOICE" == "6" ]]; then
  IS_REDEPLOY=true
  # Read all config from existing .env silently
  ADMIN_USER=$(read_env INITIAL_ADMIN_USERNAME)
  ADMIN_PASS=""
  BOOTSTRAP_ADMIN_FILE=$(read_env INITIAL_ADMIN_PASSWORD_FILE)
  if [[ -n "$BOOTSTRAP_ADMIN_FILE" && -f "$BOOTSTRAP_ADMIN_FILE" ]]; then
    ADMIN_PASS=$(tr -d '\r\n' < "$BOOTSTRAP_ADMIN_FILE" 2>/dev/null || true)
  fi
  SERVICE_USER=$(grep "^User=" /etc/systemd/system/sentinelcore.service 2>/dev/null | cut -d= -f2 | xargs || true)
  [[ -z "$SERVICE_USER" ]] && SERVICE_USER="deploy"
  SECRET_PATH=$(read_env SECRET_PATH)
  CURRENT_LISTEN=$(read_env LISTEN_ADDR)
  PORT="${CURRENT_LISTEN##*:}"
  PUBLIC_ACCESS="n"
  [[ "$CURRENT_LISTEN" == 0.0.0.0* ]] && PUBLIC_ACCESS="y"
  LISTEN_ADDR="$CURRENT_LISTEN"
  ALERT_EMAIL=$(read_env ALERT_EMAIL)
  SMTP_HOST=$(read_env SMTP_HOST)
  SMTP_PORT=$(read_env SMTP_PORT)
  SMTP_USER=$(read_env SMTP_USER)
  SMTP_PASS=""
  BOOTSTRAP_SMTP_FILE=$(read_env SMTP_PASS_FILE)
  if [[ -n "$BOOTSTRAP_SMTP_FILE" && -f "$BOOTSTRAP_SMTP_FILE" ]]; then
    SMTP_PASS=$(tr -d '\r\n' < "$BOOTSTRAP_SMTP_FILE" 2>/dev/null || true)
  fi
  INSTALL_SERVICE="y"
  info "Config loaded from existing .env — rebuilding without prompts."
fi

###############################################################################
# Install Go
###############################################################################
install_go() {
  info "Installing Go 1.23..."
  local GO_VER="1.23.4"
  local ARCH; ARCH=$(uname -m)
  local GOARCH
  case "$ARCH" in
    x86_64)  GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    *)       error "Unsupported arch: $ARCH" ;;
  esac
  local URL="https://go.dev/dl/go${GO_VER}.linux-${GOARCH}.tar.gz"
  info "Downloading $URL"
  curl -fsSL "$URL" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  export PATH="$PATH:/usr/local/go/bin"
  success "Go $(go version) installed."
}

if ! command -v go &>/dev/null; then
  install_go
else
  GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  info "Go ${GO_VERSION} already installed."
fi

###############################################################################
# Install Node.js + npm  (distro-aware)
###############################################################################
install_node() {
  info "Installing Node.js 20 LTS..."
  local family; family=$(detect_distro_family)
  case "$family" in
    debian)
      curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
      apt-get install -y nodejs
      ;;
    rhel)
      curl -fsSL https://rpm.nodesource.com/setup_20.x | bash -
      if command -v dnf &>/dev/null; then
        dnf install -y nodejs
      else
        yum install -y nodejs
      fi
      ;;
    arch)
      pacman -Sy --noconfirm nodejs npm
      ;;
    suse)
      zypper install -y nodejs20 npm20
      ;;
    *)
      error "Cannot install Node.js automatically on distro '$(detect_distro)'. Install Node.js 20 manually then re-run."
      ;;
  esac
  success "Node.js $(node -v) / npm $(npm -v) installed."
}

if ! command -v node &>/dev/null; then
  install_node
else
  info "Node.js $(node -v) already installed."
fi

###############################################################################
# Interactive prompts (skipped for rebuild — variables already loaded from .env)
###############################################################################
if [[ "${MENU_CHOICE:-}" != "6" ]]; then
  echo -e "\n${BOLD}─── Configuration ─────────────────────────────────────────────────${NC}\n"

  # Use existing .env values as defaults when redeploying
  if [[ "$IS_REDEPLOY" == "true" ]]; then
    DEF_ADMIN_USER=$(read_env INITIAL_ADMIN_USERNAME)
    DEF_SERVICE_USER=$(grep "^User=" /etc/systemd/system/sentinelcore.service 2>/dev/null | cut -d= -f2 | xargs || true)
    [[ -z "$DEF_SERVICE_USER" ]] && DEF_SERVICE_USER="deploy"
    DEF_SECRET_PATH=$(read_env SECRET_PATH)
    DEF_PORT="$CURRENT_PORT"
    DEF_PUBLIC="n"
    [[ "$CURRENT_LISTEN" == 0.0.0.0* ]] && DEF_PUBLIC="y"
  else
    DEF_ADMIN_USER="admin"
    DEF_SERVICE_USER="deploy"
    DEF_SECRET_PATH="sentinel-core"
    DEF_PORT="8080"
    DEF_PUBLIC="n"
  fi

  prompt ADMIN_USER  "Admin username"              "$DEF_ADMIN_USER"
  prompt SERVICE_USER "Linux service user"          "$DEF_SERVICE_USER"

  if [[ "$IS_REDEPLOY" == "true" ]]; then
    warn "To change the admin password, use option 2 (Reset user password) from the maintenance menu."
    ADMIN_PASS=""
  else
    promptpw ADMIN_PASS "Admin password (min 12 chars)"
  fi

  prompt SECRET_PATH "Secret URL path (no slashes)" "$DEF_SECRET_PATH"
  prompt PORT        "Listen port"                  "$DEF_PORT"

  PUBLIC_ACCESS="$DEF_PUBLIC"
  read -r -p "$(echo -e "${BOLD}Allow PUBLIC internet access?${NC} [current: ${CYAN}${DEF_PUBLIC}${NC}] (y/N): ")" PA
  [[ "${PA,,}" == "y" || "${PA,,}" == "yes" ]] && PUBLIC_ACCESS="y"
  [[ "${PA,,}" == "n" || "${PA,,}" == "no"  ]] && PUBLIC_ACCESS="n"

  if [[ "$PUBLIC_ACCESS" == "y" ]]; then
    LISTEN_ADDR="0.0.0.0:${PORT}"
    warn "Public access enabled. Ensure your firewall is properly configured."
  else
    LISTEN_ADDR="127.0.0.1:${PORT}"
    info "Localhost-only mode. Access via SSH tunnel: ssh -L ${PORT}:127.0.0.1:${PORT} user@server"
  fi

  echo ""
  prompt ALERT_EMAIL "Alert email (optional, leave blank to skip)" "$(read_env ALERT_EMAIL)"
  SMTP_HOST="" SMTP_PORT="587" SMTP_USER="" SMTP_PASS=""
  if [[ -n "$ALERT_EMAIL" ]]; then
    prompt SMTP_HOST "SMTP host"     "$(read_env SMTP_HOST 2>/dev/null || echo smtp.gmail.com)"
    prompt SMTP_PORT "SMTP port"     "$(read_env SMTP_PORT 2>/dev/null || echo 587)"
    prompt SMTP_USER "SMTP username" "${ALERT_EMAIL}"
    read -r -s -p "$(echo -e "${BOLD}SMTP password${NC} (leave blank to keep existing): ")" SMTP_PASS; echo
    # If blank on redeploy, keep existing encrypted DB value by not creating a bootstrap file.
  fi

  INSTALL_SERVICE="y"
  read -r -p "$(echo -e "${BOLD}Install as systemd service?${NC} (${BOLD}Y${NC}/n): ")" IS
  [[ "${IS,,}" == "n" ]] && INSTALL_SERVICE="n"
fi

###############################################################################
# Build frontend
###############################################################################
echo -e "\n${BOLD}─── Building Frontend ──────────────────────────────────────────────${NC}\n"
FRONTEND_DIR="${SCRIPT_DIR}/frontend"
[[ -d "$FRONTEND_DIR" ]] || error "frontend/ directory not found in $SCRIPT_DIR"

cd "$FRONTEND_DIR"
info "Installing npm dependencies..."
npm install --legacy-peer-deps --loglevel=error
info "Building Vue frontend..."
npm run build
success "Frontend built → ${FRONTEND_DIR}/dist"

###############################################################################
# Build backend
###############################################################################
echo -e "\n${BOLD}─── Building Backend ───────────────────────────────────────────────${NC}\n"
BACKEND_DIR="${SCRIPT_DIR}/backend"
[[ -d "$BACKEND_DIR" ]] || error "backend/ directory not found in $SCRIPT_DIR"

cd "$BACKEND_DIR"
export PATH="$PATH:/usr/local/go/bin"
info "Downloading Go dependencies..."
go mod tidy
info "Compiling sentinelcore binary..."
go build -ldflags="-s -w" -o sentinelcore ./cmd/sentinelcore/
success "Binary built → ${BACKEND_DIR}/sentinelcore"

###############################################################################
# Install to /opt/sentinelcore
###############################################################################
echo -e "\n${BOLD}─── Installing ─────────────────────────────────────────────────────${NC}\n"

# Stop service before replacing binary
if systemctl is-active sentinelcore &>/dev/null 2>&1; then
  systemctl stop sentinelcore
  info "Stopped existing service."
fi

mkdir -p "${INSTALL_DIR}/data"
cp "${BACKEND_DIR}/sentinelcore" "${INSTALL_DIR}/sentinelcore"
rm -rf "${INSTALL_DIR}/frontend"
cp -r "${FRONTEND_DIR}/dist" "${INSTALL_DIR}/frontend"
chmod +x "${INSTALL_DIR}/sentinelcore"
success "Files copied to ${INSTALL_DIR}"

if ! id "${SERVICE_USER:-deploy}" &>/dev/null; then
  warn "Linux user '${SERVICE_USER:-deploy}' does not exist; creating it."
  useradd -m -s /bin/bash "${SERVICE_USER:-deploy}" || error "Failed to create service user '${SERVICE_USER:-deploy}'."
fi

SECRETS_KEY_PATH="${INSTALL_DIR}/data/.master.key"
if [[ ! -f "$SECRETS_KEY_PATH" ]]; then
  info "Generating DB secrets master key..."
  if command -v openssl &>/dev/null; then
    openssl rand -base64 32 > "$SECRETS_KEY_PATH"
  else
    tr -dc 'A-Za-z0-9+/=' </dev/urandom | head -c 44 > "$SECRETS_KEY_PATH"
    echo >> "$SECRETS_KEY_PATH"
  fi
fi

# Keep master key root-owned but readable by service group only.
chown "root:${SERVICE_USER:-deploy}" "$SECRETS_KEY_PATH" 2>/dev/null || true
chmod 640 "$SECRETS_KEY_PATH"

INITIAL_ADMIN_PASS_FILE="${INSTALL_DIR}/data/.bootstrap_admin_pass"
if [[ -n "${ADMIN_PASS:-}" ]]; then
  printf "%s\n" "$ADMIN_PASS" > "$INITIAL_ADMIN_PASS_FILE"
  chmod 600 "$INITIAL_ADMIN_PASS_FILE"
  chown "${SERVICE_USER:-deploy}:${SERVICE_USER:-deploy}" "$INITIAL_ADMIN_PASS_FILE" 2>/dev/null || true
else
  rm -f "$INITIAL_ADMIN_PASS_FILE" 2>/dev/null || true
  INITIAL_ADMIN_PASS_FILE=""
fi

SMTP_PASS_FILE="${INSTALL_DIR}/data/.bootstrap_smtp_pass"
if [[ -n "${SMTP_PASS:-}" ]]; then
  printf "%s\n" "$SMTP_PASS" > "$SMTP_PASS_FILE"
  chmod 600 "$SMTP_PASS_FILE"
  chown "${SERVICE_USER:-deploy}:${SERVICE_USER:-deploy}" "$SMTP_PASS_FILE" 2>/dev/null || true
else
  rm -f "$SMTP_PASS_FILE" 2>/dev/null || true
  SMTP_PASS_FILE=""
fi

# Ensure service user owns install dir before service starts
chown -R "${SERVICE_USER:-deploy}:${SERVICE_USER:-deploy}" "$INSTALL_DIR" 2>/dev/null || \
  warn "Could not chown ${INSTALL_DIR} — run option 9 (Fix Permissions) after install."
chmod 700 "${INSTALL_DIR}/data"
# Re-assert master key ownership/permissions after recursive chown.
chown "root:${SERVICE_USER:-deploy}" "$SECRETS_KEY_PATH" 2>/dev/null || true
chmod 640 "$SECRETS_KEY_PATH"

###############################################################################
# JWT secret — preserve existing on redeploy to keep sessions valid
###############################################################################
EXISTING_JWT=$(read_env JWT_SECRET 2>/dev/null || true)
if [[ -n "$EXISTING_JWT" ]]; then
  JWT_SECRET="$EXISTING_JWT"
  info "Reusing existing JWT secret (active sessions remain valid)."
else
  JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || tr -dc 'a-f0-9' </dev/urandom | head -c 64)
fi

###############################################################################
# Write .env
###############################################################################
ENV_FILE="${INSTALL_DIR}/.env"
cat > "$ENV_FILE" << ENVEOF
LISTEN_ADDR=${LISTEN_ADDR}
FRONTEND_DIR=${INSTALL_DIR}/frontend
JWT_SECRET=${JWT_SECRET}
DB_PATH=${INSTALL_DIR}/data/app.db
SECRET_PATH=${SECRET_PATH}
LOG_LEVEL=info
METRICS_INTERVAL=2
SECRETS_KEY_PATH=${SECRETS_KEY_PATH}

INITIAL_ADMIN_USERNAME=${ADMIN_USER}
INITIAL_ADMIN_PASSWORD_FILE=${INITIAL_ADMIN_PASS_FILE}
TERMINAL_RUN_AS_USER=${SERVICE_USER}

SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASS_FILE=${SMTP_PASS_FILE}
ALERT_EMAIL=${ALERT_EMAIL}

BRUTE_FORCE_THRESHOLD=5
ENVEOF
chmod 600 "$ENV_FILE"
success ".env written to ${ENV_FILE}"

###############################################################################
# Systemd service
###############################################################################
if [[ "$INSTALL_SERVICE" == "y" ]]; then
  SERVICE_FILE="/etc/systemd/system/sentinelcore.service"
  cat > "$SERVICE_FILE" << SVCEOF
[Unit]
Description=SentinelCore Security Dashboard
Documentation=https://github.com/ehsanR91/sentinelcore
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/sentinelcore
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
# Explicit PATH so that ufw/ss/journalctl are always found
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
Restart=always
RestartSec=5
User=${SERVICE_USER:-deploy}
SupplementaryGroups=adm systemd-journal
# NoNewPrivileges intentionally omitted — the backend calls sudo -n via sudoers.
# ProtectSystem is intentionally absent: the backend installs/removes system packages
# (apt/dpkg) which must write to /usr/bin and /etc — incompatible with any
# ProtectSystem level. Security boundary is the unprivileged user + scoped sudoers.

[Install]
WantedBy=multi-user.target
SVCEOF
  systemctl daemon-reload
  systemctl enable sentinelcore
  systemctl restart sentinelcore
  success "sentinelcore.service enabled and started."
  sleep 2
  systemctl is-active sentinelcore &>/dev/null \
    && success "Service is running." \
    || warn "Service may have failed — check: journalctl -u sentinelcore -n 30"
else
  info "To run manually:"
  echo -e "  ${CYAN}cd ${INSTALL_DIR} && env \$(cat .env | xargs) ./sentinelcore${NC}"
fi

###############################################################################
# Sudoers — called here during install/redeploy (function defined above)
###############################################################################
setup_all_sudoers "${SERVICE_USER:-deploy}"

###############################################################################
# UFW — open port automatically if public access is enabled
###############################################################################
_UFW_BIN=$(find_ufw_bin)
_FW_TYPE=$(detect_firewall)
if [[ -n "$_UFW_BIN" ]]; then
  if [[ "$PUBLIC_ACCESS" == "y" ]]; then
    # Close old port if port changed during redeploy
    if [[ "$IS_REDEPLOY" == "true" && -n "${CURRENT_PORT:-}" && "${CURRENT_PORT:-}" != "$PORT" ]]; then
      ufw_close "$CURRENT_PORT"
      info "Closed old port ${CURRENT_PORT} in UFW."
    fi
    ufw_open "$PORT"
  else
    # If switching from public to private, close the port
    if [[ "$IS_REDEPLOY" == "true" && -n "${CURRENT_PORT:-}" ]] && ufw_port_open "${CURRENT_PORT:-}"; then
      read -r -p "$(echo -e "${BOLD}Port ${CURRENT_PORT} is open in UFW. Close it now?${NC} (${BOLD}Y${NC}/n): ")" CLOSE_PORT
      [[ "${CLOSE_PORT,,}" != "n" ]] && ufw_close "$CURRENT_PORT"
    fi
  fi
else
  if [[ "$PUBLIC_ACCESS" == "y" ]]; then
    case "$_FW_TYPE" in
      firewalld) warn "firewalld detected — open port ${PORT} manually: firewall-cmd --permanent --add-port=${PORT}/tcp && firewall-cmd --reload" ;;
      nftables)  nft_open "${PORT}" || warn "nftables detected — open port ${PORT} manually: nft add rule inet filter input tcp dport ${PORT} accept" ;;
      iptables)  warn "iptables detected — open port ${PORT} manually: iptables -A INPUT -p tcp --dport ${PORT} -j ACCEPT" ;;
      *)         warn "No supported firewall found — open port ${PORT} in your firewall manually." ;;
    esac
  fi
fi

###############################################################################
# Done
###############################################################################
echo -e "
${GREEN}╔══════════════════════════════════════════════════════════════╗
║                    DEPLOYMENT COMPLETE                       ║
╚══════════════════════════════════════════════════════════════╝${NC}

${BOLD}Secret URL (bookmark this — required to access the login page):${NC}
  ${CYAN}http://<your-server-ip>:${PORT}/${SECRET_PATH}/${NC}
"

if [[ "$PUBLIC_ACCESS" != "y" ]]; then
  echo -e "${BOLD}SSH tunnel command (run on your local machine):${NC}
  ${CYAN}ssh -L ${PORT}:127.0.0.1:${PORT} <user>@<server-ip>
  Then open: http://localhost:${PORT}/${SECRET_PATH}/${NC}
"
fi

echo -e "${YELLOW}⚠  Security reminders:${NC}
  • Protect ${INSTALL_DIR}/data/.master.key like a production secret.
  • If an attacker gets BOTH the DB file and .master.key, encrypted values can be decrypted.
  • Clear your shell history:  history -c
  • The .env file at ${INSTALL_DIR}/.env is root-readable only (chmod 600).
  • After first login, go to Settings → 2FA and enable TOTP authentication.
  • Re-run this script at any time for maintenance (password reset, 2FA removal, UFW, uninstall).
"
