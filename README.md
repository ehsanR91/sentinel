# SentinelCore - OpenSource Ubuntu/Debian Server Control Panel

SentinelCore is a self-hosted Linux security and operations dashboard for hardened VPS and Docker hosts.

It combines monitoring, firewall management, alerts, audit logs, user management, secure terminal access, and operational controls behind a single web UI with role-based access control and 2FA support.

> Author: <https://github.com/ehsanR91>

---

## 🚀 Quick Install

SentinelCore is built for Linux servers and VPS hosts.
Use the deploy script to install and configure the stack safely.

### Server prerequisites

- Debian / Ubuntu:

  ```bash
  sudo apt update && sudo apt install -y curl git bash
  ```

- CentOS / RHEL / AlmaLinux / Amazon Linux:

  ```bash
  sudo yum install -y curl git bash
  ```

- Fedora:

  ```bash
  sudo dnf install -y curl git bash
  ```

### Clone and deploy

```bash
git clone https://github.com/ehsanR91/sentinel.git
cd sentinelcore
sudo bash deploy-sentinel.sh
```

### One-liner install

```bash
curl -fsSL https://raw.githubusercontent.com/ehsanR91/sentinel/main/deploy-sentinel.sh | sudo bash
```

### Alternative one-liner using tarball

```bash
curl -fsSL https://github.com/ehsanR91/sentinel/archive/refs/heads/main.tar.gz | tar xz
cd sentinel-main
sudo bash deploy-sentinel.sh
```

> If your server does not have `curl`, substitute `wget -qO-` for `curl -fsSL`.

---

## What It Does

SentinelCore is designed to replace a pile of separate tools with a single security-first control plane:

- Live host monitoring
- Security status and ban management
- Firewall inspection and rule changes
- Alerting and audit trails
- User management with RBAC
- TOTP 2FA
- Web terminal with strict risk gating
- Secure settings storage for selected secrets

---

## Project Layout

