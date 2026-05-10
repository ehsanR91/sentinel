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

  <div class="card sc-panel-card mb-4 services-table-card">
    <div class="card-header services-table-header">
      <div>
        <h6><i class="mdi mdi-cog-refresh-outline me-2" style="color:var(--sc-blue)"></i>Managed Services</h6>
        <div class="services-table-note">Browse, monitor and manage all system services from one pane.</div>
      </div>
      <div class="services-toolbar">
        <div class="toolbar-left">
          <div class="services-search">
            <i class="mdi mdi-magnify"></i>
            <input
              v-model="q"
              @input="clearExpanded"
              class="form-control"
              placeholder="Search services…"
              type="search"
            />
          </div>
          <button type="button" class="filter-chip" :class="{ active: runningOnly }" @click="runningOnly = !runningOnly">
            Running only
          </button>
          <button type="button" class="filter-chip" :class="{ active: failedOnly }" @click="failedOnly = !failedOnly">
            With issues
          </button>
        </div>
        <div class="toolbar-right">
          <button type="button" class="btn btn-sm btn-secondary" @click="toggleDensity">
            <i class="mdi mdi-view-grid-outline me-1"></i>{{ densityLabel }}
          </button>
          <button type="button" class="btn btn-sm btn-secondary" :disabled="loading" @click="loadAll">
            <i :class="`mdi ${loading ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>Refresh
          </button>
        </div>
      </div>
    </div>
    <div class="card-body p-0">
      <table class="services-table">
        <thead>
          <tr>
            <th class="service-col">Service</th>
            <th>Category</th>
            <th>Status</th>
            <th>Uptime</th>
            <th class="actions-col">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading" v-for="n in 6" :key="`skeleton-${n}`" class="service-row skeleton-row">
            <td colspan="5"><div class="skeleton-row-placeholder"></div></td>
          </tr>
          <tr v-else-if="!filtered.length" class="service-row no-data-row">
            <td colspan="5" class="text-center py-4 sc-text-muted">
              <div class="no-data-title">No services match your filter</div>
              <button class="btn btn-sm btn-secondary" @click="clearFilter">Clear filters</button>
            </td>
          </tr>
          <template v-else>
            <template v-for="s in filtered" :key="s.name">
              <tr
                class="service-row"
                :class="{ expanded: expandedService === s.name, compact: compactDensity }"
                @click="toggleExpand(s)"
              >
                <td class="service-cell">
                  <span class="status-dot" :class="statusDotClass(s)" aria-hidden="true"></span>
                  <div class="service-meta">
                    <div class="service-label">{{ s.label }}</div>
                    <div class="service-id font-mono">{{ s.package || s.name }}</div>
                  </div>
                </td>
                <td>
                  <span class="category-chip" :class="categoryClass(s)">{{ categoryLabel(s) }}</span>
                </td>
                <td>
                  <span class="status-pill" :class="statusPillClass(s)">{{ statusText(s) }}</span>
                </td>
                <td>{{ statusDetail(s) }}</td>
                <td class="actions-cell">
                  <div class="service-actions">
                    <button
                      type="button"
                      class="service-btn primary"
                      :disabled="busy[s.name] || isTransitioning(s)"
                      @click.stop="primaryAction(s)"
                    >
                      <i :class="`mdi ${primaryIcon(s)} me-1 ${busy[s.name] ? 'mdi-loading mdi-spin' : ''}`"></i>
                      {{ primaryLabel(s) }}
                    </button>
                    <button
                      type="button"
                      class="service-btn secondary"
                      :disabled="busy[s.name]"
                      @click.stop="openServiceLogs(s)"
                    >
                      <i class="mdi mdi-file-document-box-outline me-1"></i>Logs
                    </button>
                    <button
                      type="button"
                      class="service-btn overflow"
                      :disabled="busy[s.name]"
                      @click.stop="toggleActionMenu(s.name)"
                    >
                      <i class="mdi mdi-dots-vertical"></i>
                    </button>
                    <div v-if="actionMenuOpen === s.name" class="service-action-menu shadow-sm">
                      <button class="dropdown-item" @click.stop="openServiceLogs(s)">
                        <i class="mdi mdi-file-document-box-outline me-1"></i>View logs
                      </button>
                      <button class="dropdown-item" @click.stop="openConfigEditor(s)" :disabled="!s.config">
                        <i class="mdi mdi-file-document-edit me-1"></i>Edit config
                      </button>
                      <button class="dropdown-item" @click.stop="showServiceInfo(s)">
                        <i class="mdi mdi-information-outline me-1"></i>View details
                      </button>
                      <div class="dropdown-divider"></div>
                      <button class="dropdown-item" @click.stop="act(s, s.running ? 'stop' : 'start')" :disabled="busy[s.name] || isTransitioning(s)">
                        <i :class="`mdi mdi-${s.running ? 'stop' : 'play'} me-1`"></i>{{ s.running ? 'Stop' : 'Start' }}
                      </button>
                      <button class="dropdown-item" @click.stop="confirmReinstall(s)" :disabled="busy[s.name]">
                        <i class="mdi mdi-refresh-circle me-1"></i>Reinstall
                      </button>
                      <button class="dropdown-item text-danger" @click.stop="confirmUninstall(s)" :disabled="busy[s.name]">
                        <i class="mdi mdi-delete-outline me-1"></i>Uninstall
                      </button>
                    </div>
                  </div>
                </td>
              </tr>
              <tr v-if="expandedService === s.name" :key="`${s.name}-details`" class="service-details-row">
                <td colspan="5" class="service-details-cell">
                  <div class="service-details-grid">
                    <div>
                      <span class="detail-label">Description</span>
                      <p class="detail-text">{{ getServiceDescription(s.name) }}</p>
                    </div>
                    <div>
                      <span class="detail-label">Config path</span>
                      <div class="detail-meta font-mono">{{ s.config || 'None' }}</div>
                    </div>
                    <div>
                      <span class="detail-label">Unit name</span>
                      <div class="detail-meta font-mono">{{ s.unit || s.name }}</div>
                    </div>
                  </div>
                  <div class="service-details-actions">
                    <button class="service-btn secondary" @click.stop="openConfigEditor(s)" :disabled="!s.config">Edit config</button>
                    <button class="service-btn secondary" @click.stop="showServiceInfo(s)">Details</button>
                    <button class="service-btn secondary" @click.stop="openServiceLogs(s)">View logs</button>
                    <button v-if="s.config" class="service-btn secondary" @click.stop="copyConfigPath(s.config)">Copy path</button>
                  </div>
                </td>
              </tr>
            </template>
          </template>
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

