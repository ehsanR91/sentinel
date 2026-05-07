<template>
<div class="sc-view sc-view-services">
  <PageHeader title="Services" icon="mdi mdi-server" :items="[{ text: 'Services', active: true, icon: 'mdi mdi-server' }]">
    <template #actions>
      <button class="btn btn-sm btn-sc-primary" :disabled="loading" @click="loadAll">
        <i :class="`mdi ${loading ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>
        Refresh
      </button>
    </template>
  </PageHeader>

  <!-- Live Service Installation Log Window -->
  <div v-if="showInstallLogWindow" class="card mb-4 service-install-log-card">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6><i class="mdi mdi-terminal me-2" style="color:var(--sc-cyan)"></i>Live Installation Log - {{ installingService }}</h6>
      <div class="d-flex gap-2">
        <span class="status-dot" :class="installing ? 'online' : 'offline'"></span>
        <button class="btn btn-sm" style="background:rgba(90,116,153,.1);color:#5a7499" @click="showInstallLogWindow=false">
          <i class="mdi mdi-close"></i>
        </button>
      </div>
    </div>
    <div class="card-body p-0">
      <div class="service-install-log-window" ref="serviceInstallLogWindow">
        <div v-for="(line, idx) in installLogs" :key="idx" class="service-install-log-line font-mono">
          <span class="log-timestamp">{{ line.ts }}</span>
          <span :class="`log-content ${line.type}`">{{ line.text }}</span>
        </div>
        <div v-if="installing" class="service-install-log-line font-mono log-cursor">
          <span class="log-timestamp">—</span>
          <span class="log-content">running...</span>
        </div>
      </div>
    </div>
    <div class="card-footer d-flex align-items-center justify-content-between">
      <button class="btn btn-sm" style="background:rgba(74,158,255,.1);color:#4a9eff;font-size:.72rem" @click="clearInstallLogs">
        <i class="mdi mdi-delete-empty me-1"></i>Clear
      </button>
      <span style="font-size:.72rem;color:var(--sc-text-muted)">{{ installLogs.length }} lines</span>
    </div>
  </div>

  <div class="row g-3 mb-4">
    <div class="col-xl-3 col-md-6">
      <StatCard label="Total" :value="services.length" sub="managed" icon="mdi mdi-cog-box" icon-color="#4a9eff" icon-bg="rgba(74,158,255,.12)" />
    </div>
    <div class="col-xl-3 col-md-6">
      <StatCard label="Installed" :value="installedCount" sub="packages" icon="mdi mdi-package-variant-closed-check" icon-color="#22d67c" icon-bg="rgba(34,214,124,.12)" />
    </div>
    <div class="col-xl-3 col-md-6">
      <StatCard label="Running" :value="runningCount" sub="active units" icon="mdi mdi-play-circle" icon-color="#22d67c" icon-bg="rgba(34,214,124,.12)" />
    </div>
    <div class="col-xl-3 col-md-6">
      <StatCard label="Missing" :value="missingCount" sub="needs install" icon="mdi mdi-alert-circle" icon-color="#f5a623" icon-bg="rgba(245,166,35,.12)" />
    </div>
  </div>

  <div class="card sc-panel-card mb-4">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6><i class="mdi mdi-cog-refresh-outline me-2" style="color:var(--sc-blue)"></i>Managed Services</h6>
      <input v-model="q" class="form-control form-control-sm" placeholder="Filter services..." style="width:200px" />
    </div>
    <div class="card-body p-0">
      <table class="table mb-0">
        <thead>
          <tr>
            <th>Service</th>
            <th>Package</th>
            <th>Category</th>
            <th>Status</th>
            <th>Config</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!filtered.length">
            <td colspan="6" class="text-center py-4 sc-text-muted">No services match your filter</td>
          </tr>
          <tr v-for="s in filtered" :key="s.name">
            <td>
              <div class="d-flex align-items-center gap-2">
                <span class="status-dot" :class="s.running ? 'online' : (s.installed ? 'warn' : 'offline')"></span>
                <div>
                  <div class="service-label">{{ s.label }}</div>
                  <div class="service-name font-mono">{{ s.name }}</div>
                </div>
                <button class="btn btn-sm service-info-btn" @click="showServiceInfo(s)">
                  <i class="mdi mdi-information-outline"></i>
                </button>
                <button class="btn btn-sm service-config-btn" @click="openConfigEditor(s)" :title="'Edit config: ' + s.config">
                  <i class="mdi mdi-file-document-edit"></i>
                </button>
              </div>
            </td>
            <td class="font-mono sc-text-secondary">{{ s.package }}</td>
            <td>
              <span class="badge category-badge" :class="getCategoryClass(s.name)">{{ getCategory(s.name) }}</span>
            </td>
            <td>
              <span class="badge rounded-pill" :class="statusBadge(s)">{{ statusText(s) }}</span>
            </td>
            <td class="font-mono sc-text-muted">{{ s.config || '—' }}</td>
            <td>
              <div class="d-flex gap-1 flex-wrap">
                <button class="btn btn-sm btn-install" :disabled="busy[s.name] || s.installed" @click="installService(s)">
                  <i class="mdi mdi-package-down me-1"></i>Install
                </button>
                <button class="btn btn-sm btn-start" :disabled="busy[s.name] || !s.installed || isGhost(s)" :title="actionTitle(s, 'start')" @click="act(s, 'start')">
                  Start
                </button>
                <button class="btn btn-sm btn-restart" :disabled="busy[s.name] || !s.installed || isGhost(s)" :title="actionTitle(s, 'restart')" @click="act(s, 'restart')">
                  Restart
                </button>
                <button class="btn btn-sm btn-stop" :disabled="busy[s.name] || !s.installed || isGhost(s)" :title="actionTitle(s, 'stop')" @click="act(s, 'stop')">
                  Stop
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>

  <div class="card sc-panel-card">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6><i class="mdi mdi-tune me-2" style="color:var(--sc-purple)"></i>Service Configuration Profiles</h6>
      <button class="btn btn-sm btn-sc-primary" :disabled="cfgSaving" @click="saveCfg">
        <i :class="`mdi ${cfgSaving ? 'mdi-loading mdi-spin' : 'mdi-content-save'} me-1`"></i>Save Config
      </button>
    </div>
    <div class="card-body">
      <div class="row g-3">
        <div v-for="key in cfgKeys" :key="key" class="col-xl-4 col-md-6">
          <label class="form-label sc-form-label">{{ prettyKey(key) }}</label>
          <input v-model="cfg[key]" class="form-control form-control-sm" :placeholder="key" />
        </div>
      </div>
      <div class="mt-3 sc-text-muted">
        These values are stored in SentinelCore and can be consumed by automated service templates/playbooks.
      </div>
    </div>
  </div>