```text
sentinelcore/
|- backend/                 Go API, auth, DB, alerting, terminal, admin tools
|  |- cmd/sentinelcore/     Main server + admin subcommands
|  `- internal/
|- frontend/                Vue dashboard
|  `- src/
|- deploy-sentinel.sh       Interactive build/deploy/maintenance script
`- README.md
```

---

## Stack

| Layer | Technology |
|-------|------------|
| Backend | Go, chi, gorilla/websocket |
| Auth | JWT, Argon2id, TOTP 2FA |
| DB | SQLite (WAL mode) |
| Frontend | Vue, Vue Router, Vuex, Bootstrap |
| Charts | ApexCharts |
| Deployment | systemd service + static frontend |

---

## Main Features

| Area | Details |
|------|---------|
| Authentication | JWT login, TOTP 2FA, brute-force tracking |
| Roles | Viewer, Operator, Admin, Superadmin |
| Audit | Login attempts, terminal actions, privileged operations |
| Terminal | Audited WebSocket terminal with safe-command allowlist and 2FA-gated high-risk mode |
| Firewall | UFW inspection and management |
| Monitoring | Metrics, services, suspicious processes, alerts |
| Users | Create, edit, delete users; reset passwords; reset 2FA |
| Settings | Secret path, SMTP, alerts, thresholds |
| Maintenance | Backup, restore, permission repair, master-key rotation |

---

## Terminal Security Model

SentinelCore's terminal is intentionally not a raw unrestricted shell.

### Default mode

- Runs commands as a lower-privileged Linux service user, not root
- Allows only a small safe allowlist of commands without elevation
- Audits terminal connect, command execution, blocked commands, and high-risk unlock actions

### High-risk mode

- High-risk commands require explicit re-authentication with TOTP 2FA
- Unlock is temporary
- Unlock can be revoked manually
- Destructive patterns are permanently blocked

### Important note

SentinelCore is designed to reduce accidental or malicious damage, not to turn a browser terminal into a fully trusted root shell.

---

## Secrets and Encryption Model

This repository now uses a stronger secret handling model than plain `.env` storage.

### Passwords

User passwords are never stored in plaintext.

- Passwords are hashed with Argon2id
- Each hash includes its own salt and parameters
- These hashes are not reversible

### `.env` hardening

The deployment flow no longer keeps the initial admin password in plaintext inside `.env`.

Instead:

- `.env` stores `INITIAL_ADMIN_PASSWORD_FILE`, not the password itself
- The deploy script writes a one-time bootstrap password file
- The backend consumes it on first boot if needed
- The bootstrap file is deleted after use

The same pattern is used for SMTP credentials:

- `.env` stores `SMTP_PASS_FILE`
- The deploy script creates a one-time bootstrap file when needed
- The backend ingests it and stores the value encrypted in the DB
- The bootstrap file is deleted after use

### Database-encrypted secrets

Selected sensitive DB values are encrypted at rest using AES-GCM with a master key file.

Currently protected values include:

- User TOTP secrets
- Stored SMTP password

### Master key file

The encryption key is stored separately from the DB at:

`/opt/sentinelcore/data/.master.key`

This file is protected with strict permissions and is more sensitive than `.env`.

### Threat model

If an attacker gets only:

- `.env`: they should not gain your initial admin password or SMTP password
- DB file: encrypted protected values remain unreadable without the master key

If an attacker gets both:

- DB file
- `.master.key`

then encrypted at-rest values can be decrypted.

So the master key must be protected like a production secret.

### What is not covered

This is application-level field encryption, not full SQLite file encryption.

That means:

- The whole DB file is not opaque like SQLCipher would make it
- Only selected protected values are encrypted at rest

This is still a major improvement over storing those secrets in plaintext.

---

## Master Key Rotation

SentinelCore now supports explicit master-key rotation.

### What rotation does

Rotation generates a fresh master key and safely re-encrypts protected DB values under that new key.

### Protected values that are rotated

- TOTP secrets
- Encrypted SMTP password
- Any settings values already stored in encrypted form

### How it works safely

The rotation process uses a temporary key-ring phase:

1. A new key is generated
2. The key file temporarily contains the new active key plus the previous key(s)
3. Protected DB values are re-encrypted using the new key
4. The key file is compacted back to the new single active key

This avoids a dangerous window where old encrypted values become unreadable before re-encryption is complete.

### How to run it

Use the deploy script maintenance menu:

```bash
sudo bash deploy-sentinel.sh
```

Then choose:

`10) Rotate master key`

SentinelCore will offer to create a backup first before rotating the key.

### Visibility

The Settings page shows:

- the configured master key path
- the last successful master key rotation time

---

## 📦 Deploy Script

The preferred production path is the interactive deploy script:

```bash
sudo bash deploy-sentinel.sh
```

If you do not already have the repo, you can fetch and execute the installer in one line:

```bash
curl -fsSL https://raw.githubusercontent.com/ehsanR91/sentinel/main/deploy-sentinel.sh | sudo bash
```

It handles:

- Go and Node checks/install
- frontend build
- backend build
- install into `/opt/sentinelcore`
- `.env` generation
- secure bootstrap files
- systemd service setup
- service user creation
- UFW helper sudoers setup
- master key creation
- maintenance operations

### Maintenance menu

When SentinelCore is already installed, the script provides a maintenance menu with actions such as:

1. Redeploy / Upgrade
2. Reset a user password
3. Remove 2FA from a user
4. Manage UFW port
5. Uninstall SentinelCore
6. Rebuild binary and UI
7. Export backup
8. Import backup
9. Fix permissions
10. Rotate master key

The rotate flow offers an on-demand backup before re-encrypting protected values.

---

## Production Deployment Notes

### Recommended access model

Keep SentinelCore bound to localhost and access it via SSH tunnel unless you have a strong reason to expose it publicly.

Example:

```bash
ssh -L 8080:127.0.0.1:8080 your-user@your-server
```

Then open:

```text
http://localhost:8080/<secret-path>/
```

### Service user

The service is intended to run as a non-root Linux user such as `deploy`.

This same user is also used as the terminal execution identity when appropriate.

### Permissions

Important paths under `/opt/sentinelcore/data/`:

- `app.db`
- `.master.key`
- temporary bootstrap secret files

Protect these carefully.

### UFW sudoers

SentinelCore may use narrowly scoped passwordless `sudo` for UFW access where required. The deploy script writes the minimal sudoers entry it needs.

---

## Development Setup

This section covers everything needed to run SentinelCore locally for development with live-reload on both the backend and frontend.

### Prerequisites

| Tool | Purpose | Install |
|------|---------|----------|
| Go ≥ 1.21 | Backend | <https://go.dev/dl/> |
| Node.js ≥ 18 | Frontend | <https://nodejs.org> |
| [Air](https://github.com/air-verse/air) | Go live-reload | `go install github.com/air-verse/air@latest` |

---

### 1. Enable dev mode (`.dev` flag)

Create an empty `.dev` file inside the `backend/` directory:

```bash
touch backend/.dev
```

When this file is present the backend automatically applies development defaults so you do not need a real deployment or database setup:

| Setting | Dev value |
|---------|-----------|
| Port | `8888` |
| Secret gate path | `/dev/` → `http://localhost:8888/dev/` |
| Admin username | `admin` |
| Admin password | `admin` |
| JWT secret | injected automatically (dummy) |

