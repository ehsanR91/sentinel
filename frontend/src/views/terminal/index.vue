<template>
  <div>
    <PageHeader title="Terminal" icon="mdi mdi-terminal" :items="[{text:'Terminal',active:true,icon:'mdi mdi-console-line'}]">
      <template #actions>
        <!-- Elevation badge -->
        <span v-if="elevated" class="badge me-2 d-flex align-items-center gap-1" style="background:rgba(240,64,64,0.15);color:#f04040;font-size:0.72rem;padding:4px 10px;border-radius:6px">
          <i class="mdi mdi-shield-alert"></i>
          High-Risk: {{ elevationCountdown }}s
          <button class="btn btn-sm p-0 ms-1" style="color:#f04040;line-height:1" @click="revokeElevation" title="Disable high-risk mode">
            <i class="mdi mdi-close"></i>
          </button>
        </span>
        <button v-else class="btn btn-sm me-2" style="background:rgba(245,166,35,0.1);color:#f5a623;border:1px solid rgba(245,166,35,0.25);font-size:0.75rem" :disabled="checking2FA" @click="openUnlockModal">
          <i :class="`mdi ${checking2FA ? 'mdi-loading mdi-spin' : 'mdi-shield-key-outline'} me-1`"></i>Enable High-Risk Commands
        </button>
        <button class="btn btn-sm btn-sc-danger" @click="killSession">
          <i class="mdi mdi-close me-1"></i>Kill Session
        </button>
      </template>
    </PageHeader>

    <!-- Security notice -->
    <div class="alert d-flex align-items-start gap-2 mb-3 py-2" style="background:rgba(245,166,35,0.08);border:1px solid rgba(245,166,35,0.2);border-radius:6px;font-size:0.78rem;color:#f5a623">
      <i class="mdi mdi-alert-outline" style="font-size:1rem;margin-top:1px"></i>
      <div>
        <strong>Audited Session</strong> — All commands are logged. This terminal runs as
        <code style="background:rgba(245,166,35,0.12);padding:1px 5px;border-radius:3px">{{ sessionUser }}</code>.
        Destructive commands are blocked. High-risk commands require 2FA re-authentication.
      </div>
    </div>

    <!-- Elevation active banner -->
    <div v-if="elevated" class="alert d-flex align-items-center gap-2 mb-3 py-2" style="background:rgba(240,64,64,0.08);border:1px solid rgba(240,64,64,0.25);border-radius:6px;font-size:0.78rem;color:#f04040">
      <i class="mdi mdi-shield-alert" style="font-size:1rem"></i>
      <span><strong>High-Risk Mode Active</strong> — Elevated commands permitted for {{ elevationCountdown }} seconds. Exercise extreme caution.</span>
    </div>

    <div class="row g-3">
      <div class="col-xl-9">
        <div class="card" style="background:#040810;border-color:#1e2d4a">
          <!-- Title bar -->
          <div class="card-header d-flex align-items-center justify-content-between" style="background:#0a0e1a;border-bottom:1px solid #1e2d4a;padding:.5rem 1rem">
            <div class="d-flex align-items-center gap-2">
              <div style="width:12px;height:12px;border-radius:50%;background:#f04040;opacity:0.8"></div>
              <div style="width:12px;height:12px;border-radius:50%;background:#f5a623;opacity:0.8"></div>
              <div style="width:12px;height:12px;border-radius:50%;background:#22d67c;opacity:0.8"></div>
              <span style="font-size:0.72rem;color:#5a7499;margin-left:8px;font-family:monospace">{{ sessionUser }}@{{ hostname }} — bash</span>
            </div>
            <div class="d-flex align-items-center gap-2">
              <span class="status-dot" :class="connected ? 'online' : 'offline'"></span>
              <span style="font-size:0.7rem;color:#5a7499">{{ connected ? 'Connected' : (connecting ? 'Connecting…' : 'Disconnected') }}</span>
            </div>
          </div>

          <!-- Output -->
          <div ref="termOutput" class="log-terminal" style="height:480px;overflow-y:auto;border-radius:0" @click="focusInput" @scroll.passive="onTerminalScroll" @mouseup="autoCopySelection" @contextmenu.prevent="showContextMenu" @touchstart.passive="handleTouchStart" @touchend="handleTouchEnd">
            <div v-for="(line, i) in termLines" :key="i">
              <span v-if="line.type === 'cmd'" style="user-select:none">
                <span style="color:#22d67c;font-family:monospace;font-size:0.78rem">{{ sessionUser }}@{{ hostname }}</span>
                <span style="color:#8aa4c8;font-family:monospace;font-size:0.78rem">:</span>
                <span style="color:#a78bfa;font-family:monospace;font-size:0.78rem">~</span>
                <span style="color:#c9d8f0;font-family:monospace;font-size:0.78rem">$ </span>
              </span>
              <span :class="lineClass(line)" style="font-family:monospace;font-size:0.78rem;white-space:pre-wrap">{{ line.text }}</span>
            </div>

            <div v-if="!connected && !connecting" class="d-flex align-items-center gap-2 mt-2">
              <span style="color:#f04040;font-size:0.78rem;font-family:monospace">Connection closed.</span>
              <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;font-size:0.72rem;padding:2px 10px" @click="connectWS">
                <i class="mdi mdi-refresh me-1"></i>Reconnect
              </button>
            </div>

            <div v-if="connected" class="d-flex align-items-center">
              <span style="color:#22d67c;font-family:monospace;font-size:0.78rem;user-select:none">{{ sessionUser }}@{{ hostname }}</span>
              <span style="color:#8aa4c8;font-family:monospace;font-size:0.78rem;user-select:none">:</span>
              <span style="color:#a78bfa;font-family:monospace;font-size:0.78rem;user-select:none">~</span>
              <span :style="`color:${elevated?'#f04040':'#c9d8f0'};font-family:monospace;font-size:0.78rem;user-select:none`">$ </span>
              <input
                ref="termInput"
                v-model="currentInput"
                class="terminal-input"
                style="background:transparent;border:none;outline:none;color:#c9d8f0;font-family:monospace;font-size:0.78rem;flex:1;caret-color:#22d67c"
                @keydown.enter="executeCommand"
                @keydown.up.prevent="navigateHistory(-1)"
                @keydown.down.prevent="navigateHistory(1)"
                @keydown.tab.prevent="autoComplete"
                autocomplete="off"
                spellcheck="false"
              />
            </div>
            <div ref="termBottom"></div>
          </div>
        </div>
      </div>

      <!-- Context Menu -->
      <div v-if="contextMenu.visible" ref="contextMenu" class="terminal-context-menu" :style="{ left: contextMenu.left + 'px', top: contextMenu.top + 'px' }">
        <div class="context-menu-section">
          <div class="context-menu-item" @click="copySelection">
            <i class="mdi mdi-content-copy"></i>
            <span>Copy</span>
          </div>
          <div class="context-menu-item" @click="openCopySelectionModal">
            <i class="mdi mdi-clipboard-text"></i>
            <span>Copy Selection</span>
          </div>
          <div class="context-menu-item" @click="pasteText">
            <i class="mdi mdi-content-paste"></i>
            <span>Paste</span>
          </div>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-section">
          <div class="context-menu-item" @click="clearTerminal">
            <i class="mdi mdi-delete-empty"></i>
            <span>Clear Terminal</span>
          </div>
        </div>
      </div>

      <!-- Sidebar -->
      <div class="col-xl-3">
        <div class="card mb-3">
          <div class="card-header">
            <h6><i class="mdi mdi-information-outline me-2" style="color:#4a9eff"></i>Session Info</h6>
          </div>
          <div class="card-body" style="font-size:0.78rem">
            <div class="d-flex justify-content-between mb-2">
              <span style="color:#5a7499">User</span>
              <span class="font-mono" style="color:#c9d8f0">{{ sessionUser }}</span>
            </div>
            <div class="d-flex justify-content-between mb-2">
              <span style="color:#5a7499">Host</span>
              <span class="font-mono" style="color:#c9d8f0">{{ hostname }}</span>
            </div>
            <div class="d-flex justify-content-between mb-2">
              <span style="color:#5a7499">Shell</span>
              <span class="font-mono" style="color:#22d67c">bash</span>
            </div>
            <div class="d-flex justify-content-between mb-2">
              <span style="color:#5a7499">Idle timeout</span>
              <span style="color:#f5a623">{{ idleTimeout }}m</span>
            </div>
            <div class="d-flex justify-content-between mb-2">
              <span style="color:#5a7499">High-risk mode</span>
              <span :style="`color:${elevated?'#f04040':'#5a7499'}`">{{ elevated ? 'Active' : 'Disabled' }}</span>
            </div>
            <div class="d-flex justify-content-between">
              <span style="color:#5a7499">Session start</span>
              <span style="color:#8aa4c8">{{ sessionStart }}</span>
            </div>
          </div>
        </div>

        <!-- Risk legend -->
        <div class="card mb-3">
          <div class="card-header">
            <h6><i class="mdi mdi-shield-outline me-2" style="color:#a78bfa"></i>Command Risk Levels</h6>
          </div>
          <div class="card-body" style="font-size:0.72rem">
            <div class="d-flex align-items-center gap-2 mb-2">
              <span class="badge" style="background:rgba(34,214,124,0.12);color:#22d67c;font-size:0.6rem">NORMAL</span>
              <span style="color:#8aa4c8">Runs immediately</span>
            </div>
            <div class="d-flex align-items-center gap-2 mb-2">
              <span class="badge" style="background:rgba(245,166,35,0.12);color:#f5a623;font-size:0.6rem">HIGH RISK</span>
              <span style="color:#8aa4c8">Requires 2FA unlock</span>
            </div>
            <div class="d-flex align-items-center gap-2">
              <span class="badge" style="background:rgba(240,64,64,0.12);color:#f04040;font-size:0.6rem">BLOCKED</span>
              <span style="color:#8aa4c8">Always denied</span>
            </div>
          </div>
        </div>

        <!-- SSH Port Forwarding -->
        <div class="card mb-3">
          <div class="card-header d-flex align-items-center justify-content-between" style="cursor:pointer" @click="showSshTunnels = !showSshTunnels">
            <h6 class="mb-0"><i class="mdi mdi-lan-connect me-2" style="color:#22d67c"></i>SSH Tunnels</h6>
            <i :class="`mdi mdi-chevron-${showSshTunnels ? 'up' : 'down'}`" style="color:#5a7499;font-size:0.85rem"></i>
          </div>
          <div v-if="showSshTunnels" class="card-body" style="font-size:0.72rem">
            <p style="color:#5a7499;margin-bottom:0.6rem;font-size:0.7rem">
              Run these on your <strong style="color:#8aa4c8">local machine</strong> to access server apps via your browser.
            </p>
            <div
              v-for="svc in sshServices"
              :key="svc.name"
              class="d-flex align-items-center justify-content-between mb-2 p-2"
              style="background:#0a0e1a;border:1px solid #1e2d4a;border-radius:6px"
            >
              <div class="d-flex align-items-center gap-2" style="min-width:0">
                <i :class="`mdi ${svc.icon} me-1`" :style="`color:${svc.color}`"></i>
                <div style="min-width:0">
                  <div style="color:#c9d8f0;font-weight:500;font-size:0.72rem">{{ svc.name }}</div>
                  <code style="color:#5a7499;font-size:0.65rem;white-space:nowrap">localhost:{{ svc.localPort }} → server:{{ svc.remotePort }}</code>
                </div>
              </div>
              <button
                class="btn btn-sm ms-2 flex-shrink-0"
                :style="`background:rgba(34,214,124,0.08);color:${copiedTunnel===svc.name?'#22d67c':'#5a7499'};border:1px solid rgba(34,214,124,0.15);font-size:0.65rem;white-space:nowrap`"
                @click="copyTunnel(svc)"
              >
                <i :class="`mdi mdi-${copiedTunnel===svc.name?'check':'content-copy'} me-1`"></i>{{ copiedTunnel === svc.name ? 'Copied!' : 'Copy' }}
              </button>
            </div>
            <div class="mt-2 p-2" style="background:rgba(74,158,255,0.05);border:1px solid rgba(74,158,255,0.12);border-radius:6px">
              <div class="d-flex align-items-center gap-2 mb-1">
                <span style="color:#5a7499;font-size:0.68rem">SSH user / port override:</span>
                <input v-model="sshUser" type="text" class="form-control form-control-sm d-inline-block" style="width:100px;font-size:0.68rem;padding:1px 6px;height:22px;font-family:monospace" placeholder="user" />
                <input v-model="sshPort" type="text" class="form-control form-control-sm d-inline-block" style="width:60px;font-size:0.68rem;padding:1px 6px;height:22px;font-family:monospace" placeholder="22" />
              </div>
              <div style="color:#5a7499;font-size:0.66rem;margin-top:2px">
                <i class="mdi mdi-information-outline me-1"></i>
                After connecting, open <code style="color:#4a9eff">http://localhost:&lt;localPort&gt;</code> in your browser.
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="card-header">
            <h6><i class="mdi mdi-lightning-bolt me-2" style="color:#f5a623"></i>Quick Commands</h6>
          </div>
          <div class="card-body d-flex flex-column gap-1">
            <button
              v-for="cmd in quickCmds"
              :key="cmd.label"
              class="btn btn-sm text-start"
              style="background:#0d1321;border:1px solid #1e2d4a;color:#8aa4c8;font-family:monospace;font-size:0.72rem"
              :disabled="!connected"
              @click="runQuick(cmd.cmd)"
            >
              <i class="mdi mdi-chevron-right me-1" style="color:#4a9eff"></i>{{ cmd.label }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 2FA Unlock Modal -->
    <div v-if="showUnlockModal" class="modal-backdrop-custom" @click.self="closeUnlockModal">
      <div class="modal-card-custom" style="max-width:420px">
        <div class="modal-header-custom">
          <h6 class="mb-0 d-flex align-items-center gap-2">
            <i class="mdi mdi-shield-key" style="color:#f5a623;font-size:1.1rem"></i>
            Enable High-Risk Commands
          </h6>
          <button class="btn btn-sm p-0" style="color:#5a7499" @click="closeUnlockModal">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div class="modal-body-custom">
          <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:1rem">
            Enter your TOTP 2FA code to enable high-risk terminal commands for <strong style="color:#f5a623">5 minutes</strong>.
            This session will be audited.
          </p>
          <div class="mb-3">
            <label style="font-size:0.78rem;color:#5a7499;display:block;margin-bottom:.3rem">2FA Code (6 digits)</label>
            <input
              ref="totpInput"
              v-model="totpCode"
              type="text"
              maxlength="6"
              inputmode="numeric"
              pattern="[0-9]*"
              class="form-control"
              style="font-family:monospace;letter-spacing:.2em;font-size:1.1rem;text-align:center"
              placeholder="000000"
              @keydown.enter="submitUnlock"
              @input="totpCode = totpCode.replace(/\D/g,'')"
            />
          </div>
          <div v-if="unlockError" class="mb-2" style="font-size:0.78rem;color:#f04040">
            <i class="mdi mdi-alert-circle-outline me-1"></i>{{ unlockError }}
          </div>
          <div class="d-flex gap-2">
            <button class="btn btn-sm flex-fill" style="background:rgba(245,166,35,0.12);color:#f5a623;border:1px solid rgba(245,166,35,0.3)" :disabled="totpCode.length !== 6 || unlocking" @click="submitUnlock">
              <i v-if="unlocking" class="mdi mdi-loading mdi-spin me-1"></i>
              <i v-else class="mdi mdi-shield-check me-1"></i>
              {{ unlocking ? 'Verifying…' : 'Verify & Unlock' }}
            </button>
            <button class="btn btn-sm" style="background:rgba(90,116,153,0.1);color:#5a7499" @click="closeUnlockModal">
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Copy Selection Modal -->
    <div v-if="showSelectionModal" class="modal-backdrop-custom" @click.self="closeSelectionModal">
      <div class="modal-card-custom" style="max-width:520px">
        <div class="modal-header-custom">
          <h6 class="mb-0 d-flex align-items-center gap-2">
            <i class="mdi mdi-clipboard-text" style="color:#4a9eff;font-size:1.1rem"></i>
            Copy Selection
          </h6>
          <button class="btn btn-sm p-0" style="color:#5a7499" @click="closeSelectionModal">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div class="modal-body-custom">
          <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:1rem">
            Review the selected terminal text below, then copy it manually or use the Select all button.
          </p>
          <textarea
            ref="copySelectionTextarea"
            class="form-control"
            style="min-height:180px;font-family:monospace;resize:vertical"
            readonly
            :value="selectionTextareaText"
          ></textarea>
          <div class="d-flex justify-content-between align-items-center mt-3">
            <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;border:1px solid rgba(74,158,255,0.2)" @click="selectAllSelectionText" :disabled="!selectionTextareaText">
              <i class="mdi mdi-select-all me-1"></i>Select all
            </button>
            <button class="btn btn-sm" style="background:rgba(90,116,153,0.1);color:#5a7499" @click="closeSelectionModal">
              Close
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Clipboard Fallback Modal -->
    <div v-if="showCopyFallbackModal" class="modal-backdrop-custom" @click.self="closeCopyFallbackModal">
      <div class="modal-card-custom" style="max-width:520px">
        <div class="modal-header-custom">
          <h6 class="mb-0 d-flex align-items-center gap-2">
            <i class="mdi mdi-content-copy" style="color:#4a9eff;font-size:1.1rem"></i>
            Copy Selected Terminal Text
          </h6>
          <button class="btn btn-sm p-0" style="color:#5a7499" @click="closeCopyFallbackModal">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div class="modal-body-custom">
          <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:1rem">
            Clipboard access is unavailable in your browser context. Please copy the text below manually and then close this dialog.
          </p>
          <textarea
            class="form-control"
            style="min-height:180px;font-family:monospace;resize:vertical"
            readonly
            :value="fallbackTextareaText"
          ></textarea>
          <div class="text-end mt-3">
            <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;border:1px solid rgba(74,158,255,0.2)" @click="closeCopyFallbackModal">
              Close
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 2FA required first modal -->
    <div v-if="show2FARequiredModal" class="modal-backdrop-custom" @click.self="show2FARequiredModal=false">
      <div class="modal-card-custom" style="max-width:460px">
        <div class="modal-header-custom">
          <h6 class="mb-0 d-flex align-items-center gap-2">
            <i class="mdi mdi-two-factor-authentication" style="color:#f5a623;font-size:1.1rem"></i>
            2FA Required
          </h6>
          <button class="btn btn-sm p-0" style="color:#5a7499" @click="show2FARequiredModal=false">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div class="modal-body-custom">
          <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:1rem">
            High-risk terminal commands require Two-Factor Authentication to be enabled on your account first.
          </p>
          <p style="font-size:0.78rem;color:#5a7499;margin-bottom:1rem">
            Set up 2FA in Settings, then you will be returned to Terminal to continue.
          </p>
          <div class="d-flex gap-2">
            <button class="btn btn-sm btn-sc-primary" @click="goTo2FASetup">
              <i class="mdi mdi-cog-outline me-1"></i>Go to 2FA Setup
            </button>
            <button class="btn btn-sm" style="background:rgba(90,116,153,0.1);color:#5a7499" @click="show2FARequiredModal=false">
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Pending high-risk command dialog -->
    <div v-if="pendingCmd && !showUnlockModal && !show2FARequiredModal" class="modal-backdrop-custom" @click.self="pendingCmd = null">
      <div class="modal-card-custom" style="max-width:500px">
        <div class="modal-header-custom">
          <h6 class="mb-0 d-flex align-items-center gap-2">
            <i class="mdi mdi-shield-alert" style="color:#f5a623;font-size:1.1rem"></i>
            High-Risk Command Blocked
          </h6>
          <button class="btn btn-sm p-0" style="color:#5a7499" @click="pendingCmd = null">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div class="modal-body-custom">
          <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:.75rem">
            This command matches a high-risk pattern and requires 2FA elevation:
          </p>
          <div class="mb-3 p-2" style="background:#0a0e1a;border-radius:6px;border:1px solid #1e2d4a">
            <code style="font-size:0.8rem;color:#f5a623">{{ pendingCmd.cmd }}</code>
          </div>
          <p style="font-size:0.75rem;color:#5a7499;margin-bottom:1rem">
            Matched pattern: <code style="color:#f04040">{{ pendingCmd.reason }}</code>
          </p>
          <div class="d-flex gap-2">
            <button class="btn btn-sm" style="background:rgba(245,166,35,0.12);color:#f5a623;border:1px solid rgba(245,166,35,0.3)" @click="pendingCmdUnlock">
              <i class="mdi mdi-shield-key me-1"></i>Unlock with 2FA & Run
            </button>
            <button class="btn btn-sm" style="background:rgba(90,116,153,0.1);color:#5a7499" @click="pendingCmd = null">
              Discard Command
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'

export default {
  name: 'TerminalPage',
  components: { PageHeader },

  data() {
    const userObj = (() => {
      try { return JSON.parse(sessionStorage.getItem('sc_user') || 'null') } catch (_) { return null }
    })()

    return {
      connected: false,
      connecting: false,
      sessionUser: userObj?.username || 'user',
      hostname: window.location.hostname || 'server',
      idleTimeout: 15,
      sessionStart: new Date().toLocaleTimeString(),
      currentInput: '',
      commandHistory: [],
      historyIdx: -1,
      termLines: [
        { type: 'out', text: 'SentinelCore Gated Terminal', style: 'info' },
        { type: 'out', text: 'Connecting to server…', style: '' }
      ],
      quickCmds: [
        { label: 'uptime',                    cmd: 'uptime' },
        { label: 'df -h',                     cmd: 'df -h' },
        { label: 'free -h',                   cmd: 'free -h' },
        { label: 'docker ps',                 cmd: 'docker ps' },
        { label: 'ufw status',                cmd: 'ufw status' },
        { label: 'systemctl status fail2ban', cmd: 'systemctl status fail2ban' }
      ],
      wsConn: null,

      // Elevation state
      elevated: false,
      elevationExpiry: null,
      elevationCountdown: 0,
      elevationTimer: null,

      // Unlock modal
      showUnlockModal: false,
      totpCode: '',
      unlocking: false,
      unlockError: '',
      show2FARequiredModal: false,
      checking2FA: false,

      // Pending high-risk command that triggered unlock prompt
      pendingCmd: null,

      // SSH tunnels
      showSshTunnels: false,
      copiedTunnel: '',
      sshUser: '',
      sshPort: '22',
      sshServices: [
        { name: 'Portainer',   remotePort: 9000, localPort: 9000, icon: 'mdi-docker',             color: '#2496ed' },
        { name: 'Portainer HTTPS', remotePort: 9443, localPort: 9443, icon: 'mdi-docker',         color: '#2496ed' },
        { name: 'Grafana',     remotePort: 3000, localPort: 3000, icon: 'mdi-chart-areaspline',   color: '#f46800' },
        { name: 'Prometheus',  remotePort: 9090, localPort: 9090, icon: 'mdi-database-search',    color: '#e6522c' },
        { name: 'Netdata',     remotePort: 19999, localPort: 19999, icon: 'mdi-chart-timeline',   color: '#00ab44' },
        { name: 'Uptime Kuma', remotePort: 3001, localPort: 3001, icon: 'mdi-monitor-heart',      color: '#5cdd8b' },
      ],

      // Context menu
      contextMenu: {
        visible: false,
        left: 0,
        top: 0
      },
      touchStartTime: null,
      touchX: 0,
      touchY: 0,
      touchThreshold: 500, // 500ms for touch-and-hold
      autoScroll: true,

      // Clipboard fallback
      showCopyFallbackModal: false,
      fallbackTextareaText: '',
      showSelectionModal: false,
      selectionTextareaText: ''
    }
  },

  mounted() {
    this.connectWS()
    document.addEventListener('click', this.hideContextMenuOnOutsideClick)
    document.addEventListener('keydown', this.handleGlobalCopy, true)
  },

  beforeUnmount() {
    if (this.wsConn) { this.wsConn.onclose = null; this.wsConn.close(); this.wsConn = null }
    clearInterval(this.elevationTimer)
    document.removeEventListener('click', this.hideContextMenuOnOutsideClick)
    document.removeEventListener('keydown', this.handleGlobalCopy, true)
  },

  methods: {
    // ── WebSocket ────────────────────────────────────────────────────────────
    connectWS() {
      if (this.wsConn) { this.wsConn.onclose = null; this.wsConn.close(); this.wsConn = null }
      this.connecting = true
      this.connected = false

      const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
      const host = import.meta.env.VITE_WS_HOST || window.location.host
      const url = `${proto}://${host}/api/v1/terminal/ws`

      const ws = new WebSocket(url)
      this.wsConn = ws

      ws.onopen = () => {
        this.connected = true
        this.connecting = false
        const idx = this.termLines.findIndex(l => l.text === 'Connecting to server…')
        if (idx !== -1) this.termLines.splice(idx, 1)
        this.$nextTick(() => { this.$refs.termInput?.focus({ preventScroll: true }); this.scrollBottom() })
      }

      ws.onmessage = (evt) => {
        try {
          const msg = JSON.parse(evt.data)
          this.handleServerMsg(msg)
        } catch (_) {}
      }

      ws.onerror = () => {}

      ws.onclose = () => {
        this.wsConn = null
        this.connected = false
        this.connecting = false
        this.termLines.push({ type: 'out', text: 'Session disconnected.', style: 'error' })
        this.$nextTick(this.scrollBottom)
      }
    },

    handleServerMsg(msg) {
      switch (msg.type) {
        case 'info':
          // Server sends hostname
          if (msg.data) this.hostname = msg.data
          break

        case 'output': {
          const clean = this.stripAnsi(msg.data || '')
          const style = msg.type === 'error' ? 'error' : ''
          clean.split('\n').forEach(line => {
            this.termLines.push({ type: 'out', text: line, style })
          })
          this.$nextTick(this.scrollBottom)
          break
        }

        case 'need_unlock':
          // Server rejected command — needs 2FA
          this.pendingCmd = { cmd: msg.cmd || '', reason: msg.data || '' }
          break

        case 'unlocked':
          this.setElevated(true)
          this.closeUnlockModal()
          this.termLines.push({ type: 'out', text: '[OK] High-risk mode enabled for 5 minutes.', style: 'warn' })
          // If there was a pending command, run it now
          if (this.pendingCmd) {
            const cmd = this.pendingCmd.cmd
            this.pendingCmd = null
            this.$nextTick(() => this.sendCommand(cmd))
          }
          this.$nextTick(this.scrollBottom)
          break

        case 'unlock_fail':
          this.unlocking = false
          this.unlockError = msg.data || 'Verification failed.'
          break

        case 'revoked':
          this.setElevated(false)
          this.termLines.push({ type: 'out', text: '[INFO] High-risk mode disabled.', style: 'info' })
          this.$nextTick(this.scrollBottom)
          break

        default:
          if (msg.data) {
            const clean = this.stripAnsi(msg.data)
            clean.split('\n').forEach(line => {
              this.termLines.push({ type: 'out', text: line, style: '' })
            })
            this.$nextTick(this.scrollBottom)
          }
      }
    },

    // ── Elevation ─────────────────────────────────────────────────────────────
    setElevated(val) {
      this.elevated = val
      clearInterval(this.elevationTimer)
      if (val) {
        this.elevationExpiry = Date.now() + 5 * 60 * 1000
        this.elevationTimer = setInterval(() => {
          const rem = Math.max(0, Math.ceil((this.elevationExpiry - Date.now()) / 1000))
          this.elevationCountdown = rem
          if (rem === 0) {
            this.elevated = false
            clearInterval(this.elevationTimer)
            this.termLines.push({ type: 'out', text: '[INFO] High-risk mode expired.', style: 'warn' })
            this.$nextTick(this.scrollBottom)
          }
        }, 1000)
        this.elevationCountdown = 300
      }
    },

    revokeElevation() {
      if (this.wsConn && this.wsConn.readyState === WebSocket.OPEN) {
        this.wsConn.send(JSON.stringify({ type: 'revoke' }))
      }
      this.setElevated(false)
    },

    async openUnlockModal() {
      // Ensure only one security modal is active at a time.
      this.show2FARequiredModal = false
      this.totpCode = ''
      this.unlockError = ''
      this.unlocking = false
      this.checking2FA = true
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getMe()
        if (!data?.totp_enabled) {
          this.show2FARequiredModal = true
          this.showUnlockModal = false
          return
        }
        this.showUnlockModal = true
        this.$nextTick(() => this.$refs.totpInput?.focus())
      } catch (err) {
        this.$swal({
          icon: 'error',
          title: 'Unable to verify 2FA status',
          text: err.response?.data?.error || err.message || 'Please try again.'
        })
      } finally {
        this.checking2FA = false
      }
    },

    closeUnlockModal() {
      this.showUnlockModal = false
    },

    submitUnlock() {
      if (this.totpCode.length !== 6 || this.unlocking) return
      this.unlocking = true
      this.unlockError = ''
      if (this.wsConn && this.wsConn.readyState === WebSocket.OPEN) {
        this.wsConn.send(JSON.stringify({ type: 'unlock', totp_code: this.totpCode }))
      } else {
        this.unlockError = 'Not connected.'
        this.unlocking = false
      }
    },

    pendingCmdUnlock() {
      // Dismiss the blocking dialog first; pending command still auto-runs after unlock.
      // (The command is kept in memory until unlock succeeds.)
      this.openUnlockModal()
    },

    goTo2FASetup() {
      this.show2FARequiredModal = false
      this.$router.push({ path: '/settings', query: { return_to: 'terminal' } })
    },

    // ── Command execution ──────────────────────────────────────────────────────
    executeCommand() {
      const cmd = this.currentInput.trim()
      if (!cmd || !this.connected) return

      const wasAtBottom = this.isScrolledToBottom()
      this.commandHistory.push(cmd)
      this.historyIdx = -1
      this.currentInput = ''
      this.autoScroll = wasAtBottom

      if (cmd === 'clear') {
        this.termLines = []
        this.$nextTick(this.scrollBottom)
        return
      }

      this.sendCommand(cmd)
    },

    sendCommand(cmd) {
      this.termLines.push({ type: 'cmd', text: cmd, style: '' })
      if (this.wsConn && this.wsConn.readyState === WebSocket.OPEN) {
        this.wsConn.send(JSON.stringify({ type: 'input', data: cmd }))
      }
      this.$nextTick(this.scrollBottom)
    },

    navigateHistory(dir) {
      const len = this.commandHistory.length
      if (len === 0) return
      if (dir === -1) {
        if (this.historyIdx === -1) this.historyIdx = len - 1
        else if (this.historyIdx > 0) this.historyIdx--
      } else {
        if (this.historyIdx === -1) return
        if (this.historyIdx < len - 1) this.historyIdx++
        else this.historyIdx = -1
      }
      this.currentInput = this.historyIdx === -1 ? '' : this.commandHistory[this.historyIdx]
    },

    autoComplete() {
      if (!this.currentInput) return
      const match = this.quickCmds.map(c => c.cmd).find(c => c.startsWith(this.currentInput))
      if (match) this.currentInput = match
    },

    runQuick(cmd) { this.currentInput = cmd; this.executeCommand() },

    copyTunnel(svc) {
      const host = window.location.hostname
      const user = this.sshUser || 'user'
      const port = this.sshPort || '22'
      const portFlag = port !== '22' ? ` -p ${port}` : ''
      const cmd = `ssh -L ${svc.localPort}:localhost:${svc.remotePort}${portFlag} ${user}@${host}`
      navigator.clipboard?.writeText(cmd).catch(() => {})
      this.copiedTunnel = svc.name
      setTimeout(() => { if (this.copiedTunnel === svc.name) this.copiedTunnel = '' }, 2000)
    },

    killSession() {
      this.$swal({
        title: 'Kill session?',
        text: 'This will close the WebSocket connection.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#f04040',
        confirmButtonText: 'Kill'
      }).then(r => {
        if (r.isConfirmed) {
          if (this.wsConn) { this.wsConn.onclose = null; this.wsConn.close(); this.wsConn = null }
          this.connected = false
          this.setElevated(false)
          this.termLines.push({ type: 'out', text: 'Session terminated by user.', style: 'error' })
          this.$nextTick(this.scrollBottom)
        }
      })
    },

    // ── Helpers ───────────────────────────────────────────────────────────────
    stripAnsi(str) {
      // eslint-disable-next-line no-control-regex
      return str.replace(/\x1B\[[0-?]*[ -/]*[@-~]/g, '')
    },

    focusInput() {
      const selectedText = window.getSelection?.()?.toString?.() || ''
      if (selectedText.trim().length > 0) return
      this.$refs.termInput?.focus({ preventScroll: true })
    },

    isScrolledToBottom() {
      const el = this.$refs.termOutput
      if (!el) return true
      return el.scrollTop + el.clientHeight >= el.scrollHeight - 28
    },

    scrollBottom() {
      if (!this.autoScroll) return
      this.$refs.termBottom?.scrollIntoView({ behavior: 'smooth' })
    },

    onTerminalScroll() {
      const el = this.$refs.termOutput
      if (!el) return
      const threshold = 28
      this.autoScroll = el.scrollTop + el.clientHeight >= el.scrollHeight - threshold
    },

    async autoCopySelection() {
      const sel = window.getSelection?.()
      const text = sel?.toString?.() || ''
      if (!text.trim()) return
      const out = this.$refs.termOutput
      const anchorNode = sel.anchorNode
      const focusNode = sel.focusNode
      const inTerminal = !!out && ((anchorNode && out.contains(anchorNode)) || (focusNode && out.contains(focusNode)))
      if (!inTerminal) return
      try {
        if (navigator.clipboard?.writeText) {
          await navigator.clipboard.writeText(text)
        }
      } catch (_) {}
    },

    handleGlobalCopy(e) {
      const key = (e.key || '').toLowerCase()
      if (!(e.ctrlKey || e.metaKey) || key !== 'c') return
      const sel = window.getSelection?.()
      const text = sel?.toString?.() || ''
      if (!text.trim()) return
      const out = this.$refs.termOutput
      const anchorNode = sel.anchorNode
      const focusNode = sel.focusNode
      const inTerminal = !!out && ((anchorNode && out.contains(anchorNode)) || (focusNode && out.contains(focusNode)))
      if (!inTerminal) return
      e.preventDefault()

      if (!navigator.clipboard?.writeText) {
        this.fallbackTextareaText = text
        this.showCopyFallbackModal = true
        return
      }

      navigator.clipboard.writeText(text).catch(() => {
        this.fallbackTextareaText = text
        this.showCopyFallbackModal = true
      })
    },

    lineClass(line) {
      if (line.style === 'error') return 'log-error'
      if (line.style === 'info')  return 'log-info'
      if (line.style === 'warn')  return 'log-warn'
      return ''
    },

    // ── Context Menu ───────────────────────────────────────────────────────────
    showContextMenu(e) {
      e.preventDefault()
      // Don't show if clicking on input
      if (e.target.tagName === 'INPUT') return

      const menuWidth = 200
      const menuHeight = 150
      this.contextMenu.left = Math.min(e.clientX, window.innerWidth - menuWidth - 12)
      this.contextMenu.top = Math.min(e.clientY, window.innerHeight - menuHeight - 12)
      this.contextMenu.visible = true
    },

    hideContextMenuOnOutsideClick(e) {
      if (this.contextMenu.visible) {
        const menu = this.$refs.contextMenu
        if (menu && !menu.contains(e.target)) {
          this.contextMenu.visible = false
        }
      }
    },

    handleTouchStart(e) {
      this.touchStartTime = Date.now()
      // Store touch position for context menu
      if (e.touches && e.touches[0]) {
        this.touchX = e.touches[0].clientX
        this.touchY = e.touches[0].clientY
      }
    },

    handleTouchEnd(e) {
      const touchDuration = Date.now() - this.touchStartTime
      if (touchDuration >= this.touchThreshold) {
        // Touch-and-hold detected, show context menu
        this.contextMenu.visible = true
        this.contextMenu.left = this.touchX || 0
        this.contextMenu.top = this.touchY || 0
      }
      this.touchStartTime = null
    },

    async copySelection() {
      const selection = window.getSelection()
      const text = selection?.toString?.() || ''
      if (!text.trim()) return

      try {
        if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable')
        await navigator.clipboard.writeText(text)
        this.contextMenu.visible = false
      } catch (err) {
        console.error('Copy failed:', err)
        this.fallbackTextareaText = text
        this.showCopyFallbackModal = true
        this.contextMenu.visible = false
      }
    },

    openCopySelectionModal() {
      const selection = window.getSelection?.()?.toString?.() || ''
      this.selectionTextareaText = selection.trim()
      this.showSelectionModal = true
      this.contextMenu.visible = false
      this.$nextTick(() => {
        if (this.selectionTextareaText && this.$refs.copySelectionTextarea) {
          this.$refs.copySelectionTextarea.select()
        }
      })
    },

    selectAllSelectionText() {
      if (this.$refs.copySelectionTextarea) {
        this.$refs.copySelectionTextarea.select()
      }
    },

    closeSelectionModal() {
      this.showSelectionModal = false
      this.selectionTextareaText = ''
    },

    async pasteText() {
      try {
        if (!navigator.clipboard?.readText) throw new Error('Clipboard API not available')
        const text = await navigator.clipboard.readText()
        this.currentInput += text
        this.$refs.termInput?.focus()
        this.contextMenu.visible = false
      } catch (err) {
        console.error('Paste failed:', err)
      }
    },

    closeCopyFallbackModal() {
      this.showCopyFallbackModal = false
      this.fallbackTextareaText = ''
    },

    runQuickFromMenu(cmd) {
      this.currentInput = cmd
      this.executeCommand()
      this.contextMenu.visible = false
    },

    clearTerminal() {
      this.termLines = []
      this.contextMenu.visible = false
    }
  }
}
</script>

<style scoped>
.log-warn { color: #f5a623 }

.modal-backdrop-custom {
  position: fixed;
  inset: 0;
  background: rgba(4, 8, 16, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1050;
  backdrop-filter: blur(2px);
}

.modal-card-custom {
  background: #0d1321;
  border: 1px solid #1e2d4a;
  border-radius: 10px;
  width: 100%;
  margin: 1rem;
  overflow: hidden;
}

.modal-header-custom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: .75rem 1rem;
  border-bottom: 1px solid #1e2d4a;
  font-size: 0.85rem;
  color: #c9d8f0;
}

.modal-body-custom {
  padding: 1.25rem 1rem;
}

/* Terminal output helpers */
.log-terminal {
  font-family: monospace;
  white-space: pre-wrap;
  word-break: break-word;
  -webkit-user-select: text;
  user-select: text;
}

/* Context Menu Styles - macOS inspired */
.terminal-context-menu {
  position: fixed;
  width: 86px;
  min-width: 86px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  z-index: 9999;
  padding: 4px;
  animation: contextMenuFadeIn 0.2s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

@keyframes contextMenuFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-10px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.context-menu-section {
  padding: 2px 0;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px 6px;
  color: #333;
  font-size: 11px;
  font-weight: 400;
  cursor: pointer;
  transition: all 0.15s ease;
  border-radius: 4px;
  margin: 1px;
}

.context-menu-item:hover {
  background: rgba(0, 122, 255, 0.1);
  color: #007AFF;
}

.context-menu-item:active {
  background: rgba(0, 122, 255, 0.2);
  transform: scale(0.98);
}

.context-menu-item i {
  width: 12px;
  height: 12px;
  text-align: center;
  color: #666;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.context-menu-item:hover i {
  color: #007AFF;
}

.context-menu-divider {
  height: 1px;
  background: rgba(0, 0, 0, 0.1);
  margin: 3px 8px;
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
  .terminal-context-menu {
    background: rgba(30, 30, 30, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }
  
  .context-menu-item {
    color: #fff;
  }
  
  .context-menu-item:hover {
    background: rgba(0, 122, 255, 0.2);
    color: #007AFF;
  }
  
  .context-menu-item i {
    color: #999;
  }
  
  .context-menu-item:hover i {
    color: #007AFF;
  }
  
  .context-menu-divider {
    background: rgba(255, 255, 255, 0.1);
  }
}
</style>