</div>

<!-- Service Info Modal -->
<div v-if="showInfoModal" class="sc-modal-overlay" @click.self="showInfoModal = false">
  <div class="sc-modal-card">
    <div class="d-flex align-items-center justify-content-between mb-3">
      <h6 class="mb-0">
        <i class="mdi mdi-information-outline me-2" style="color:var(--sc-blue)"></i>
        {{ selectedService?.label }}
      </h6>
      <button class="btn btn-sm btn-sc-danger" @click="showInfoModal = false">
        <i class="mdi mdi-close"></i>
      </button>
    </div>
    <div v-if="selectedService">
      <div class="info-row">
        <span class="info-label">Service Name:</span>
        <span class="info-value font-mono">{{ selectedService.name }}</span>
      </div>
      <div class="info-row">
        <span class="info-label">Package:</span>
        <span class="info-value font-mono">{{ selectedService.package }}</span>
      </div>
      <div class="info-row">
        <span class="info-label">Category:</span>
        <span class="info-value">{{ getCategory(selectedService.name) }}</span>
      </div>
      <div class="info-row">
        <span class="info-label">Description:</span>
        <p class="info-desc">{{ getServiceDescription(selectedService.name) }}</p>
      </div>
      <div class="info-row">
        <span class="info-label">Config File:</span>
        <span class="info-value font-mono">{{ selectedService.config || 'N/A' }}</span>
      </div>
      <div class="info-row">
        <span class="info-label">Status:</span>
        <span class="badge rounded-pill" :class="statusBadge(selectedService)">{{ statusText(selectedService) }}</span>
      </div>
    </div>
  </div>
