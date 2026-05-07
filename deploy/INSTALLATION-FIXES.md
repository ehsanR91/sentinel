 a# Installation Troubleshooting Guide

This document describes fixes for common installation issues with SentinelCore.

## Issues Fixed

### 1. Firewall Connections Not Showing

**Problem:** The `/firewall` section shows no active connections even though the API returns data.

**Cause:** The backend returns connection state as "ESTAB" but the frontend was checking for "ESTABLISHED".

**Fix Applied:** Updated [`sentinelcore/frontend/src/views/firewall/index.vue`](sentinelcore/frontend/src/views/firewall/index.vue) to:

- Normalize connection state display (ESTAB → ESTABLISHED)
- Add auto-refresh for connections every 10 seconds
- Add `formatState()` and `getConnColor()` helper methods

### 2. Offline/Online Status in Header

**Problem:** The WebSocket status indicator shows "Offline" randomly.

**Cause:** The status depends on WebSocket connection which can be affected by:

- Network interruptions
- Server restarts
- Session timeouts

**Status:** The WebSocket status indicator in [`topbar.vue`](sentinelcore/frontend/src/components/topbar.vue:24) correctly reflects the real-time connection state. If it shows "Offline" frequently, check:

- Network stability
- Backend server availability
- WebSocket endpoint accessibility

### 3. ClamAV Installation Failure (GPG Key Issues)

**Error:**

```
Error during installation: exit status 100
```

**Cause:** Missing or expired GPG keys for repository signatures.

**Fix:** Run the comprehensive fix script:

```bash
sudo bash sentinelcore/deploy/fix-all-repo-keys.sh
```

This script:

- Checks filesystem writability
- Cleans up stale dpkg/apt locks
- Imports missing GPG keys for common repositories
- Updates source lists with signed-by directives
- Repairs broken package states

### 4. Read-Only Filesystem Errors

**Error:**

```
dpkg: error while cleaning up:
unable to remove newly-extracted version of '/usr/bin/dockerd-rootless-setuptool.sh': Read-only file system
```

**Cause:** The filesystem is mounted as read-only, possibly due to:

- Filesystem errors
- Container restrictions
- Mount configuration issues

**Fix:**

1. Check filesystem status:

   ```bash
   mount | grep " ro "
   ```

2. Attempt remount:

   ```bash
   sudo mount -o remount,rw /
   ```

3. If remount fails, check for filesystem errors:

   ```bash
   sudo fsck -n /
   ```

4. Run the fix script which handles read-only detection:

   ```bash
   sudo bash sentinelcore/deploy/fix-all-repo-keys.sh
   ```

### 5. GitHub CLI Installation Failure

**Error:**

```
dd: failed to open '/usr/share/keyrings/githubcli-archive-keyring.gpg': Read-only file system
curl: (23) Failure writing output to destination
```

**Cause:** Same as #4 - filesystem is read-only.

**Fix Applied:** Updated [`apps/install-apps.sh`](apps/install-apps.sh) to:

- Check if filesystem is writable before attempting installation
- Use proper GPG key import with error handling
- Provide clear error messages when filesystem is read-only

**Manual Fix:**

```bash
# First fix the filesystem
sudo mount -o remount,rw /usr

# Then run the fix script
sudo bash sentinelcore/deploy/fix-all-repo-keys.sh

# Retry installation
sudo bash apps/install-apps.sh
```

## Quick Start - Fix All Issues

Run this single command to fix all repository and installation issues:

```bash
sudo bash sentinelcore/deploy/fix-all-repo-keys.sh
```

Then retry your installation:

```bash
# For ClamAV
sudo apt install clamav-daemon

# Or run the app installer
sudo bash apps/install-apps.sh
```

## Manual Steps (if script fails)

### Fix GPG Keys Manually

```bash
# Create keyring directory
sudo mkdir -p /usr/share/keyrings
sudo mkdir -p /etc/apt/keyrings

# Import Docker key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Import Netdata key  
curl -fsSL https://packagecloud.io/netdata/netdata/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/netdata-archive-keyring.gpg

# Import GitHub CLI key
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg

# Update apt
sudo apt-get update
```

### Fix dpkg State

```bash
# Configure any pending packages
sudo dpkg --configure -a

# Fix broken dependencies
sudo apt-get install -f -y

# Clean apt cache
sudo apt-get clean
sudo apt-get autoclean
```

## Prevention

To prevent these issues in the future:

1. **Regular Updates:** Run `sudo apt update && sudo apt upgrade -y` weekly
2. **Monitor Disk Space:** Ensure at least 10% free disk space
3. **Check Filesystem:** Run `sudo fsck` periodically
4. **Use Fix Script:** Run `fix-all-repo-keys.sh` before major installations

## Support

If issues persist after running the fix scripts:

1. Check system logs: `journalctl -xe`
2. Check apt logs: `cat /var/log/apt/history.log`
3. Check dpkg status: `cat /var/lib/dpkg/status`
4. Verify disk health: `sudo smartctl -a /dev/sda`
