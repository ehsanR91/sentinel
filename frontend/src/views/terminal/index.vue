<template>
  <div :class="{ 'terminal-popout-root': isPopout }">
    <PageHeader v-if="!isPopout" title="Terminal" icon="mdi mdi-console" :items="[{text:'Terminal',active:true,icon:'mdi mdi-console-line'}]">
      <template #actions>
        <!-- Elevation badge -->
        <span v-if="elevated" class="badge me-2 d-flex align-items-center gap-1" style="background:rgba(240,64,64,0.15);color:#f04040;font-size:0.72rem;padding:4px 10px;border-radius:6px">
          <i class="mdi mdi-shield-alert"></i>
          High-Risk: {{ elevationCountdown }}s
          <Tooltip label="Disable high-risk mode" description="Immediately revoke elevated terminal permissions for this session." variant="rich" as-child>
            <button class="btn btn-sm p-0 ms-1" style="color:#f04040;line-height:1" @click="revokeElevation">
              <i class="mdi mdi-close"></i>
            </button>
          </Tooltip>
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
    <div v-if="!isPopout" class="alert d-flex align-items-start gap-2 mb-3 py-2" style="background:rgba(245,166,35,0.08);border:1px solid rgba(245,166,35,0.2);border-radius:6px;font-size:0.78rem;color:#f5a623">
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
      <div :class="isPopout ? 'col-12' : 'col-xl-9'">
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
              <button
                v-if="!isPopout"
                class="terminal-window-btn"
                :class="{ active: popoutOpen }"
                :title="popoutOpen ? 'Focus popout terminal' : 'Pop out terminal'"
                @click="openPopout"
              ><i class="mdi mdi-open-in-new"></i></button>
              <button
                v-else
                class="terminal-window-btn terminal-popback-btn"
                title="Pop back into main app"
                @click="popBackIn"
              ><i class="mdi mdi-arrow-collapse-left"></i></button>
            </div>
          </div>

          <!-- Risk legend strip -->
          <div class="term-risk-legend">
            <span class="term-risk-badge term-risk-badge--ok">NORMAL</span><span class="term-risk-label">runs immediately</span>
            <span class="term-risk-sep"></span>
            <span class="term-risk-badge term-risk-badge--warn">HIGH RISK</span><span class="term-risk-label">requires 2FA</span>
            <span class="term-risk-sep"></span>
            <span class="term-risk-badge term-risk-badge--err">BLOCKED</span><span class="term-risk-label">always denied</span>
          </div>

          <!-- Output -->
          <div ref="termOutput" class="log-terminal" :style="`height:${isPopout ? 'calc(100vh - 90px)' : '480px'};overflow-y:auto;border-radius:0`" @click="focusInput" @scroll.passive="onTerminalScroll" @mouseup="autoCopySelection" @contextmenu.prevent="showContextMenu" @touchstart.passive="handleTouchStart" @touchend="handleTouchEnd">
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
                @paste.prevent="handlePaste"
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
          <div class="context-menu-item" @click="openQuickCmdPalette">
            <i class="mdi mdi-lightning-bolt"></i>
            <span>Quick Commands</span>
            <i class="mdi mdi-chevron-right" style="margin-left:auto;opacity:0.5"></i>
          </div>
          <div class="context-menu-item" @click="clearTerminal">
            <i class="mdi mdi-delete-empty"></i>
            <span>Clear Terminal</span>
          </div>
        </div>
      </div>

      <!-- Quick Commands Palette -->
      <div v-if="quickCmdPalette.visible" ref="quickCmdPalette" class="quick-cmd-palette" :style="{ left: quickCmdPalette.left + 'px', top: quickCmdPalette.top + 'px' }" @click.stop>
        <div class="quick-cmd-palette__header">
          <i class="mdi mdi-lightning-bolt" style="color:#f5a623"></i>
          <span>Quick Commands</span>
          <button class="quick-cmd-palette__close" @click="quickCmdPalette.visible = false"><i class="mdi mdi-close"></i></button>
        </div>
        <div class="quick-cmd-palette__search-wrap">
          <i class="mdi mdi-magnify"></i>
          <input
            ref="quickCmdSearch"
            v-model="quickCmdPalette.search"
            class="quick-cmd-palette__search"
            placeholder="Search commands…"
            autocomplete="off"
            spellcheck="false"
            @click.stop
            @keydown.escape="quickCmdPalette.visible = false"
          />
        </div>
        <div class="quick-cmd-palette__list">
          <template v-if="filteredQuickCmds.length">
            <template v-for="(group, cat) in groupedQuickCmds" :key="cat">
              <div class="quick-cmd-palette__category">{{ cat }}</div>
              <div
                v-for="cmd in group"
                :key="cmd.cmd"
                class="quick-cmd-palette__item"
                @click="runQuickFromMenu(cmd.cmd)"
              >
                <span class="quick-cmd-palette__label">{{ cmd.label }}</span>
                <code class="quick-cmd-palette__cmd">{{ cmd.cmd }}</code>
              </div>
            </template>
          </template>
          <div v-else class="quick-cmd-palette__empty">No commands match</div>
        </div>
      </div>

      <!-- Sidebar -->
      <div v-if="!isPopout" class="col-xl-3">
        <div class="card mb-3">
          <div class="card-header d-flex align-items-center justify-content-between" style="cursor:pointer" @click="showSessionInfo = !showSessionInfo">
            <h6 class="mb-0"><i class="mdi mdi-information-outline me-2" style="color:#4a9eff"></i>Session Info</h6>
            <i :class="`mdi mdi-chevron-${showSessionInfo ? 'up' : 'down'}`" style="color:#5a7499;font-size:0.85rem"></i>
          </div>
          <div v-if="showSessionInfo" class="card-body" style="font-size:0.78rem">
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
          <div class="card-header d-flex align-items-center justify-content-between" style="cursor:pointer" @click="showQuickCmds = !showQuickCmds">
            <h6 class="mb-0"><i class="mdi mdi-lightning-bolt me-2" style="color:#f5a623"></i>Quick Commands</h6>
            <i :class="`mdi mdi-chevron-${showQuickCmds ? 'up' : 'down'}`" style="color:#5a7499;font-size:0.85rem"></i>
          </div>
          <div v-if="showQuickCmds" class="qc-panel">
            <div v-for="(cmds, cat) in quickCmdsByCategory" :key="cat" class="qc-cat">
              <button class="qc-cat__head" @click.stop="toggleCmdCategory(cat)">
                <i class="mdi mdi-chevron-right qc-cat__arrow" :class="{ open: openCmdCategories[cat] }"></i>
                <span class="qc-cat__name">{{ cat }}</span>
                <span class="qc-cat__count">{{ cmds.length }}</span>
              </button>
              <div v-if="openCmdCategories[cat]" class="qc-cat__items">
                <div v-for="cmd in cmds" :key="cmd.cmd" class="qc-item">
                  <button class="qc-item__run" :disabled="!connected" @click="runQuick(cmd.cmd)">
                    <i class="mdi mdi-chevron-right" style="color:#4a9eff;font-size:0.7rem;flex-shrink:0"></i>
                    <span>{{ cmd.label }}</span>
                  </button>
                  <div class="qc-item__meta">
                    <Tooltip :label="cmd.label" :description="cmd.desc || cmd.cmd" variant="rich" as-child>
                      <button class="qc-item__meta-btn"><i class="mdi mdi-information-outline"></i></button>
                    </Tooltip>
                    <a :href="googleSearchUrl(cmd)" target="_blank" rel="noopener noreferrer" class="qc-item__meta-btn qc-item__search-btn" title="Search on Google" @click.stop>
                      <i class="mdi mdi-magnify"></i>
                    </a>
                  </div>
                </div>
              </div>
            </div>
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
import Tooltip from '@/components/ui/tooltip.vue'
import api from '@/services/api'