</div>

<!-- Service Config Editor Modal -->
<div v-if="showConfigModal" class="sc-modal-overlay" @click.self="closeConfigModal">
  <div class="sc-modal-card config-editor-modal">
    <div class="d-flex align-items-center justify-content-between mb-3">
      <h6 class="mb-0">
        <i class="mdi mdi-file-document-edit me-2" style="color:var(--sc-blue)"></i>
        Edit Config: {{ selectedService?.label }}
      </h6>
      <button class="btn btn-sm btn-sc-danger" @click="closeConfigModal">
        <i class="mdi mdi-close"></i>
      </button>
    </div>
    
    <div v-if="selectedService && selectedService.config">
      <!-- Config file info -->
      <div class="alert alert-info py-2" style="background:rgba(74,158,255,.1);border:none">
        <i class="mdi mdi-information-outline me-2"></i>
        Config file: <code style="font-size:0.8rem">{{ selectedService.config }}</code>
      </div>
      
      <!-- Config editor -->
      <div class="mb-3">
        <label class="form-label sc-form-label">Configuration Content</label>
        <textarea
          v-model="configContent"
          class="form-control font-mono"
          rows="15"
          style="font-size:0.85rem;background:var(--sc-bg-secondary);border-color:var(--sc-border);color:var(--sc-text)"
          placeholder="Edit configuration here..."
        ></textarea>
        <div v-if="configHasChanges" class="mt-1" style="font-size:0.75rem;color:var(--sc-warning)">
          <i class="mdi mdi-alert me-1"></i>Changes detected from original
        </div>
      </div>
      
      <!-- Integrity verification -->
      <div class="mb-3 config-integrity-section">
        <label class="form-label sc-form-label d-flex align-items-center gap-2">
          <i class="mdi mdi-shield-check" style="color:var(--sc-success)"></i>
          Integrity Verification
        </label>
        <div class="d-flex gap-2 flex-wrap">
          <button
            class="btn btn-sm"
            style="background:rgba(34,214,124,.1);color:#22d67c"
            @click="verifyConfigSyntax"
            :disabled="verifying"
          >
            <i :class="`mdi ${verifying ? 'mdi-loading mdi-spin' : 'mdi-check'} me-1`"></i>
            Verify Syntax
          </button>
          <button
            class="btn btn-sm"
            style="background:rgba(245,166,35,.1);color:#f5a623"
            @click="backupConfig"
            :disabled="backingUp"
          >
            <i :class="`mdi ${backingUp ? 'mdi-loading mdi-spin' : 'mdi-backup-restore'} me-1`"></i>
            Backup Current
          </button>
          <button
            class="btn btn-sm"
            style="background:rgba(74,158,255,.1);color:#4a9eff"
            @click="restoreDefault"
            :disabled="restoring"
          >
            <i :class="`mdi ${restoring ? 'mdi-loading mdi-spin' : 'mdi-undo-variant'} me-1`"></i>
            Restore Default
          </button>
          <button
            class="btn btn-sm"
            style="background:rgba(167,139,250,.1);color:#a78bfa"
            @click="applyRecommended"
            :disabled="!hasRecommended"
          >
            <i class="mdi mdi-star me-1"></i>
            Apply Recommended
          </button>
        </div>
        <div v-if="verificationResult" class="mt-2" style="font-size:0.75rem">
          <span :class="verificationResult.valid ? 'text-success' : 'text-danger'">
            <i :class="`mdi mdi-${verificationResult.valid ? 'check-circle' : 'alert-circle'} me-1`"></i>
            {{ verificationResult.message }}
          </span>
        </div>
      </div>
      
      <!-- Action buttons -->
      <div class="d-flex gap-2 justify-content-end mt-4">
        <button class="btn btn-sm btn-sc-secondary" @click="closeConfigModal">
          <i class="mdi mdi-cancel me-1"></i>Cancel
        </button>
        <button
          class="btn btn-sm btn-sc-primary"
          @click="saveConfig"
          :disabled="savingConfig || !configHasChanges"
        >
          <i :class="`mdi ${savingConfig ? 'mdi-loading mdi-spin' : 'mdi-content-save'} me-1`"></i>
          Save Changes
        </button>
        <button
          class="btn btn-sm"
          style="background:rgba(34,214,124,.1);color:#22d67c"
          @click="saveAndReload"
          :disabled="savingConfig || !configHasChanges"
        >
          <i :class="`mdi ${savingConfig ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>
          Save & Reload Service
        </button>
      </div>
    </div>
    
    <div v-else class="text-center py-4">
      <i class="mdi mdi-alert-outline" style="font-size:2rem;color:var(--sc-text-muted)"></i>
      <p class="mt-2 sc-text-muted">No configuration file available for this service</p>
    </div>
  </div>
