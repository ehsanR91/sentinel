<template>
  <div>
    <PageHeader title="Settings" icon="mdi mdi-cog" :items="[{text:'Settings',active:true,icon:'mdi mdi-cog-outline'}]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" @click="saveSettings">
          <i class="mdi mdi-content-save me-1"></i> Save
        </button>
      </template>
    </PageHeader>

    <div class="row g-3">
      <!-- General -->
      <div class="col-lg-4">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-cog me-2" style="color:#4a9eff"></i>General</h6></div>
          <div class="card-body">
            <div class="mb-3">
              <label class="form-label">Server Hostname</label>
              <input v-model="settings.hostname" class="form-control font-mono" />
            </div>
            <div class="mb-3">
              <label class="form-label">Listen Address</label>
              <select v-model="settings.listenAddr" class="form-select">
                <option value="127.0.0.1:8080">127.0.0.1:8080 (localhost only — recommended)</option>
                <option value="0.0.0.0:8080">0.0.0.0:8080 (public — requires TLS)</option>
              </select>
            </div>
            <div class="mb-3">
              <label class="form-label">TLS / HTTPS</label>
              <div class="form-check form-switch">
                <input v-model="settings.tls" class="form-check-input" type="checkbox" />
                <label class="form-check-label" style="font-size:0.8rem;color:#8aa4c8">Enable TLS</label>
              </div>
            </div>
            <div class="mb-3">
              <label class="form-label">Admin Email</label>
              <input v-model="settings.adminEmail" type="email" class="form-control" placeholder="admin@yourdomain.com" />
              <div style="font-size:0.72rem;color:#5a7499;margin-top:4px">Used for alert notifications</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Security -->
      <div class="col-lg-4">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-shield-lock me-2" style="color:#22d67c"></i>Security</h6></div>
          <div class="card-body">
            <div class="mb-3">
              <label class="form-label">Max Login Attempts (per 10 min / IP)</label>
              <input v-model.number="settings.brute_force_threshold" type="number" min="3" max="20" class="form-control" />
            </div>
            <div class="mb-3">
              <label class="form-label">IP Allowlist (one per line)</label>
              <textarea v-model="settings.ipAllowlist" class="form-control font-mono" rows="3" placeholder="192.168.1.0/24&#10;10.0.0.0/8"></textarea>
              <div style="font-size:0.72rem;color:#5a7499;margin-top:4px">Leave empty to allow all (not recommended for public mode)</div>
            </div>
            <div class="sc-divider my-3"></div>
            <button class="btn btn-sc-primary btn-sm" @click="saveSecurity">
              <i class="mdi mdi-content-save me-1"></i> Save Security Settings
            </button>
          </div>
        </div>
      </div>

      <!-- 2FA Setup -->
      <div class="col-lg-4">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-two-factor-authentication me-2" style="color:#a78bfa"></i>Two-Factor Authentication</h6></div>
          <div class="card-body">
            <div v-if="me.totp_enabled" class="mb-3">
              <div class="d-flex align-items-center gap-2 mb-3">
                <span class="badge badge-online">Enabled</span>
                <span style="font-size:0.8rem;color:#8aa4c8">TOTP 2FA is active for your account</span>
              </div>
              <div v-if="!showDisable2FA">
                <button class="btn btn-sm" style="background:rgba(240,64,64,0.12);color:#f04040;border:1px solid rgba(240,64,64,0.2)" @click="showDisable2FA=true">
                  <i class="mdi mdi-shield-off-outline me-1"></i> Disable 2FA
                </button>
              </div>
              <div v-else>
                <label class="form-label" style="font-size:0.8rem">Enter current authenticator code to disable:</label>
                <div class="input-group mb-2">
                  <input v-model="disableCode" type="text" maxlength="6" class="form-control font-mono" placeholder="000000" style="letter-spacing:.3em" />
                  <button class="btn btn-sm" style="background:rgba(240,64,64,0.15);color:#f04040;border:1px solid rgba(240,64,64,0.2)" @click="disable2FA">Confirm Disable</button>
                </div>
                <button class="btn btn-link btn-sm p-0" style="font-size:0.75rem;color:#5a7499" @click="showDisable2FA=false">Cancel</button>
              </div>
            </div>

            <div v-else>
              <div class="d-flex align-items-center gap-2 mb-3">
                <span class="badge badge-offline">Disabled</span>
                <span style="font-size:0.8rem;color:#8aa4c8">Protect your account with an authenticator app</span>
              </div>

              <div v-if="!totpSetup.secret">
                <button class="btn btn-sc-primary btn-sm" @click="initSetup2FA" :disabled="totpSetup.loading">
                  <span v-if="totpSetup.loading" class="spinner-border spinner-border-sm me-1"></span>
                  <i v-else class="mdi mdi-qrcode me-1"></i>
                  Set Up 2FA
                </button>
              </div>

              <div v-else>
                <p style="font-size:0.78rem;color:#8aa4c8;margin-bottom:.75rem">
                  Scan with Google Authenticator, Authy, or any TOTP app:
                </p>
                <div class="text-center mb-3">
                  <canvas ref="qrCanvas" style="border-radius:8px;background:#fff;padding:8px"></canvas>
                </div>
                <div class="mb-2" style="font-size:0.72rem;color:#5a7499">
                  Can't scan? Enter this code manually:
                </div>
                <div class="font-mono p-2 rounded mb-3" style="background:#0d1321;font-size:0.8rem;color:#22d67c;letter-spacing:.15em;word-break:break-all">
                  {{ totpSetup.secret }}
                </div>
                <label class="form-label" style="font-size:0.8rem">Enter code from app to activate:</label>
                <div class="input-group mb-2">
                  <input v-model="totpSetup.verifyCode" type="text" maxlength="6" class="form-control font-mono" placeholder="000000" style="letter-spacing:.3em" />
                  <button class="btn btn-sc-primary btn-sm" @click="enable2FA">Activate</button>
                </div>
                <button class="btn btn-link btn-sm p-0" style="font-size:0.75rem;color:#5a7499" @click="totpSetup.secret=''">Cancel</button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Secret Link -->
      <div class="col-lg-4">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-link-lock me-2" style="color:#f5a623"></i>Secret Link Gate</h6></div>
          <div class="card-body">
            <p style="font-size:0.78rem;color:#8aa4c8;margin-bottom:1rem">
              Visitors must navigate to the secret URL first. Any other URL returns a bare 403 page, hiding the login screen from port scanners.
            </p>
            <div class="mb-3">
              <label class="form-label">Secret Path</label>
              <div class="input-group">
                <span class="input-group-text font-mono" style="font-size:0.8rem;color:#5a7499">http://{{ hostDisplay }}/</span>
                <input v-model="settings.secret_path" type="text" class="form-control font-mono" placeholder="sentinel-core" />
                <span class="input-group-text font-mono" style="font-size:0.8rem;color:#5a7499">/</span>
              </div>
              <div style="font-size:0.72rem;color:#5a7499;margin-top:4px">Avoid common words. Use something hard to guess.</div>
            </div>
            <div class="mb-3">
              <label class="form-label">Gate Cookie Expiry</label>
              <select v-model="settings.gate_expiry_days" class="form-select">
                <option value="0">Session only (expires when browser closes)</option>
                <option value="1">1 day</option>
                <option value="7">7 days</option>
                <option value="30">30 days</option>
              </select>
            </div>
            <div class="mb-3 p-2 rounded" style="background:rgba(245,166,35,0.08);border:1px solid rgba(245,166,35,0.15)">
              <div style="font-size:0.75rem;color:#f5a623;margin-bottom:.25rem"><i class="mdi mdi-bookmark-outline me-1"></i>Your secret URL (bookmark this):</div>
              <code style="font-size:0.8rem;color:#f5a623">{{ secretURL }}</code>
            </div>
            <button class="btn btn-sc-primary btn-sm" @click="saveGateSettings">
              <i class="mdi mdi-content-save me-1"></i> Save Gate Settings
            </button>
          </div>
        </div>
      </div>

      <!-- Alerting -->
      <div class="col-lg-6">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-bell-ring me-2" style="color:#f5a623"></i>Alert Notifications</h6></div>
          <div class="card-body">
            <!-- Email -->
            <div class="mb-3 p-3 rounded" style="background:#0d1321;border:1px solid #1e2d4a">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <span style="font-weight:600;font-size:0.82rem;color:#c9d8f0"><i class="mdi mdi-email me-2" style="color:#4a9eff"></i>Email</span>
                <div class="form-check form-switch mb-0">
                  <input v-model="settings.email_alerts_enabled" class="form-check-input" type="checkbox" />
                </div>
              </div>
              <div v-if="settings.email_alerts_enabled">
                <input v-model="settings.smtp_host" class="form-control mb-2" placeholder="smtp.yourhost.com" />
                <div class="row g-2 mb-2">
                  <div class="col-8"><input v-model="settings.smtp_user" class="form-control" placeholder="SMTP username / from address" /></div>
                  <div class="col-4"><input v-model="settings.smtp_port" class="form-control" placeholder="587" /></div>
                </div>
                <input v-model="settings.smtp_pass" type="password" class="form-control mb-2" placeholder="SMTP password" />
                <input v-model="settings.alert_email" class="form-control mb-2" placeholder="Alert recipient email" />
                <button class="btn btn-sm btn-sc-primary" @click="saveEmail">
                  <i class="mdi mdi-content-save me-1"></i>Save Email Settings
                </button>
              </div>
            </div>

            <!-- Telegram -->
            <div class="mb-3 p-3 rounded" style="background:#0d1321;border:1px solid #1e2d4a">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <span style="font-weight:600;font-size:0.82rem;color:#c9d8f0"><i class="mdi mdi-send me-2" style="color:#22d3ee"></i>Telegram</span>
                <div class="form-check form-switch mb-0">
                  <input v-model="settings.telegramEnabled" class="form-check-input" type="checkbox" />
                </div>
              </div>
              <div v-if="settings.telegramEnabled">
                <input v-model="settings.telegramToken" class="form-control mb-2 font-mono" placeholder="Bot Token" />
                <input v-model="settings.telegramChatId" class="form-control font-mono" placeholder="Chat ID" />
              </div>
            </div>

            <!-- Webhook -->
            <div class="p-3 rounded" style="background:#0d1321;border:1px solid #1e2d4a">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <span style="font-weight:600;font-size:0.82rem;color:#c9d8f0"><i class="mdi mdi-webhook me-2" style="color:#a78bfa"></i>Webhook</span>
                <div class="form-check form-switch mb-0">
                  <input v-model="settings.webhookEnabled" class="form-check-input" type="checkbox" />
                </div>
              </div>
              <div v-if="settings.webhookEnabled">
                <input v-model="settings.webhookUrl" class="form-control" placeholder="https://hooks.slack.com/..." />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Database + System -->
      <div class="col-lg-6">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-database me-2" style="color:#a78bfa"></i>Database & System</h6></div>
          <div class="card-body">
            <!-- Stats -->
            <div v-if="dbStats" class="row g-2 mb-3">
              <div class="col-4 text-center p-2 rounded" style="background:#0d1321">
                <div style="font-size:1.3rem;font-weight:700;color:#4a9eff">{{ dbStats.login_attempts }}</div>
                <div style="font-size:0.65rem;color:#5a7499">Login attempts</div>
              </div>
              <div class="col-4 text-center p-2 rounded" style="background:#0d1321">
                <div style="font-size:1.3rem;font-weight:700;color:#f5a623">{{ dbStats.alerts }}</div>
                <div style="font-size:0.65rem;color:#5a7499">Alerts</div>
              </div>
              <div class="col-4 text-center p-2 rounded" style="background:#0d1321">
                <div style="font-size:1.3rem;font-weight:700;color:#f04040">{{ dbStats.manual_bans }}</div>
                <div style="font-size:0.65rem;color:#5a7499">Manual bans</div>
              </div>
            </div>

            <div class="sc-divider my-3"></div>

            <div class="mb-3 p-3 rounded" style="background:#0d1321;border:1px solid #1e2d4a">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <span style="font-weight:600;font-size:0.82rem;color:#c9d8f0"><i class="mdi mdi-key-variant me-2" style="color:#f5a623"></i>Master Key</span>
                <span class="badge" style="background:rgba(34,214,124,0.12);color:#22d67c;font-size:0.65rem">Enabled</span>
              </div>
              <div class="mb-2" style="font-size:0.75rem;color:#8aa4c8">
                Path: <span class="font-mono" style="color:#c9d8f0">{{ settings.secrets_key_path || 'Not configured' }}</span>
              </div>
              <div style="font-size:0.75rem;color:#8aa4c8">
                Last rotation: <span style="color:#f5a623">{{ formatRotationTime(settings.last_master_key_rotation) }}</span>
              </div>
            </div>

            <div class="mb-3">
              <label class="form-label">Prune old records</label>
              <div class="d-flex gap-2 align-items-center mb-2">
                <select v-model="pruneTarget" class="form-select form-select-sm" style="flex:1">
                  <option value="login_attempts">Login attempts</option>
                  <option value="alerts">Alerts</option>
                </select>
                <select v-model="pruneDays" class="form-select form-select-sm" style="width:130px">
                  <option value="7">Older than 7d</option>
                  <option value="30">Older than 30d</option>
                  <option value="90">Older than 90d</option>
                  <option value="0">Delete ALL</option>
                </select>
                <button class="btn btn-sm btn-sc-danger" :disabled="pruning" @click="doPrune">
                  <span v-if="pruning" class="spinner-border spinner-border-sm me-1"></span>
                  <i v-else class="mdi mdi-delete-sweep me-1"></i>Prune
                </button>
              </div>
            </div>

            <div class="d-flex gap-2 flex-wrap">
              <button class="btn btn-sm" style="background:rgba(167,139,250,0.12);color:#a78bfa;border:1px solid rgba(167,139,250,0.2)" :disabled="loadingDbStats" @click="loadDbStats">
                <i class="mdi mdi-refresh me-1"></i> Refresh Stats
              </button>
              <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;border:1px solid rgba(74,158,255,0.2)" @click="downloadDb">
                <i class="mdi mdi-database-export me-1"></i> Export DB
              </button>
              <label class="btn btn-sm mb-0" style="background:rgba(34,214,124,0.12);color:#22d67c;border:1px solid rgba(34,214,124,0.2);cursor:pointer" :class="{ disabled: importing }">
                <span v-if="importing" class="spinner-border spinner-border-sm me-1"></span>
                <i v-else class="mdi mdi-database-import me-1"></i> Import DB
                <input ref="dbImportInput" type="file" accept=".db" class="d-none" @change="importDb" />
              </label>
              <button class="btn btn-sm" style="background:rgba(240,64,64,0.12);color:#f04040;border:1px solid rgba(240,64,64,0.2)" @click="confirmRestart">
                <i class="mdi mdi-restart me-1"></i> Restart Daemon
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Lock Screen PIN -->
      <div class="col-lg-4">
        <div class="card h-100">
          <div class="card-header"><h6><i class="mdi mdi-lock-outline me-2" style="color:#22d3ee"></i>Lock Screen</h6></div>
          <div class="card-body">
            <p style="font-size:0.78rem;color:#8aa4c8;margin-bottom:1rem">
              Set a 6-digit PIN to quickly lock your screen. Press <kbd>Space</kbd> to lock when enabled.
            </p>
            <div class="form-check form-switch mb-3">
              <input v-model="lockEnabled" class="form-check-input" type="checkbox" @change="toggleLock" />
              <label class="form-check-label" style="font-size:0.8rem">Enable lock screen</label>
            </div>
            <div v-if="lockEnabled">
              <div class="mb-2">
                <label class="form-label">New PIN (6 digits)</label>
                <input v-model="newPin" type="password" inputmode="numeric" maxlength="6" class="form-control font-mono" placeholder="••••••" style="letter-spacing:.5em;width:140px" />
              </div>
              <div class="mb-3">
                <label class="form-label">Confirm PIN</label>
                <input v-model="confirmPin" type="password" inputmode="numeric" maxlength="6" class="form-control font-mono" placeholder="••••••" style="letter-spacing:.5em;width:140px" />
              </div>
              <button class="btn btn-sm btn-sc-primary" :disabled="settingPin" @click="saveLockPin">
                <span v-if="settingPin" class="spinner-border spinner-border-sm me-1"></span>
                <i v-else class="mdi mdi-lock-check me-1"></i>
                {{ lockPinSet ? 'Change PIN' : 'Set PIN' }}
              </button>
              <button v-if="lockPinSet" class="btn btn-sm btn-sc-danger ms-2" @click="clearLockPin">
                <i class="mdi mdi-lock-open-outline me-1"></i> Remove PIN
              </button>
              <div v-if="lockPinSet" class="mt-2" style="font-size:0.72rem;color:#22d67c">
                <i class="mdi mdi-check-circle me-1"></i> PIN is set — press Space to lock
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import QRCode from 'qrcode'