<!-- Service Logs Modal -->
<div v-if="showLogsModal" class="sc-modal-overlay" @click.self="closeLogsModal">
  <div class="sc-modal-card logs-modal-card">
    <div class="d-flex align-items-center justify-content-between mb-3">
      <h6 class="mb-0">
        <i class="mdi mdi-file-document-box-outline me-2" style="color:var(--sc-cyan)"></i>
        Logs: {{ logService?.label || 'Service' }}
      </h6>
      <button class="btn btn-sm btn-sc-danger" @click="closeLogsModal">
        <i class="mdi mdi-close"></i>
      </button>
    </div>
    <div class="d-flex align-items-center gap-2 mb-3 flex-wrap">
      <label class="form-label sc-form-label mb-0">Lines</label>
      <input type="number" min="10" max="2000" step="10" v-model.number="logLineCount" class="form-control form-control-sm" style="width:90px" />
      <button class="btn btn-sm btn-sc-primary" :disabled="logLoading" @click="loadServiceLogs">
        <i :class="`mdi ${logLoading ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>Refresh
      </button>
      <span class="text-muted" style="font-size:0.78rem">Showing last {{ logLineCount }} lines</span>
    </div>
    <div class="log-output-card">
      <pre class="log-output font-mono">{{ logLines.join('\n') }}</pre>
      <div v-if="logLoading" class="log-loading-overlay">
        <span class="spinner-border spinner-border-sm text-info"></span>
        <span class="ms-2">Loading logs…</span>
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
import { useDocumentVisibility } from '@vueuse/core'
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
  setup() {
    return {
      documentVisibility: useDocumentVisibility()
    }
  },
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
      expandedService: null,
      runningOnly: false,
      failedOnly: false,
      compactDensity: false,
      // Service installation log window
      showInstallLogWindow: false,
      installingService: '',
      installing: false,
      installLogs: [],
      installLogSeenCount: 0,
      installHintShown: false,
      installLogPollTimer: null,
      showLogsModal: false,
      logService: null,
      logLines: [],
      logLineCount: 50,
      logLoading: false,
      actionMenuOpen: null,
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
      const query = this.q.trim().toLowerCase()
      return this.services
        .filter(s => {
          if (query) {
            const haystack = `${s.name} ${s.label} ${s.package || ''}`.toLowerCase()
            if (!haystack.includes(query)) return false
          }
          if (this.runningOnly && !s.running) return false
          if (this.failedOnly && (s.running || !s.installed)) return false
          return true
        })
    },
    densityLabel() {
      return this.compactDensity ? 'Compact view' : 'Comfort view'
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
  watch: {
    documentVisibility(value) {
      if (value === 'visible') {
        this.loadAll()
        if (this.installing) {
          this.fetchServiceInstallLogs()
        }
      }
    },
    '$route.query': {
      immediate: false,
      handler() {
        this.applyRouteQuery()
      }
    }
  },
  async mounted() {
    document.addEventListener('click', this.closeActionMenus)
    await this.loadAll()
    await this.applyRouteQuery()
  },
  beforeUnmount() {
    this.stopInstallLogPolling()
    document.removeEventListener('click', this.closeActionMenus)
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
      if (this.statusLabel(s) === 'Active') return 'badge-online'
      if (this.statusLabel(s) === 'Disabled') return 'badge-warning'
      return 'badge-offline'
    },
    statusLabel(s) {
      if (this.isGhost(s)) return 'Stale Unit'
      if (!s.installed) return 'Not Installed'
      if (s.running) return 'Active'
      if (s.active_state === 'inactive') return 'Disabled'
      return 'Exited'
    },
    statusTooltip(s) {
      const label = this.statusLabel(s)
      if (label === 'Active') return 'The service is installed and running fine.'
      if (label === 'Disabled') return 'The service is installed but currently disabled.'
      if (label === 'Exited') return 'The service is installed but currently not active; it may have stopped or failed.'
      if (label === 'Not Installed') return 'The service package is not installed.'
      if (label === 'Stale Unit') return 'A systemd unit exists without an installed package. Install the package to reconcile it.'
      return 'The service state is unavailable or inconsistent.'
    },
    toggleDensity() {
      this.compactDensity = !this.compactDensity
    },
    clearExpanded() {
      this.expandedService = null
    },
    clearFilter() {
      this.q = ''
      this.runningOnly = false
      this.failedOnly = false
      this.clearExpanded()
    },
    async applyRouteQuery() {
      const { state, service, panel } = this.$route.query || {}
      if (state === 'running') {
        this.runningOnly = true
        this.failedOnly = false
      } else if (state === 'stopped') {
        this.runningOnly = false
        this.failedOnly = true
      }

      if (service) {
        const query = String(service)
        this.q = query
        const match = this.services.find(entry => {
          const haystack = `${entry.name} ${entry.label} ${entry.package || ''}`.toLowerCase()
          return haystack.includes(query.toLowerCase())
        })
        if (match) {
          this.selectedService = match
          this.expandedService = match.name
          if (panel === 'logs') {
            await this.openServiceLogs(match)
          }
        }
      }
    },
    toggleExpand(s) {
      this.expandedService = this.expandedService === s.name ? null : s.name
    },
    statusDotClass(s) {
      if (this.isGhost(s)) return 'status-ghost'
      if (!s.installed) return 'status-missing'
      if (s.running) return 'status-online'
      if (s.active_state === 'inactive') return 'status-disabled'
      return 'status-offline'
    },
    statusPillClass(s) {
      if (this.isGhost(s)) return 'pill-ghost'
      if (!s.installed) return 'pill-warning'
      if (s.running) return 'pill-online'
      if (s.active_state === 'inactive') return 'pill-disabled'
      return 'pill-offline'
    },
    categoryLabel(s) {
      return this.getCategory(s.name)
    },
    categoryClass(s) {
      return `category-${this.getCategory(s.name)}`
    },
    statusDetail(s) {
      if (!s.installed) return 'Install package'
      if (s.running && Number.isFinite(s.uptime)) {
        return `Running for ${this.formatDuration(s.uptime)}`
      }
      if (s.active_state === 'inactive') return 'Service disabled'
      if (this.isGhost(s)) return 'Stale unit'
      return s.active_state ? this.capitalize(s.active_state) : 'Unknown state'
    },
    isTransitioning(s) {
      return ['activating', 'deactivating', 'starting', 'stopping'].includes(s.active_state)
    },
    primaryLabel(s) {
      if (!s.installed) return 'Install'
      if (s.running) return 'Restart'
      return 'Start'
    },
    primaryIcon(s) {
      if (!s.installed) return 'mdi-package-down'
      if (s.running) return 'mdi-restart'
      return 'mdi-play'
    },
    async primaryAction(s) {
      if (!s.installed) {
        return this.installService(s)
      }
      if (s.running) {
        return this.act(s, 'restart')
      }
      return this.act(s, 'start')
    },
    copyConfigPath(path) {
      if (!navigator.clipboard) return
      navigator.clipboard.writeText(path).then(() => {
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Copied config path', showConfirmButton: false, timer: 1500 })
      })
    },
    formatDuration(seconds) {
      if (!Number.isFinite(seconds) || seconds < 0) return 'Unknown duration'
      const days = Math.floor(seconds / 86400)
      const hours = Math.floor((seconds % 86400) / 3600)
      const mins = Math.floor((seconds % 3600) / 60)
      const parts = []
      if (days) parts.push(`${days}d`)
      if (hours) parts.push(`${hours}h`)
      if (mins || !parts.length) parts.push(`${mins}m`)
      return parts.join(' ')
    },
    capitalize(value) {
      return value ? `${value.charAt(0).toUpperCase()}${value.slice(1)}` : ''
    },
    prettyKey(key) {
      return key.replaceAll('_', ' ')
    },
    async loadServices() {
      try {
        const sRes = await api.getManagedServices()
        this.services = sRes.data || []
      } catch (e) {
        console.error('Failed to refresh services:', e)
      }
    },

    async loadConfig() {
      try {
        const cRes = await api.getServiceConfig()
        this.cfg = cRes.data || {}
      } catch (e) {
        console.error('Failed to load service config:', e)
      }
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
        if (this.documentVisibility !== 'visible') return
        this.fetchServiceInstallLogs()
      }, 2000)
    },

    async fetchServiceInstallLogs() {
      try {
        const { data } = await api.getServiceInstallLogs()
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
    toggleActionMenu(name) {
      this.actionMenuOpen = this.actionMenuOpen === name ? null : name
    },
    closeActionMenu() {
      this.actionMenuOpen = null
    },
    closeActionMenus(e) {
      if (!e.target.closest('.service-action-menu') && !e.target.closest('.btn-more')) {
        this.closeActionMenu()
      }
    },
    async confirmUninstall(s) {
      const confirmed = await this.$swal({
        title: `Uninstall ${s.label}?`,
        text: `This will remove the package and stop the service unit.`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Uninstall'
      })
      if (!confirmed.isConfirmed) return
      await this.act(s, 'uninstall')
    },
    async confirmReinstall(s) {
      const confirmed = await this.$swal({
        title: `Reinstall ${s.label}?`,
        text: `This will reinstall the package and attempt to restore the service configuration. Existing service files may be overwritten or reloaded.`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Reinstall'
      })
      if (!confirmed.isConfirmed) return
      await this.act(s, 'reinstall')
    },
    async act(s, action) {
      this.busy = { ...this.busy, [s.name]: true }
      try {
        await api.serviceAction(s.name, action)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${s.label}: ${action} success`, showConfirmButton: false, timer: 2000 })
        await this.loadServices()
        this.closeActionMenu()
      } catch (e) {
        const error = e.response?.data?.error || e.detailedMessage || e.message || 'Unknown error'
        const hint = e.response?.data?.hint || ''
        const html = hint ? `<p>${error}</p><hr><p style="font-size:0.85rem;color:#c9d8f0">Suggested fix:</p><pre style="background:rgba(15,23,42,0.95);padding:10px;border-radius:8px;color:#f8fafc;font-size:0.82rem;white-space:pre-wrap;word-break:break-word">${hint}</pre>` : `<p>${error}</p>`
        this.$swal({ icon: 'error', title: `${action} failed`, html, customClass: { popup: 'sc-swal-popup' }, width: 600 })
      } finally {
        this.busy = { ...this.busy, [s.name]: false }
      }
    },
    async openServiceLogs(s) {
      this.logService = s
      this.logLines = []
      this.logLineCount = 50
      this.showLogsModal = true
      await this.loadServiceLogs()
    },

    async loadServiceLogs() {
      if (!this.logService) return
      this.logLoading = true
      try {
        const source = this.logService.unit || this.logService.name
        const { data } = await api.getLogs(source, this.logLineCount)
        this.logLines = data.lines || []
      } catch (e) {
        this.logLines = [`Failed to load logs: ${e.response?.data?.error || e.message || 'unknown error'}`]
      } finally {
        this.logLoading = false
      }
    },

    closeLogsModal() {
      this.showLogsModal = false
      this.logService = null
      this.logLines = []
      this.logLineCount = 50
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

.services-table-card .card-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.services-table-note {
  margin-top: 0.25rem;
  color: var(--sc-text-secondary, #8aa4c8);
  font-size: 0.86rem;
}

.services-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
}

.toolbar-left {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.services-search {
  position: relative;
  min-width: 240px;
  width: min(100%, 320px);
}

.services-search i {
  position: absolute;
  left: 1rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--sc-text-muted, #5a7499);
}

.services-search input {
  padding-left: 2.4rem;
}

.filter-chip {
  border: 1px solid var(--sc-border, #1e2d4a);
  background: var(--sc-bg-secondary, #0b1525);
  color: var(--sc-text, #e2ecff);
  border-radius: 999px;
  padding: 0.45rem 0.95rem;
  font-size: 0.78rem;
  transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease;
}

.filter-chip.active {
  background: rgba(74, 158, 255, 0.14);
  border-color: rgba(74, 158, 255, 0.3);
  color: #4a9eff;
}

.toolbar-right {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.services-table {
  width: 100%;
  border-collapse: collapse;
}

.services-table th,
.services-table td {
  padding: 1rem 1.25rem;
  vertical-align: middle;
  border-bottom: 1px solid var(--sc-border, #1e2d4a);
}

.services-table thead {
  background: rgba(255, 255, 255, 0.03);
  position: sticky;
  top: 0;
  z-index: 1;
}

.service-cell {
  min-width: 250px;
}

.service-meta {
  display: flex;
  flex-direction: column;
  gap: 0.18rem;
}

.category-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.35rem 0.8rem;
  font-size: 0.75rem;
  text-transform: capitalize;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.35rem 0.8rem;
  font-size: 0.75rem;
  font-weight: 600;
}

.service-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  justify-content: flex-end;
}

.service-btn {
  border: 1px solid transparent;
  border-radius: 999px;
  min-height: 36px;
  padding: 0.55rem 0.95rem;
  font-size: 0.8rem;
  color: var(--sc-text, #e2ecff);
  background: rgba(255, 255, 255, 0.04);
  transition: background 0.18s ease, transform 0.15s ease;
}

.service-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.08);
}

.service-btn.primary {
  background: rgba(74, 158, 255, 0.14);
  border-color: rgba(74, 158, 255, 0.28);
}

.service-btn.secondary {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.08);
}

.service-btn.overflow {
  min-width: 36px;
  width: 36px;
  padding: 0;
  display: grid;
  place-items: center;
}

.service-details-row {
  background: rgba(255, 255, 255, 0.02);
}

.service-details-cell {
  padding: 1rem 1.25rem 1.25rem;
}

.service-details-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.detail-label {
  display: block;
  font-size: 0.72rem;
  color: var(--sc-text-secondary, #8aa4c8);
  margin-bottom: 0.3rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.detail-text {
  margin: 0;
  color: var(--sc-text, #e2ecff);
}

.detail-meta {
  color: var(--sc-text-muted, #5a7499);
}

.service-details-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 1rem;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 0.85rem;
  flex-shrink: 0;
  background: var(--sc-border, #1e2d4a);
}

.status-online { background: #22d67c; }
.status-offline { background: #f5a623; }
.status-disabled { background: #f5a623; }
.status-missing { background: #f5a623; }
.status-ghost { background: #f5a623; border: 1px dashed rgba(245, 166, 35, 0.6); }

.pill-online { background: rgba(34, 214, 124, 0.12); color: #22d67c; }
.pill-offline { background: rgba(245, 166, 35, 0.12); color: #f5a623; }
.pill-disabled { background: rgba(245, 166, 35, 0.12); color: #f5a623; }
.pill-warning { background: rgba(245, 166, 35, 0.12); color: #f5a623; }
.pill-ghost { background: rgba(245, 166, 35, 0.12); color: #f5a623; border: 1px dashed rgba(245, 166, 35, 0.35); }

.skeleton-row-placeholder {
  height: 72px;
  width: 100%;
  background: linear-gradient(90deg, rgba(255,255,255,0.06) 0%, rgba(255,255,255,0.1) 50%, rgba(255,255,255,0.06) 100%);
  border-radius: 16px;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.7; }
  50% { opacity: 1; }
}

@media (max-width: 900px) {
  .service-details-grid {
    grid-template-columns: 1fr;
  }
  .services-table th,
  .services-table td {
    padding: 0.85rem 1rem;
  }
}

@media (max-width: 700px) {
  .services-table thead {
    display: none;
  }
  .service-row,
  .service-details-row {
    display: block;
    border-bottom: 1px solid var(--sc-border, #1e2d4a);
  }
  .service-row {
    padding: 0.75rem 0;
  }
  .service-cell,
  .actions-cell {
    display: block;
    width: 100%;
  }
  .service-cell {
    padding-bottom: 0.75rem;
  }
  .actions-cell {
    padding-top: 0.75rem;
  }
  .service-actions {
    justify-content: flex-start;
  }
  .service-details-cell {
    padding: 1rem 0 1.25rem;
  }
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

.btn-more {
  background: transparent !important;
  color: var(--sc-text-muted, #5a7499) !important;
  border: 1px solid var(--sc-border, #2a3f5f) !important;
  padding: 2px 8px !important;
  font-size: 0.75rem !important;
}

.service-action-menu {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  z-index: 1000;
  min-width: 190px;
  background: var(--sc-bg-card, #0d1b2a);
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 10px;
  padding: 0.35rem 0;
}

.service-action-menu .dropdown-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 0.85rem;
  color: var(--sc-text, #e2ecff);
  background: transparent;
  border: none;
  text-align: left;
}

.service-action-menu .dropdown-item:hover {
  background: rgba(74,158,255,0.08);
}

.service-action-menu .dropdown-divider {
  height: 1px;
  margin: 0.35rem 0;
  background: rgba(255,255,255,0.08);
  border: none;
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

.logs-modal-card {
  width: min(840px, 95vw) !important;
  max-height: 90vh;
}

.log-output-card {
  position: relative;
  min-height: 360px;
  max-height: 60vh;
  overflow: hidden;
  background: rgba(var(--sc-border-rgb, 30, 45, 74), 0.45);
  border: 1px solid var(--sc-border);
  border-radius: 12px;
  padding: 1rem;
}

.log-output {
  margin: 0;
  font-size: 0.74rem;
  line-height: 1.4;
  color: var(--sc-text);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: calc(60vh - 2rem);
  overflow-y: auto;
}

.log-loading-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  color: var(--sc-text);
  font-size: 0.9rem;
  gap: 0.5rem;
  z-index: 10;
}

.sc-swal-popup {
  max-width: 640px !important;
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