</div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard from '@/components/widgets/stat-card.vue'
import api from '@/services/api'

// Service categories
const serviceCategories = {
  // Security
  'ufw': 'security',
  'fail2ban': 'security',
  'crowdsec': 'security',
  'psad': 'security',
  'clamav-daemon': 'security',
  'auditd': 'security',
  'apparmor': 'security',
  'aide': 'security',
  'rkhunter': 'security',
  // System
  'docker': 'system',
  'nginx': 'system',
  'sshd': 'system',
  'netdata': 'monitoring',
  'unattended-upgrades': 'system'
}

// Service descriptions
const serviceDescriptions = {
  'ufw': 'Uncomplicated Firewall - Easy to use firewall management tool for Linux.',
  'fail2ban': 'Fail2Ban - Scans log files and bans IPs showing malicious signs.',
  'crowdsec': 'CrowdSec - Collaborative security IPS that analyzes behavior.',
  'psad': 'Passive Security Attack Detection - Detects suspicious traffic.',
  'clamav-daemon': 'ClamAV - Antivirus engine for detecting trojans, viruses, and malware.',
  'auditd': 'Audit Daemon - Linux kernel auditing framework for security monitoring.',
  'apparmor': 'AppArmor - Application Armor - Mandatory access control system.',
  'aide': 'Advanced Intrusion Detection Environment - File integrity checker.',
  'rkhunter': 'Rootkit Hunter - Scans for rootkits, backdoors, and local exploits.',
  'docker': 'Docker - Container runtime for running applications in isolated environments.',
  'nginx': 'Nginx - High-performance web server and reverse proxy.',
  'sshd': 'OpenSSH Server - Secure Shell daemon for remote access.',
  'netdata': 'Netdata - Real-time performance monitoring tool.',
  'unattended-upgrades': 'Automatic package updates for security patches.'
}