export default {
  name: 'TerminalPage',
  components: { PageHeader, Tooltip },

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
        // System Info
        { label: 'Uptime',              cmd: 'uptime',                                              category: 'System Info',        desc: 'Shows how long the system has been running, plus 1/5/15-min load averages and logged-in user count.' },
        { label: 'Hostname (FQDN)',     cmd: 'hostname -f',                                         category: 'System Info',        desc: 'Prints the fully-qualified domain name (FQDN) of the machine.' },
        { label: 'OS release',          cmd: 'cat /etc/os-release',                                category: 'System Info',        desc: 'Shows Linux distribution name, version ID, codename, and official URLs.' },
        { label: 'Kernel version',      cmd: 'uname -r',                                            category: 'System Info',        desc: 'Prints the running kernel release version string.' },
        { label: 'Full system info',    cmd: 'uname -a',                                            category: 'System Info',        desc: 'All uname info: kernel, hostname, machine type, and OS in one line.' },
        { label: 'Date & time',         cmd: 'date && timedatectl',                                 category: 'System Info',        desc: 'Shows current date/time and NTP synchronization/timezone status.' },
        { label: 'Boot time',           cmd: 'who -b',                                              category: 'System Info',        desc: 'Prints the exact date and time of the last system boot.' },
        { label: 'CPU info',            cmd: 'lscpu | head -25',                                    category: 'System Info',        desc: 'CPU architecture, cores, threads, speed, cache sizes, and virtualization flags.' },
        { label: 'Environment vars',    cmd: 'printenv | sort',                                     category: 'System Info',        desc: 'Lists all environment variables for the current shell session, sorted alphabetically.' },
        { label: 'System locale',       cmd: 'locale',                                              category: 'System Info',        desc: 'Current locale settings: language, character encoding, and collation order.' },
        // Process Monitoring
        { label: 'Top by CPU',          cmd: 'ps aux --sort=-%cpu | head -20',                     category: 'Process Monitoring', desc: 'Top 20 processes sorted by CPU usage (highest first).' },
        { label: 'Top by RAM',          cmd: 'ps aux --sort=-%mem | head -20',                     category: 'Process Monitoring', desc: 'Top 20 processes sorted by memory (RSS) usage.' },
        { label: 'Process tree',        cmd: 'pstree -p',                                           category: 'Process Monitoring', desc: 'Entire process hierarchy as an ASCII tree with PIDs.' },
        { label: 'Process count',       cmd: 'ps aux | wc -l',                                      category: 'Process Monitoring', desc: 'Total number of running processes on the system.' },
        { label: 'Zombie processes',    cmd: "ps aux | awk '$8 ~ /Z/ {print $0}'",                 category: 'Process Monitoring', desc: 'Zombie processes: finished but not reaped by their parent (potential resource leak).' },
        { label: 'Load averages',       cmd: 'cat /proc/loadavg',                                   category: 'Process Monitoring', desc: '1m/5m/15m load averages, running/total thread count, and last created PID.' },
        { label: 'CPU stats (vmstat)',  cmd: 'vmstat 1 5',                                          category: 'Process Monitoring', desc: 'Virtual memory stats sampled 5 times: CPU, I/O, context switches, interrupts.' },
        { label: 'I/O wait (iostat)',   cmd: 'iostat -c 1 5 2>/dev/null || echo "sysstat not installed"', category: 'Process Monitoring', desc: 'CPU idle/wait % sampled 5 times — high iowait indicates a disk bottleneck.' },
        { label: 'Per-process I/O',     cmd: 'iotop -bo -n1 2>/dev/null || echo "iotop not installed"', category: 'Process Monitoring', desc: 'One-shot view of I/O per process. Requires iotop (apt install iotop).' },
        { label: 'Open file handles',   cmd: 'lsof 2>/dev/null | wc -l',                           category: 'Process Monitoring', desc: 'Total number of open file descriptors across all processes (system-wide).' },
        // Memory & Swap
        { label: 'Memory overview',     cmd: 'free -h',                                             category: 'Memory & Swap',      desc: 'Human-readable total/used/free RAM and swap, including buffer/cache breakdown.' },
        { label: 'Memory detail',       cmd: 'cat /proc/meminfo | head -30',                       category: 'Memory & Swap',      desc: 'Kernel-level memory breakdown: MemFree, Buffers, Cached, SReclaimable, Slab, dirty pages.' },
        { label: 'Swap usage',          cmd: 'swapon --show',                                       category: 'Memory & Swap',      desc: 'Active swap partitions/files with size, used space, and priority.' },
        { label: 'VM statistics',       cmd: 'vmstat -s | head -25',                               category: 'Memory & Swap',      desc: 'Cumulative virtual memory stats: pages swapped in/out, faults, interrupts since boot.' },
        { label: 'Huge pages',          cmd: 'grep -i hugepage /proc/meminfo',                     category: 'Memory & Swap',      desc: 'Huge pages configuration: HugePages_Total, Free, Rsvd, Surp, and page size.' },
        { label: 'Top mem consumers',   cmd: "ps aux --sort=-%mem | awk 'NR==1 || $4>0.5 {print}' | head -15", category: 'Memory & Swap', desc: 'Processes using more than 0.5% of RAM, sorted by memory consumption.' },
        // Disk & Storage
        { label: 'Disk usage',          cmd: 'df -h',                                               category: 'Disk & Storage',     desc: 'Disk space used/available on all mounted filesystems in human-readable format.' },
        { label: 'Inode usage',         cmd: 'df -i',                                               category: 'Disk & Storage',     desc: 'Inode usage per filesystem. Running out of inodes prevents new file creation even when bytes are free.' },
        { label: 'Block devices',       cmd: 'lsblk',                                               category: 'Disk & Storage',     desc: 'Block devices (disks, partitions, LVM, RAID) in a tree format with sizes.' },
        { label: 'Mount points',        cmd: 'mount | column -t',                                   category: 'Disk & Storage',     desc: 'All currently mounted filesystems with device, mount point, and options.' },
        { label: 'Dir sizes /',         cmd: 'du -sh /* 2>/dev/null | sort -rh | head -15',        category: 'Disk & Storage',     desc: 'Top-level directory sizes sorted largest first. Good for finding space hogs.' },
        { label: 'Large files >100MB',  cmd: 'find / -xdev -type f -size +100M 2>/dev/null | head -20', category: 'Disk & Storage', desc: 'Files larger than 100 MB. -xdev avoids crossing filesystem boundaries.' },
        { label: 'Disk I/O stats',      cmd: 'iostat -x 1 3 2>/dev/null || cat /proc/diskstats | head -20', category: 'Disk & Storage', desc: 'Per-device extended I/O stats: utilization, await, r/w throughput over 3 seconds.' },
        // Network
        { label: 'Interfaces',          cmd: 'ip addr show',                                        category: 'Network',            desc: 'All network interfaces with IP addresses, MAC addresses, MTU, and UP/DOWN state.' },
        { label: 'Routing table',       cmd: 'ip route show',                                       category: 'Network',            desc: 'Kernel routing table including default gateway, interface routes, and metrics.' },
        { label: 'Listening ports',     cmd: 'ss -tulpn',                                           category: 'Network',            desc: 'All TCP/UDP listening sockets with the process name and PID that owns each port.' },
        { label: 'Established conns',   cmd: 'ss -tn state established',                            category: 'Network',            desc: 'All currently established TCP connections with local and remote endpoints.' },
        { label: 'Connection summary',  cmd: 'ss -s',                                               category: 'Network',            desc: 'Summary of socket statistics: total, TCP states, UDP, raw, fragment counts.' },
        { label: 'DNS config',          cmd: 'cat /etc/resolv.conf',                               category: 'Network',            desc: 'Configured DNS nameserver IPs, search domains, and resolver options.' },
        { label: 'Ping gateway',        cmd: "ping -c4 $(ip route | awk '/default/{print $3}')",   category: 'Network',            desc: 'Pings the default gateway 4 times to verify local network connectivity and latency.' },
        { label: 'ARP table',           cmd: 'arp -n 2>/dev/null || ip neigh show',                category: 'Network',            desc: 'ARP cache: known IP-to-MAC mappings for devices on the local network segment.' },
        { label: 'Interface stats',     cmd: 'cat /proc/net/dev',                                   category: 'Network',            desc: 'Raw kernel packet/byte counters per interface: received, transmitted, errors, drops.' },
        { label: 'External IP',         cmd: 'curl -s --max-time 5 ifconfig.me || curl -s --max-time 5 icanhazip.com', category: 'Network', desc: 'Fetches your public/external IP address from an external web service.' },
        // Firewall
        { label: 'UFW status',          cmd: 'ufw status',                                          category: 'Firewall',           desc: 'UFW firewall status (active/inactive) and summarized rule list.' },
        { label: 'UFW numbered',        cmd: 'ufw status numbered',                                 category: 'Firewall',           desc: 'UFW rules with numbers so you can delete specific rules by number.' },
        { label: 'UFW verbose',         cmd: 'ufw status verbose',                                  category: 'Firewall',           desc: 'Detailed UFW status including default policies, logging level, and all rules.' },
        { label: 'iptables rules',      cmd: 'iptables -L -n -v --line-numbers 2>/dev/null | head -60', category: 'Firewall',      desc: 'All iptables rules in all chains with byte/packet counters and line numbers.' },
        { label: 'nftables',            cmd: 'nft list ruleset 2>/dev/null || echo "nftables not active"', category: 'Firewall',   desc: 'Full nftables ruleset if nftables is in use as the netfilter backend.' },
        // Security
        { label: 'fail2ban status',     cmd: 'fail2ban-client status 2>/dev/null',                  category: 'Security',          desc: 'fail2ban service status and list of all configured jails (sshd, nginx, etc.).' },
        { label: 'fail2ban sshd',       cmd: 'fail2ban-client status sshd 2>/dev/null',             category: 'Security',          desc: 'fail2ban sshd jail: currently banned IPs, failure count, and ban time.' },
        { label: 'CrowdSec alerts',     cmd: 'cscli alerts list 2>/dev/null || echo "CrowdSec not installed"', category: 'Security', desc: 'Recent CrowdSec IDS alerts including attack type, source IP, and scenario name.' },
        { label: 'CrowdSec decisions',  cmd: 'cscli decisions list 2>/dev/null || echo "CrowdSec not installed"', category: 'Security', desc: 'Active CrowdSec enforcement decisions (bans, captchas) with expiry times.' },
        { label: 'Last 20 logins',      cmd: 'last -n 20',                                          category: 'Security',          desc: 'Last 20 successful logins: username, terminal, source IP, login/logout times, duration.' },
        { label: 'Failed SSH logins',   cmd: 'grep "Failed password" /var/log/auth.log 2>/dev/null | tail -25', category: 'Security', desc: 'Recent failed SSH password attempts from auth.log with source IPs.' },
        { label: 'Auth log tail',       cmd: 'tail -50 /var/log/auth.log 2>/dev/null',              category: 'Security',          desc: 'Last 50 lines of the authentication log including PAM, sudo, and SSH events.' },
        { label: 'SUID binaries',       cmd: 'find / -perm -4000 -type f 2>/dev/null | sort',      category: 'Security',          desc: 'Files with SUID bit set — they run with owner privileges. Unexpected entries = risk.' },
        { label: 'World-writable dirs', cmd: 'find / -xdev -type d -perm -0002 2>/dev/null | grep -v proc | head -20', category: 'Security', desc: 'Directories writable by any user. Can be exploited for privilege escalation.' },
        { label: 'Open on all ifaces',  cmd: 'ss -tlnp | grep "0.0.0.0\\|:::"',                   category: 'Security',          desc: 'Services listening on 0.0.0.0/::: are exposed to all network interfaces including public ones.' },
        // Services
        { label: 'Failed services',     cmd: 'systemctl --failed',                                  category: 'Services',          desc: 'All systemd units in a failed state — first thing to check when diagnosing issues.' },
        { label: 'Running services',    cmd: 'systemctl list-units --type=service --state=running', category: 'Services',          desc: 'All currently active and running systemd service units.' },
        { label: 'All loaded services', cmd: 'systemctl list-units --type=service',                 category: 'Services',          desc: 'All loaded systemd services regardless of state (active, inactive, failed).' },
        { label: 'Recent errors',       cmd: 'journalctl -b -p err --no-pager -n 30',               category: 'Services',          desc: 'Last 30 error/critical messages from all services since last boot.' },
        { label: 'Status: nginx',       cmd: 'systemctl status nginx',                               category: 'Services',          desc: 'nginx: active state, PID, memory, recent log lines, and enabled status.' },
        { label: 'Status: fail2ban',    cmd: 'systemctl status fail2ban',                            category: 'Services',          desc: 'fail2ban IDS daemon status and recent log output.' },
        { label: 'Status: ssh',         cmd: 'systemctl status ssh 2>/dev/null || systemctl status sshd', category: 'Services',   desc: 'SSH server status, PID, and recent authentication events.' },
        { label: 'Status: docker',      cmd: 'systemctl status docker',                              category: 'Services',          desc: 'Docker daemon status, version, and recent startup/error messages.' },
        { label: 'Status: cron',        cmd: 'systemctl status cron 2>/dev/null || systemctl status crond', category: 'Services', desc: 'Cron daemon status (Debian: cron, RHEL: crond).' },
        // Docker
        { label: 'Running containers',  cmd: 'docker ps',                                            category: 'Docker',            desc: 'Running containers: ID, image, command, uptime, ports, and name.' },
        { label: 'All containers',      cmd: 'docker ps -a',                                         category: 'Docker',            desc: 'All containers including stopped, created, and exited ones.' },
        { label: 'Images',              cmd: 'docker images',                                         category: 'Docker',            desc: 'Local Docker images with repository, tag, image ID, creation date, and disk size.' },
        { label: 'Docker disk usage',   cmd: 'docker system df',                                     category: 'Docker',            desc: 'How much disk Docker is using: images, containers, volumes, and build cache.' },
        { label: 'Container stats',     cmd: 'docker stats --no-stream',                             category: 'Docker',            desc: 'Snapshot of CPU%, memory, network I/O, and block I/O for all running containers.' },
        { label: 'Volumes',             cmd: 'docker volume ls',                                     category: 'Docker',            desc: 'All Docker named volumes with their driver (local, nfs, etc.).' },
        { label: 'Networks',            cmd: 'docker network ls',                                    category: 'Docker',            desc: 'Docker networks: ID, name, driver (bridge, host, overlay, none), and scope.' },
        { label: 'Compose status',      cmd: 'docker compose ps 2>/dev/null || docker-compose ps 2>/dev/null || echo "not in a compose project"', category: 'Docker', desc: 'Service status for all Docker Compose services defined in docker-compose.yml.' },
        { label: 'Dangling images',     cmd: 'docker images -f "dangling=true"',                    category: 'Docker',            desc: 'Untagged images not referenced by any container — safe to remove to reclaim disk.' },
        { label: 'Container inspect',   cmd: 'docker ps -q | head -1 | xargs docker inspect 2>/dev/null | head -40', category: 'Docker', desc: 'Full JSON config/state of the most recently started container.' },
        // Logs
        { label: 'Syslog tail',         cmd: 'tail -50 /var/log/syslog 2>/dev/null || journalctl -n 50 --no-pager', category: 'Logs', desc: 'Last 50 lines of the system log (syslog or systemd journal fallback).' },
        { label: 'Auth log',            cmd: 'tail -30 /var/log/auth.log 2>/dev/null',              category: 'Logs',              desc: 'Last 30 lines of the authentication log (SSH, sudo, PAM events).' },
        { label: 'Journal errors',      cmd: 'journalctl -p err -n 40 --no-pager',                  category: 'Logs',              desc: 'Last 40 error-level and above entries from the systemd journal across all services.' },
        { label: 'Journal since boot',  cmd: 'journalctl -b --no-pager | tail -60',                 category: 'Logs',              desc: 'Last 60 journal lines from the current boot session.' },
        { label: 'Kernel messages',     cmd: 'dmesg | tail -30',                                     category: 'Logs',              desc: 'Last 30 kernel ring buffer messages: hardware events, driver errors, OOM kills.' },
        { label: 'OOM kills',           cmd: 'dmesg | grep -i "oom\\|killed process"',              category: 'Logs',              desc: 'Out-of-Memory killer events where the kernel terminated processes to free RAM.' },
        { label: 'Nginx access log',    cmd: 'tail -30 /var/log/nginx/access.log 2>/dev/null || echo "not found"', category: 'Logs', desc: 'Last 30 nginx access log entries with IP, method, path, status code, and user agent.' },
        { label: 'Nginx error log',     cmd: 'tail -30 /var/log/nginx/error.log 2>/dev/null || echo "not found"',  category: 'Logs', desc: 'Last 30 nginx error log entries (config errors, upstream failures, permission issues).' },
        // Updates
        { label: 'Upgradable packages', cmd: 'apt list --upgradable 2>/dev/null',                   category: 'Updates',           desc: 'All packages with newer versions available in the configured repositories.' },
        { label: 'Update package list', cmd: 'apt update',                                           category: 'Updates',           desc: 'Refreshes the package index from all configured apt repository sources.' },
        { label: 'Security updates',    cmd: 'apt list --upgradable 2>/dev/null | grep -i security', category: 'Updates',          desc: 'Filters upgradable packages to show only those from security repositories.' },
        { label: 'Recently installed',  cmd: 'grep "install " /var/log/dpkg.log 2>/dev/null | tail -20', category: 'Updates',     desc: 'Last 20 package install events from the dpkg log with timestamps.' },
        { label: 'Installed packages',  cmd: 'dpkg -l | tail -40',                                   category: 'Updates',           desc: 'Last 40 installed Debian packages with version and architecture.' },
        // Users & Sessions
        { label: 'Who logged in',       cmd: 'who',                                                   category: 'Users & Sessions',  desc: 'Users currently logged in: username, terminal, login time, and source IP.' },
        { label: 'Who (detailed)',       cmd: 'w',                                                     category: 'Users & Sessions',  desc: 'Logged-in users plus what they are running, CPU time, and idle time.' },
        { label: 'Login shell users',   cmd: "cat /etc/passwd | awk -F: '$7 !~ /nologin|false/ {print $1, $6, $7}'", category: 'Users & Sessions', desc: 'User accounts with valid login shells (not service accounts).' },
        { label: 'Sudo group members',  cmd: 'getent group sudo wheel 2>/dev/null',                  category: 'Users & Sessions',  desc: 'Users in the sudo/wheel group who can run commands as root.' },
        { label: 'Last login times',    cmd: 'lastlog 2>/dev/null | grep -v "Never logged" | head -20', category: 'Users & Sessions', desc: 'Most recent login date/time and source for each user account.' },
        { label: 'Resource limits',     cmd: 'ulimit -a',                                             category: 'Users & Sessions',  desc: 'Resource limits for the current shell: open files, stack size, max processes, memory.' },
        { label: 'Login history',       cmd: 'last -n 30',                                            category: 'Users & Sessions',  desc: 'Last 30 login/logout events for all users including duration and source IP.' },
        // File System
        { label: 'Files changed <24h',  cmd: 'find /var /tmp /home /etc -mtime -1 -type f 2>/dev/null | head -25', category: 'File System', desc: 'Files modified in the last 24 hours in key directories. Useful for detecting recent changes.' },
        { label: 'Temp directories',    cmd: 'ls -lah /tmp && ls -lah /var/tmp',                    category: 'File System',       desc: 'Contents of /tmp and /var/tmp — often exploited by attackers and malware.' },
        { label: '/etc changes (7d)',   cmd: 'find /etc -mtime -7 -type f 2>/dev/null | head -20',  category: 'File System',       desc: 'Config files in /etc modified in the last 7 days.' },
        { label: 'Crontab files',       cmd: 'ls /etc/cron* 2>/dev/null && crontab -l 2>/dev/null', category: 'File System',       desc: 'Cron config files and the current user\'s crontab entries.' },
        { label: '/home permissions',   cmd: 'ls -la /home/',                                        category: 'File System',       desc: 'Permissions and ownership of home directories — check for world-readable entries.' },
        // Hardware
        { label: 'PCI devices',         cmd: 'lspci 2>/dev/null || echo "lspci not installed"',     category: 'Hardware',          desc: 'All PCI devices: network adapters, GPUs, storage controllers, USB hubs.' },
        { label: 'USB devices',         cmd: 'lsusb 2>/dev/null || echo "lsusb not installed"',     category: 'Hardware',          desc: 'Connected USB devices with vendor/product ID and description.' },
        { label: 'CPU temperature',     cmd: "sensors 2>/dev/null || cat /sys/class/thermal/thermal_zone*/temp 2>/dev/null | awk '{print $1/1000 \"°C\"}'", category: 'Hardware', desc: 'CPU and system component temperatures. High temps cause throttling and instability.' },
        { label: 'Hardware summary',    cmd: 'lshw -short 2>/dev/null | head -35 || echo "lshw not installed"', category: 'Hardware', desc: 'Brief hardware inventory listing CPU, RAM, disk, and network adapters with capacity.' },
        { label: 'Kernel modules',      cmd: 'lsmod | head -30',                                     category: 'Hardware',          desc: 'Currently loaded kernel modules (drivers). Unexpected modules could indicate rootkits.' },
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
        { name: 'Uptime Kuma', remotePort: 3001, localPort: 3001, icon: 'mdi-monitor-eye',       color: '#5cdd8b' },
      ],

      // Context menu
      contextMenu: {
        visible: false,
        left: 0,
        top: 0
      },
      // Quick commands palette
      quickCmdPalette: {
        visible: false,
        left: 0,
        top: 0,
        search: ''
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
      selectionTextareaText: '',
      // Selection captured at context-menu open time (clicking menu item clears window.getSelection)
      _capturedSelection: '',

      // Popout state
      isPopout: false,
      popoutWindow: null,
      popoutOpen: false,
      _popoutPoll: null,
      _bcChannel: null,

      // Sidebar collapse state
      showSessionInfo: false,
      showQuickCmds: false,
      openCmdCategories: {},
    }
  },

  mounted() {
    this.isPopout = !!(this.$route?.meta?.popout)

    // BroadcastChannel — lets main page know when popout closes / pops back
    try {
      const bc = new BroadcastChannel('sc-terminal')
      this._bcChannel = bc
      bc.onmessage = (evt) => {
        if (evt.data?.type === 'pop-back') {
          clearInterval(this._popoutPoll)
          this._popoutPoll = null
          this.popoutWindow = null
          this.popoutOpen = false
          // Merge command history from the popout session
          this._restoreHistory()
        }
      }
    } catch (_) {}

    // Restore terminal history snapshot when opening as popout
    if (this.isPopout) this._restoreFromStorage()

    this.connectWS()
    document.addEventListener('click', this.hideContextMenuOnOutsideClick)
    document.addEventListener('keydown', this.handleGlobalCopy, true)
  },

  beforeUnmount() {
    // If this is a popout, persist state and signal the opener before teardown
    if (this.isPopout) {
      this._persistToStorage()
      try {
        const bc = new BroadcastChannel('sc-terminal')
        bc.postMessage({ type: 'pop-back' })
        bc.close()
      } catch (_) {}
    }
    if (this._bcChannel) { this._bcChannel.close(); this._bcChannel = null }
    clearInterval(this._popoutPoll)
    if (this.wsConn) { this.wsConn.onclose = null; this.wsConn.close(); this.wsConn = null }
    clearInterval(this.elevationTimer)
    document.removeEventListener('click', this.hideContextMenuOnOutsideClick)
    document.removeEventListener('keydown', this.handleGlobalCopy, true)
  },

  computed: {
    filteredQuickCmds() {
      const q = this.quickCmdPalette.search.trim().toLowerCase()
      if (!q) return this.quickCmds
      return this.quickCmds.filter(c =>
        c.label.toLowerCase().includes(q) ||
        c.cmd.toLowerCase().includes(q) ||
        c.category.toLowerCase().includes(q)
      )
    },
    groupedQuickCmds() {
      const groups = {}
      for (const cmd of this.filteredQuickCmds) {
        if (!groups[cmd.category]) groups[cmd.category] = []
        groups[cmd.category].push(cmd)
      }
      return groups
    },
    quickCmdsByCategory() {
      const groups = {}
      for (const cmd of this.quickCmds) {
        if (!groups[cmd.category]) groups[cmd.category] = []
        groups[cmd.category].push(cmd)
      }
      return groups
    }
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

      // Capture selection NOW before the right-click can clear it
      this._capturedSelection = window.getSelection?.()?.toString?.() || ''

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
      if (this.quickCmdPalette.visible) {
        const palette = this.$refs.quickCmdPalette
        if (palette && !palette.contains(e.target)) {
          this.quickCmdPalette.visible = false
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
      const text = this._capturedSelection || window.getSelection()?.toString?.() || ''
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
      const text = this._capturedSelection || window.getSelection?.()?.toString?.() || ''
      this.selectionTextareaText = text.trim()
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
      this.contextMenu.visible = false
      try {
        if (!navigator.clipboard?.readText) throw new Error('Clipboard API not available')
        const raw = await navigator.clipboard.readText()
        this.insertNormalizedPaste(raw)
      } catch (err) {
        console.error('Paste failed:', err)
      }
    },

    normalizePasteText(raw) {
      return raw
        // Normalize line endings
        .replace(/\r\n/g, '\n')
        .replace(/\r/g, '\n')
        // Strip non-printable chars (keep \n and \t)
        // eslint-disable-next-line no-control-regex
        .replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '')
        // Expand tabs to spaces
        .replace(/\t/g, '  ')
        .trim()
    },

    insertNormalizedPaste(raw) {
      const normalized = this.normalizePasteText(raw)
      const lines = normalized.split('\n').map(l => l.trim()).filter(Boolean)
      if (lines.length > 1) {
        // Multi-line: join with '; ' so they run sequentially
        this.currentInput += lines.join('; ')
      } else {
        this.currentInput += normalized
      }
      this.$refs.termInput?.focus()
    },

    handlePaste(e) {
      e.preventDefault()
      const raw = e.clipboardData?.getData('text') || ''
      this.insertNormalizedPaste(raw)
    },

    openQuickCmdPalette() {
      this.contextMenu.visible = false
      // Position near where the context menu was
      const paletteW = 340
      const paletteH = 420
      const left = Math.min(this.contextMenu.left, window.innerWidth - paletteW - 12)
      const top = Math.min(this.contextMenu.top, window.innerHeight - paletteH - 12)
      this.quickCmdPalette.left = Math.max(8, left)
      this.quickCmdPalette.top = Math.max(8, top)
      this.quickCmdPalette.search = ''
      this.quickCmdPalette.visible = true
      this.$nextTick(() => this.$refs.quickCmdSearch?.focus())
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

    // ── Popout ────────────────────────────────────────────────────────────────
    _persistToStorage() {
      try {
        localStorage.setItem('sc_terminal_state', JSON.stringify({
          ts: Date.now(),
          termLines: this.termLines.slice(-600),
          commandHistory: this.commandHistory.slice(-200),
          sessionUser: this.sessionUser,
          hostname: this.hostname,
        }))
      } catch (_) {}
    },

    _restoreFromStorage() {
      try {
        const raw = localStorage.getItem('sc_terminal_state')
        if (!raw) return
        const s = JSON.parse(raw)
        // Don't restore stale data (older than 2 hours)
        if (s.ts && Date.now() - s.ts > 2 * 60 * 60 * 1000) return
        if (Array.isArray(s.termLines) && s.termLines.length) {
          this.termLines = [
            ...s.termLines,
            { type: 'out', text: '─── Session continues in popout ───', style: 'info' }
          ]
        }
        if (Array.isArray(s.commandHistory)) this.commandHistory = s.commandHistory
        if (s.sessionUser) this.sessionUser = s.sessionUser
        if (s.hostname) this.hostname = s.hostname
      } catch (_) {}
    },

    _restoreHistory() {
      // Merge command history only (called in main page after popout closes)
      try {
        const raw = localStorage.getItem('sc_terminal_state')
        if (!raw) return
        const s = JSON.parse(raw)
        if (Array.isArray(s.commandHistory) && s.commandHistory.length > this.commandHistory.length) {
          this.commandHistory = s.commandHistory
        }
      } catch (_) {}
    },

    openPopout() {
      // If already open, just focus it
      if (this.popoutWindow && !this.popoutWindow.closed) {
        this.popoutWindow.focus()
        return
      }
      this._persistToStorage()
      const w = window.open(
        '/terminal/popout',
        'sc-terminal-popout',
        `width=1060,height=740,left=${Math.round((screen.width - 1060) / 2)},top=${Math.round((screen.height - 740) / 2)},resizable=yes,scrollbars=no`
      )
      if (!w) {
        // Popup blocked — inform user
        this.termLines.push({ type: 'out', text: '[!] Popup blocked. Please allow popups for this site.', style: 'error' })
        this.$nextTick(this.scrollBottom)
        return
      }
      this.popoutWindow = w
      this.popoutOpen = true
      // Poll until the popout window is closed
      clearInterval(this._popoutPoll)
      this._popoutPoll = setInterval(() => {
        if (!this.popoutWindow || this.popoutWindow.closed) {
          clearInterval(this._popoutPoll)
          this._popoutPoll = null
          this.popoutWindow = null
          this.popoutOpen = false
          this._restoreHistory()
        }
      }, 600)
    },

    popBackIn() {
      this._persistToStorage()
      try {
        const bc = new BroadcastChannel('sc-terminal')
        bc.postMessage({ type: 'pop-back' })
        bc.close()
      } catch (_) {}
      // Focus the opener if available, then close this window
      if (window.opener && !window.opener.closed) {
        window.opener.focus()
      }
      setTimeout(() => window.close(), 80)
    },

    googleSearchUrl(item) {
      return 'https://www.google.com/search?q=' + encodeURIComponent((item.label || item.cmd) + ' linux command')
    },

    toggleCmdCategory(cat) {
      this.openCmdCategories = { ...this.openCmdCategories, [cat]: !this.openCmdCategories[cat] }
    },

    clearTerminal() {
      this.termLines = []
      this.contextMenu.visible = false
    }
  }
}
</script>