export default {
  name: 'SettingsPage',
  components: { PageHeader },
  data() {
    return {
      settings: {
        brute_force_threshold: 5,
        ipAllowlist: '',
        secret_path: 'sentinel-core',
        gate_expiry_days: '0',
        emailEnabled: false,
        smtp_host: '',
        smtp_port: '587',
        smtp_user: '',
        smtp_pass: '',
        alert_email: '',
        secrets_key_path: '',
        last_master_key_rotation: ''
      },
      me: { totp_enabled: false },
      totpSetup: { secret: '', otpauthUrl: '', verifyCode: '', loading: false },
      showDisable2FA: false,
      disableCode: '',
      // DB admin
      dbStats: null,
      loadingDbStats: false,
      pruneTarget: 'login_attempts',
      pruneDays: 30,
      pruning: false,
      importing: false,
      // Lock screen
      lockEnabled: false,
      lockPinSet: false,
      newPin: '',
      confirmPin: '',
      settingPin: false
    }
  },

  computed: {
    hostDisplay() {
      return window.location.host
    },
    secretURL() {
      const proto = window.location.protocol
      return `${proto}//${window.location.host}/${this.settings.secret_path || 'sentinel-core'}/`
    }
  },

  async mounted() {
    await this.loadSettings()
    await this.loadMe()
    await this.loadDbStats()
    await this.loadLockSettings()
  },

  methods: {
    async loadSettings() {
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getSettings()
        Object.assign(this.settings, data)
      } catch (_) {}
    },

    async loadMe() {
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getMe()
        this.me = data
      } catch (_) {}
    },

    async saveSettings() {
      try {
        const api = (await import('@/services/api')).default
        // Only send valid setting keys to backend
        const validKeys = ['secret_path', 'gate_expiry_days', 'smtp_host', 'smtp_port', 'smtp_user', 'smtp_pass', 'alert_email', 'brute_force_threshold', 'email_alerts_enabled']
        const settingsToSend = {}
        validKeys.forEach(key => {
          if (this.settings[key] !== undefined) {
            settingsToSend[key] = this.settings[key]
          }
        })
        await api.updateSettings(settingsToSend)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Settings saved', showConfirmButton: false, timer: 2000 })
      } catch (_) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to save', showConfirmButton: false, timer: 2000 })
      }
    },

    async saveSecurity() {
      await this.saveSettings()
    },

    async saveGateSettings() {
      await this.saveSettings()
    },

    async saveEmail() {
      await this.saveSettings()
    },

    formatRotationTime(ts) {
      if (!ts) return 'Never rotated'
      const n = Number(ts)
      if (!Number.isFinite(n) || n <= 0) return 'Unknown'
      try {
        return new Date(n * 1000).toLocaleString()
      } catch (_) {
        return 'Unknown'
      }
    },

    async initSetup2FA() {
      this.totpSetup.loading = true
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.setup2fa()
        this.totpSetup.secret = data.secret
        this.totpSetup.otpauthUrl = data.otpauth_url
        await this.$nextTick()
        QRCode.toCanvas(this.$refs.qrCanvas, data.otpauth_url, {
          width: 200, margin: 1,
          color: { dark: '#000000', light: '#ffffff' }
        })
      } catch (_) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Could not generate QR code', showConfirmButton: false, timer: 2000 })
      } finally {
        this.totpSetup.loading = false
      }
    },

    async enable2FA() {
      if (!this.totpSetup.verifyCode) return
      try {
        const api = (await import('@/services/api')).default
        await api.enable2fa(this.totpSetup.secret, this.totpSetup.verifyCode)
        this.me.totp_enabled = true
        this.totpSetup = { secret: '', otpauthUrl: '', verifyCode: '', loading: false }

        if (this.$route?.query?.return_to === 'terminal') {
          this.$swal({ toast: true, position: 'top-end', icon: 'success', title: '2FA enabled. Returning to Terminal…', showConfirmButton: false, timer: 1800 })
          setTimeout(() => {
            this.$router.push({ path: '/terminal' })
          }, 400)
          return
        }

        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: '2FA enabled', showConfirmButton: false, timer: 2000 })
      } catch (err) {
        const msg = err.response?.data?.error || 'Invalid code'
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: msg, showConfirmButton: false, timer: 2500 })
      }
    },

    async disable2FA() {
      if (!this.disableCode) return
      try {
        const api = (await import('@/services/api')).default
        await api.disable2fa(this.disableCode)
        this.me.totp_enabled = false
        this.showDisable2FA = false
        this.disableCode = ''
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: '2FA disabled', showConfirmButton: false, timer: 2000 })
      } catch (err) {
        const msg = err.response?.data?.error || 'Invalid code'
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: msg, showConfirmButton: false, timer: 2500 })
      }
    },

    confirmRestart() {
      this.$swal({ title: 'Restart SentinelCore daemon?', icon: 'warning', showCancelButton: true, confirmButtonColor: '#f04040' }).then(r => {
        if (r.isConfirmed) this.$swal({ toast: true, position: 'top-end', icon: 'info', title: 'Daemon restarting…', showConfirmButton: false, timer: 2000 })
      })
    },

    async loadDbStats() {
      this.loadingDbStats = true
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getDbStats()
        this.dbStats = data
      } catch (_) {} finally {
        this.loadingDbStats = false
      }
    },

    async doPrune() {
      const label = this.pruneDays > 0 ? `records older than ${this.pruneDays} days` : 'ALL records'
      const r = await this.$swal({
        title: `Prune ${this.pruneTarget}?`,
        text: `This will permanently delete ${label}.`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#f04040',
        confirmButtonText: 'Delete'
      })
      if (!r.isConfirmed) return
      this.pruning = true
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.pruneDb(this.pruneTarget, this.pruneDays)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Deleted ${data.deleted} records`, showConfirmButton: false, timer: 3000 })
        await this.loadDbStats()
      } catch (_) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Prune failed', showConfirmButton: false, timer: 2000 })
      } finally {
        this.pruning = false
      }
    },

    async downloadDb() {
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.exportDb()
        const url = URL.createObjectURL(new Blob([data], { type: 'application/octet-stream' }))
        const a = document.createElement('a')
        a.href = url
        a.download = 'sentinelcore.db'
        a.click()
        URL.revokeObjectURL(url)
      } catch (_) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Export failed', showConfirmButton: false, timer: 2000 })
      }
    },

    async importDb(evt) {
      const file = evt.target.files?.[0]
      if (!file) return
      const result = await this.$swal({
        title: 'Import Database?',
        text: 'This will REPLACE the current database with the uploaded file. All existing data will be overwritten.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Replace & Import',
        confirmButtonColor: '#f04040'
      })
      if (!result.isConfirmed) {
        this.$refs.dbImportInput.value = ''
        return
      }
      this.importing = true
      try {
        const formData = new FormData()
        formData.append('db', file)
        const api = (await import('@/services/api')).default
        await api.importDb(formData)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Database imported', showConfirmButton: false, timer: 2500 })
        await this.loadDbStats()
      } catch (err) {
        this.$swal({ icon: 'error', title: 'Import failed', text: err.response?.data?.error || 'Could not import database' })
      } finally {
        this.importing = false
        this.$refs.dbImportInput.value = ''
      }
    },

    async loadLockSettings() {
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getLockSettings()
        this.lockEnabled = data.enabled || false
        this.lockPinSet = data.pinSet || false
        this.$store.dispatch('lock/setLockState', {
          enabled: this.lockEnabled,
          pinSet: this.lockPinSet
        })
      } catch (err) {
        console.error('Failed to load lock settings:', err)
      }
    },

    async toggleLock() {
      if (!this.lockEnabled) {
        // User turned lock OFF — delete the pin from the backend.
        try {
          const api = (await import('@/services/api')).default
          await api.clearLockPin()
          this.lockPinSet = false
          this.$store.dispatch('lock/clearLock')
        } catch (err) {
          console.error('Failed to disable lock:', err)
          this.lockEnabled = true // revert toggle on error
        }
      }
      // Turned ON: just reveal the PIN entry form — user sets PIN via "Set PIN" button.
    },

    async saveLockPin() {
      if (this.newPin.length !== 6 || !/^\d{6}$/.test(this.newPin)) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'PIN must be exactly 6 digits', showConfirmButton: false, timer: 2000 })
        return
      }
      if (this.newPin !== this.confirmPin) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'PINs do not match', showConfirmButton: false, timer: 2000 })
        return
      }
      this.settingPin = true
      try {
        const api = (await import('@/services/api')).default
        await api.saveLockPin(this.newPin, true)
        this.lockEnabled = true
        this.lockPinSet = true
        this.$store.dispatch('lock/setLockState', {
          enabled: this.lockEnabled,
          pinSet: this.lockPinSet
        })
        this.newPin = ''
        this.confirmPin = ''
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Lock screen PIN set successfully', showConfirmButton: false, timer: 3000 })
      } catch (err) {
        const msg = err.response?.data?.error || 'Failed to save PIN'
        console.error('Failed to save PIN:', err)
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: msg, showConfirmButton: false, timer: 3000 })
      } finally {
        this.settingPin = false
      }
    },

    async clearLockPin() {
      const r = await this.$swal({ title: 'Remove lock PIN?', icon: 'question', showCancelButton: true, confirmButtonText: 'Remove' })
      if (!r.isConfirmed) return
      try {
        const api = (await import('@/services/api')).default
        await api.clearLockPin()
        this.lockPinSet = false
        this.lockEnabled = false
        this.$store.dispatch('lock/clearLock')
        this.$swal({ toast: true, position: 'top-end', icon: 'info', title: 'Lock PIN removed', showConfirmButton: false, timer: 2000 })
      } catch (err) {
        console.error('Failed to clear PIN:', err)
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to remove PIN', showConfirmButton: false, timer: 2000 })
      }
    },

    downloadBackup() {
      this.downloadDb()
    }
  }
}
</script>