export default {
  name: 'ServicesPage',
  components: { PageHeader, StatCard },
  data() {
    return {
      loading: false,
      cfgSaving: false,
      q: '',
      services: [],
      busy: {},
      cfg: {},
      showInfoModal: false,
      selectedService: null,
      // Service installation log window
      showInstallLogWindow: false,
      installingService: '',
      installing: false,
      installLogs: [],
      installLogSeenCount: 0,
      installHintShown: false,
      installLogPollTimer: null,
      // Config editor modal
      showConfigModal: false,
      configContent: '',
      originalConfig: '',
      verifying: false,
      backingUp: false,
      restoring: false,
      savingConfig: false,
      verificationResult: null,
      recommendedConfigs: {
        'ufw': '# Recommended UFW settings\nDEFAULT_INPUT_POLICY="DROP"\nDEFAULT_OUTPUT_POLICY="ACCEPT"\nDEFAULT_FORWARD_POLICY="DROP"\nLOG_LEVEL="4"',
        'fail2ban': '# Recommended Fail2Ban settings\n[DEFAULT]\nbantime = 3600\nfindtime = 600\nmaxretry = 5\n\n[sshd]\nenabled = true\nport = ssh\nlogpath = /var/log/auth.log',
        'psad': '# Recommended PSAD settings\nEMAIL="admin@example.com"\nEMAIL_FROM="psad@example.com"\nEMAIL_REPORTS=1\nPARANOID=1'
      }
    }
  },
  computed: {
    filtered() {
      if (!this.q) return this.services
      const t = this.q.toLowerCase()
      return this.services.filter(s =>
        s.name.toLowerCase().includes(t) ||
        s.label.toLowerCase().includes(t) ||
        (s.package || '').toLowerCase().includes(t)
      )
    },
    installedCount() {
      return this.services.filter(s => s.installed).length
    },
    runningCount() {
      return this.services.filter(s => s.running).length
    },
    missingCount() {
      return this.services.filter(s => !s.installed).length
    },
    cfgKeys() {
      return Object.keys(this.cfg)
    },
    configHasChanges() {
      return this.configContent !== this.originalConfig
    },
    hasRecommended() {
      return this.selectedService && this.recommendedConfigs[this.selectedService.name]
    }
  },
  async mounted() {
    await this.loadAll()
  },
  beforeUnmount() {
    this.stopInstallLogPolling()
  },
  methods: {
    isGhost(s) {
      return !s.installed && !!(s.active_state && s.active_state !== 'inactive')
    },
    actionTitle(s, action) {
      if (this.isGhost(s)) return 'Stale systemd unit detected. Install the package (and ensure binary exists) before managing it.'
      if (!s.installed) return 'Install the package before managing the service.'
      return `Run: ${action}`
    },
    getCategory(name) {
      return serviceCategories[name] || 'other'
    },
    getCategoryClass(name) {
      const cat = this.getCategory(name)
      return `category-${cat}`
    },
    getServiceDescription(name) {
      return serviceDescriptions[name] || 'No description available.'
    },
    showServiceInfo(s) {
      this.selectedService = s
      this.showInfoModal = true
    },
    statusBadge(s) {
      if (this.isGhost(s)) return 'badge-ghost'
      if (!s.installed) return 'badge-warning'
      if (s.running) return 'badge-online'
      return 'badge-offline'
    },
    statusText(s) {
      if (this.isGhost(s)) return 'stale unit'
      if (!s.installed) return 'not installed'
      if (s.running) return 'running'
      if (s.active_state) return `${s.active_state}:${s.sub_state}`
      return 'stopped'
    },
    prettyKey(key) {
      return key.replaceAll('_', ' ')
    },
    async loadAll() {
      this.loading = true
      try {
        const [sRes, cRes] = await Promise.all([api.getManagedServices(), api.getServiceConfig()])
        this.services = sRes.data || []
        this.cfg = cRes.data || {}
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Failed to load services', text: e.response?.data?.error || e.message })
      } finally {
        this.loading = false
      }
    },
    async installService(s) {
      const r = await this.$swal({
        title: `Install ${s.label}?`,
        text: `This will install ${s.package} via apt and try to enable/start ${s.name}.`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Install'
      })
      if (!r.isConfirmed) return

      this.busy = { ...this.busy, [s.name]: true }
      this.installing = true
      this.showInstallLogWindow = true
      this.installingService = s.label
      this.installLogs = []
      this.installLogSeenCount = 0
      this.installHintShown = false
      this.addInstallLogLine('Starting installation...', 'info')
      try {
        await api.installService(s.name)
        this.addInstallLogLine('Installation request sent to server', 'success')
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${s.label} installation started`, showConfirmButton: false, timer: 2500 })
        // Start polling for logs
        this.startInstallLogPolling()
      } catch (e) {
        const msg = e.response?.data?.error || 'Failed to start installation'
        this.addInstallLogLine(`Error: ${msg}`, 'error')
        this.$swal({ icon: 'error', title: `Install failed: ${s.label}`, text: msg })
        this.installing = false
      } finally {
        this.busy = { ...this.busy, [s.name]: false }
      }
    },

    startInstallLogPolling() {
      this.installLogPollTimer = setInterval(() => {
        this.fetchServiceInstallLogs()
      }, 2000)
    },

    async fetchServiceInstallLogs() {
      try {
        const apiSvc = (await import('@/services/api')).default
        const { data } = await apiSvc.getServiceInstallLogs()
        if (data.logs && data.logs.length > this.installLogSeenCount) {
          const newLines = data.logs.slice(this.installLogSeenCount)
          this.installLogSeenCount = data.logs.length
          newLines.forEach(log => {
            const lower = log.toLowerCase()
            const type = lower.includes('error') || lower.includes('fail') ? 'error'
              : lower.includes('ok') || lower.includes('success') ? 'success' : 'info'
            this.addInstallLogLine(log, type)

            if (!this.installHintShown) {
              const looksLikeApt100 = lower.includes('exit status 100')
              const looksLikeDpkgBroken = lower.includes('not fully installed or removed') ||
                lower.includes('dpkg was interrupted') ||
                lower.includes('run \'dpkg --configure -a\'')

              if (looksLikeApt100 || looksLikeDpkgBroken) {
                this.installHintShown = true
                this.addInstallLogLine(
                  "Hint: apt exit status 100 usually means a broken dpkg/apt state. Fix on the server: sudo dpkg --configure -a && sudo apt-get -f install && sudo apt-get update, then retry.",
                  'info'
                )
              }
            }
          })
        }
        if (data.done) {
          if (data.error) {
            this.addInstallLogLine(`Installation failed: ${data.error}`, 'error')
          } else {
            this.addInstallLogLine('Installation completed successfully', 'success')
          }
          this.stopInstallLogPolling()
          this.installing = false
          await this.loadAll()
        }
      } catch (err) {
        // Silent fail for polling
      }
    },

    stopInstallLogPolling() {
      if (this.installLogPollTimer) {
        clearInterval(this.installLogPollTimer)
        this.installLogPollTimer = null
      }
    },

    addInstallLogLine(text, type = 'info') {
      const now = new Date()
      const ts = now.toLocaleTimeString()
      this.installLogs.push({ ts, text, type })
      // Auto-scroll to bottom
      this.$nextTick(() => {
        const container = this.$refs.serviceInstallLogWindow
        if (container) {
          container.scrollTop = container.scrollHeight
        }
      })
    },

    clearInstallLogs() {
      this.installLogs = []
    },
    async act(s, action) {
      this.busy = { ...this.busy, [s.name]: true }
      try {
        await api.serviceAction(s.name, action)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${s.label}: ${action} success`, showConfirmButton: false, timer: 2000 })
        await this.loadAll()
      } catch (e) {
        this.$swal({ icon: 'error', title: `${action} failed`, text: e.response?.data?.error || e.message })
      } finally {
        this.busy = { ...this.busy, [s.name]: false }
      }
    },
    async saveCfg() {
      this.cfgSaving = true
      try {
        await api.updateServiceConfig(this.cfg)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Service config saved', showConfirmButton: false, timer: 2000 })
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Save failed', text: e.response?.data?.error || e.message })
      } finally {
        this.cfgSaving = false
      }
    },
    // Config editor methods
    async openConfigEditor(s) {
      if (!s.config) {
        this.$swal({ icon: 'info', title: 'No Config File', text: 'This service does not have a configurable file.' })
        return
      }
      this.selectedService = s
      this.originalConfig = ''
      this.configContent = ''
      this.verificationResult = null
      this.showConfigModal = true
      try {
        const { data } = await api.getServiceConfigFile(s.name)
        this.configContent = data.content || ''
        this.originalConfig = this.configContent
      } catch (e) {
        // Fall back to recommended template if the file can't be read yet
        this.configContent = this.recommendedConfigs[s.name] || `# ${s.config}`
        this.originalConfig = this.configContent
      }
    },
    closeConfigModal() {
      this.showConfigModal = false
      this.selectedService = null
      this.configContent = ''
      this.originalConfig = ''
      this.verificationResult = null
    },
    async verifyConfigSyntax() {
      if (!this.selectedService) return
      this.verifying = true
      this.verificationResult = null
      try {
        const { data } = await api.verifyServiceConfigFile(this.selectedService.name)
        this.verificationResult = { valid: data.valid, message: data.message }
      } catch (e) {
        this.verificationResult = { valid: false, message: e.response?.data?.error || e.message }
      } finally {
        this.verifying = false
      }
    },
    async backupConfig() {
      if (!this.selectedService) return
      this.backingUp = true
      try {
        await api.backupServiceConfigFile(this.selectedService.name)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Backup created for ${this.selectedService.label}`, showConfirmButton: false, timer: 2000 })
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Backup failed', text: e.response?.data?.error || e.message })
      } finally {
        this.backingUp = false
      }
    },
    async restoreDefault() {
      if (!this.selectedService) return
      const confirmed = await this.$swal({
        title: 'Restore from Backup?',
        text: `This will restore the most recent backup for ${this.selectedService.label}. Unsaved changes will be lost.`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Restore'
      })
      if (!confirmed.isConfirmed) return
      this.restoring = true
      try {
        await api.restoreServiceConfigFile(this.selectedService.name)
        // Reload the now-restored content from the server
        const { data } = await api.getServiceConfigFile(this.selectedService.name)
        this.configContent = data.content || ''
        this.originalConfig = this.configContent
        this.verificationResult = null
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Backup restored', showConfirmButton: false, timer: 2000 })
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Restore failed', text: e.response?.data?.error || e.message })
      } finally {
        this.restoring = false
      }
    },
    applyRecommended() {
      if (!this.selectedService || !this.recommendedConfigs[this.selectedService.name]) return
      this.configContent = this.recommendedConfigs[this.selectedService.name]
      this.verificationResult = null
      this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Recommended settings applied', showConfirmButton: false, timer: 2000 })
    },
    async saveConfig() {
      if (!this.selectedService || !this.configHasChanges) return
      this.savingConfig = true
      try {
        await api.saveServiceConfigFile(this.selectedService.name, this.configContent)
        this.originalConfig = this.configContent
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Configuration saved for ${this.selectedService.label}`, showConfirmButton: false, timer: 2000 })
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Save failed', text: e.response?.data?.error || e.message })
      } finally {
        this.savingConfig = false
      }
    },
    async saveAndReload() {
      await this.saveConfig()
      if (this.selectedService && !this.configHasChanges) {
        await this.act(this.selectedService, 'restart')
        this.closeConfigModal()
      }
    }
  }
}
</script>