> Remove `backend/.dev` before any production build or deploy.

---

### 2. Install sudoers entries for privileged commands

Many backend features (UFW, systemctl, apt, sysctl, etc.) call `sudo -n` for passwordless privilege escalation. Without the sudoers file installed these calls fail silently or return errors.

Run the dev sudoers setup script once:

```bash
sudo bash deploy-dev.sh
```

This will:

- Auto-detect your username from `$SUDO_USER`
- Substitute it into `deploy/sentinelcore.sudoers`
- Write `/etc/sudoers.d/sentinelcore` (chmod 440)
- Validate syntax with `visudo -c`
- Write `/etc/sudoers.d/sentinelcore-ufw` (UFW-specific entry)

You can also specify the user explicitly:

```bash
sudo bash deploy-dev.sh --user myuser
```

To remove the entries later:

```bash
sudo rm /etc/sudoers.d/sentinelcore /etc/sudoers.d/sentinelcore-ufw
```

---

### 3. Backend — run with Air (live reload)

Air watches `.go` files and recompiles automatically on save. Its config lives at `backend/.air.toml`.

```bash
cd backend
air
```

The backend will start at `http://127.0.0.1:8888` (dev mode) and rebuild on every `.go` file save.

To run without Air (single run):

```bash
cd backend
go run ./cmd/sentinelcore/
```

To build a binary manually:

```bash
cd backend
go build -o sentinelcore ./cmd/sentinelcore/
./sentinelcore
```

---

### 4. Frontend — run with Vite (HMR)

```bash
cd frontend
npm install       # first time only
npm run dev
```

The Vite dev server starts at `http://localhost:5173` and proxies all `/api` and WebSocket requests to the backend at `http://127.0.0.1:8888` automatically.

To build the frontend for production:

```bash
npm run build
```

---

### 5. Full dev workflow (two terminals)

**Terminal 1 — backend:**

```bash
touch backend/.dev          # enable dev mode (once)
cd backend && air           # live-reload backend on :8888
```

**Terminal 2 — frontend:**

```bash
cd frontend && npm run dev  # HMR frontend on :5173
```

Open `http://localhost:5173/dev/` in your browser.
Login with `admin` / `admin`.

---

### Local development note

For development you can inject config via environment variables or the `.dev` flag. Production must use `deploy-sentinel.sh` and the secure bootstrap flow — never the `admin`/`admin` defaults.

---

## Admin CLI

The backend binary includes maintenance subcommands:

```bash
sentinelcore admin list-users
sentinelcore admin reset-password <username> <new-password>
sentinelcore admin reset-2fa <username>
sentinelcore admin rotate-master-key
```

In production, prefer invoking these through `deploy-sentinel.sh` so the right environment is supplied.

---

## Security Summary

SentinelCore currently provides the following important protections:

- Argon2id password hashing
- TOTP 2FA
- RBAC
- Audit logging
- Secret-path gate
- Non-root service execution
- Audited and gated terminal access
- Encrypted storage for selected high-value secrets
- Explicit master-key rotation support

---

## Current Limitations

- This is not full database file encryption
- Protection depends on keeping `.master.key` safe
- Browser terminal access still requires careful operational discipline
- Some integrations still depend on the host environment being configured correctly

---

## Recommended Post-Install Checklist

1. Enable TOTP 2FA for privileged users.
2. Keep SentinelCore bound to localhost unless public exposure is truly needed.
3. Verify the Linux service user is non-root.
4. Confirm `/opt/sentinelcore/data/.master.key` ownership and permissions are correct.
5. Test login, SMTP alerts, and terminal high-risk unlock flow.
6. Export a backup after first successful setup.
7. Rotate the master key periodically or after any suspected host compromise.

---

## License

MIT