<style scoped>
/* Risk legend strip inside terminal card */
.term-risk-legend {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  padding: 4px 14px;
  background: rgba(4, 8, 16, 0.6);
  border-bottom: 1px solid #1e2d4a;
}

.term-risk-badge {
  font-size: 0.58rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  padding: 1px 5px;
  border-radius: 3px;
}

.term-risk-badge--ok   { background: rgba(34,214,124,0.12); color: #22d67c; }
.term-risk-badge--warn { background: rgba(245,166,35,0.12);  color: #f5a623; }
.term-risk-badge--err  { background: rgba(240,64,64,0.12);   color: #f04040; }
.term-risk-label { font-size: 0.6rem; color: #5a7499; }
.term-risk-sep   { width: 1px; height: 10px; background: #1e2d4a; }

/* Quick Commands accordion sidebar */
.qc-panel { border-top: 1px solid #1e2d4a; }

.qc-cat { border-bottom: 1px solid rgba(30, 45, 74, 0.45); }

.qc-cat__head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  background: none;
  border: none;
  padding: 6px 10px;
  cursor: pointer;
  color: #8aa4c8;
  font-size: 0.72rem;
  font-weight: 500;
  text-align: left;
  transition: background 0.1s;
}

.qc-cat__head:hover { background: rgba(74, 158, 255, 0.06); }

.qc-cat__arrow {
  font-size: 0.78rem;
  color: #3a5070;
  transition: transform 0.15s ease;
  flex-shrink: 0;
}

.qc-cat__arrow.open { transform: rotate(90deg); color: #4a9eff; }

.qc-cat__name { flex: 1; }

.qc-cat__count {
  font-size: 0.6rem;
  background: rgba(74, 158, 255, 0.1);
  color: #4a9eff;
  padding: 1px 5px;
  border-radius: 8px;
  flex-shrink: 0;
}

.qc-cat__items { background: rgba(4, 8, 16, 0.45); }

.qc-item {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 1px 6px 1px 18px;
  border-bottom: 1px solid rgba(30, 45, 74, 0.3);
}

.qc-item:last-child { border-bottom: none; }

.qc-item__run {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  padding: 4px 2px;
  color: #8aa4c8;
  font-size: 0.69rem;
  text-align: left;
  cursor: pointer;
  min-width: 0;
  overflow: hidden;
}

.qc-item__run span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.qc-item__run:hover:not(:disabled) { color: #c9d8f0; }
.qc-item__run:disabled { opacity: 0.35; cursor: not-allowed; }

.qc-item__meta {
  display: flex;
  align-items: center;
  gap: 1px;
  flex-shrink: 0;
}

.qc-item__meta-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  background: none;
  border: none;
  border-radius: 4px;
  color: #3a5070;
  font-size: 0.72rem;
  cursor: pointer;
  text-decoration: none;
  transition: color 0.1s, background 0.1s;
}

.qc-item__meta-btn:hover { color: #4a9eff; background: rgba(74, 158, 255, 0.1); }
.qc-item__search-btn:hover { color: #22d67c; background: rgba(34, 214, 124, 0.1); }

/* Popout root — applied when rendered outside MainLayout */
.terminal-popout-root {
  min-height: 100vh;
  background: #040810;
  padding: 10px;
  box-sizing: border-box;
}

/* Title-bar window action button (popout / pop-back-in) */
.terminal-window-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 22px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 5px;
  color: #3a5070;
  font-size: 0.82rem;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, background 0.15s ease;
  margin-left: 4px;
}

.terminal-window-btn:hover {
  color: #4a9eff;
  border-color: rgba(74, 158, 255, 0.3);
  background: rgba(74, 158, 255, 0.08);
}

.terminal-window-btn.active {
  color: #22d67c;
  border-color: rgba(34, 214, 124, 0.3);
  background: rgba(34, 214, 124, 0.07);
}

.terminal-popback-btn {
  color: #f5a623;
}

.terminal-popback-btn:hover {
  color: #f5a623;
  border-color: rgba(245, 166, 35, 0.35);
  background: rgba(245, 166, 35, 0.1);
}

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

/* Quick Commands Palette */
.quick-cmd-palette {
  position: fixed;
  width: 340px;
  max-height: min(480px, 70vh);
  background: #0d1321;
  border: 1px solid #1e2d4a;
  border-radius: 10px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: contextMenuFadeIn 0.18s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.quick-cmd-palette__header {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 9px 12px;
  border-bottom: 1px solid #1e2d4a;
  font-size: 0.78rem;
  font-weight: 600;
  color: #c9d8f0;
  flex-shrink: 0;
}

.quick-cmd-palette__close {
  margin-left: auto;
  background: none;
  border: none;
  color: #5a7499;
  font-size: 0.9rem;
  cursor: pointer;
  padding: 0 2px;
  display: flex;
  align-items: center;
}

.quick-cmd-palette__close:hover { color: #c9d8f0; }

.quick-cmd-palette__search-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid #1e2d4a;
  flex-shrink: 0;
  color: #5a7499;
}

.quick-cmd-palette__search {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: #c9d8f0;
  font-size: 0.78rem;
  font-family: monospace;
}

.quick-cmd-palette__search::placeholder { color: #3a5070; }

.quick-cmd-palette__list {
  overflow-y: auto;
  flex: 1;
  scrollbar-width: thin;
  scrollbar-color: #1e2d4a transparent;
}

.quick-cmd-palette__category {
  padding: 5px 12px 3px;
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #4a9eff;
  opacity: 0.7;
  position: sticky;
  top: 0;
  background: #0d1321;
}

.quick-cmd-palette__item {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 6px 12px;
  cursor: pointer;
  transition: background 0.1s ease;
  border-radius: 0;
}

.quick-cmd-palette__item:hover {
  background: rgba(74, 158, 255, 0.1);
}

.quick-cmd-palette__label {
  font-size: 0.75rem;
  color: #c9d8f0;
  font-weight: 500;
}

.quick-cmd-palette__cmd {
  font-size: 0.65rem;
  color: #5a7499;
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.quick-cmd-palette__empty {
  padding: 24px 12px;
  text-align: center;
  font-size: 0.75rem;
  color: #3a5070;
}


.terminal-context-menu {
  position: fixed;
  width: max-content;
  min-width: 160px;
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
  gap: 6px;
  padding: 5px 10px;
  color: #333;
  font-size: 12px;
  font-weight: 400;
  cursor: pointer;
  transition: all 0.15s ease;
  border-radius: 4px;
  margin: 1px;
  white-space: nowrap;
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