<style scoped>
/* Theme-aware text colors */
.sc-text-muted { color: var(--sc-text-muted, #5a7499); }
.sc-text-secondary { color: var(--sc-text-secondary, #8aa4c8); }

/* Service label styling */
.service-label {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--sc-text, #e2ecff);
}

.service-name {
  font-size: 0.68rem;
  color: var(--sc-text-muted, #5a7499);
}

/* Category badges */
.category-badge {
  font-size: 0.65rem !important;
  padding: 2px 8px !important;
  text-transform: capitalize;
}

.category-security {
  background: rgba(240, 64, 64, 0.12) !important;
  color: #f04040 !important;
}

.category-system {
  background: rgba(74, 158, 255, 0.12) !important;
  color: #4a9eff !important;
}

.category-monitoring {
  background: rgba(167, 139, 250, 0.12) !important;
  color: #a78bfa !important;
}

.category-other {
  background: rgba(245, 166, 35, 0.12) !important;
  color: #f5a623 !important;
}

/* Action buttons */
.btn-install {
  background: rgba(245, 166, 35, 0.12) !important;
  color: #f5a623 !important;
  padding: 2px 8px !important;
  font-size: 0.68rem !important;
}
.btn-install:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-start {
  background: rgba(34, 214, 124, 0.12) !important;
  color: #22d67c !important;
  padding: 2px 8px !important;
  font-size: 0.68rem !important;
}
.btn-start:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-restart {
  background: rgba(74, 158, 255, 0.12) !important;
  color: #4a9eff !important;
  padding: 2px 8px !important;
  font-size: 0.68rem !important;
}
.btn-restart:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-stop {
  background: rgba(240, 64, 64, 0.12) !important;
  color: #f04040 !important;
  padding: 2px 8px !important;
  font-size: 0.68rem !important;
}
.btn-stop:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.badge-ghost {
  background: rgba(245, 166, 35, 0.12) !important;
  color: #f5a623 !important;
  border: 1px dashed rgba(245, 166, 35, 0.35);
}

/* Info button */
.service-info-btn {
  background: transparent !important;
  color: var(--sc-text-muted, #5a7499) !important;
  padding: 2px 6px !important;
  font-size: 0.75rem !important;
  border: 1px solid var(--sc-border, #2a3f5f) !important;
}
.service-info-btn:hover {
  color: var(--sc-blue, #4a9eff) !important;
  border-color: var(--sc-blue, #4a9eff) !important;
}

/* Config editor button */
.service-config-btn {
  background: transparent !important;
  color: var(--sc-purple, #a78bfa) !important;
  padding: 2px 6px !important;
  font-size: 0.75rem !important;
  border: 1px solid var(--sc-border, #2a3f5f) !important;
}
.service-config-btn:hover {
  color: var(--sc-purple, #a78bfa) !important;
  border-color: var(--sc-purple, #a78bfa) !important;
}

/* Config editor modal */
.config-editor-modal {
  width: min(800px, 95vw) !important;
}

.config-integrity-section {
  padding: 0.75rem;
  background: rgba(74,158,255,.05);
  border-radius: 6px;
  border: 1px solid var(--sc-border);
}

/* Form label */
.sc-form-label {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--sc-text, #e2ecff);
}

/* Modal styles */
.sc-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1600;
  padding: 1rem;
}

.sc-modal-card {
  width: min(600px, 96vw);
  max-height: 85vh;
  overflow: auto;
  background: var(--sc-bg-card, #0d1b2a);
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 12px;
  padding: 1.5rem;
}

.info-row {
  margin-bottom: 1rem;
}

.info-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--sc-text-muted, #5a7499);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.25rem;
}

.info-value {
  font-size: 0.9rem;
  color: var(--sc-text, #e2ecff);
}

.info-desc {
  font-size: 0.85rem;
  color: var(--sc-text-secondary, #8aa4c8);
  margin: 0.25rem 0 0 0;
  line-height: 1.5;
}

/* Card styling */
.sc-view-services :deep(.card-header) { padding: 0.85rem 1rem; }
.sc-view-services :deep(.card-body) { padding: 1rem; }

/* Service installation log window */
.service-install-log-card {
  border: 1px solid var(--sc-border);
}

.service-install-log-window {
  height: 280px;
  overflow-y: auto;
  background: var(--sc-bg-secondary);
  padding: 0.75rem;
  font-size: 0.78rem;
}

.service-install-log-line {
  display: flex;
  gap: 0.5rem;
  padding: 2px 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.service-install-log-line .log-timestamp {
  color: var(--sc-text-muted);
  min-width: 70px;
}

.service-install-log-line .log-content {
  color: var(--sc-text);
}

.service-install-log-line .log-content.error {
  color: var(--sc-red);
}

.service-install-log-line .log-content.success {
  color: var(--sc-green);
}

.service-install-log-line .log-content.warn {
  color: var(--sc-amber);
}

.service-install-log-line .log-cursor {
  font-style: italic;
  color: var(--sc-text-muted);
}
</style>
