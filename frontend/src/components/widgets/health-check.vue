<template>
  <div class="card h-100">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6><i class="mdi mdi-heart-pulse me-2" style="color:#4a9eff"></i>SentinelCore Health</h6>
      <div class="d-flex align-items-center gap-2">
        <span class="badge" :class="overallStatusClass" style="font-size:0.72rem">{{ overallStatusText }}</span>
        <span class="score-badge">{{ healthData.score }}/100</span>
      </div>
    </div>

    <div class="card-body health-card-body">
      <div v-if="loading" class="health-loading">
        <div class="shimmer-line"></div>
        <div class="shimmer-line"></div>
        <div class="shimmer-line"></div>
      </div>

      <div v-else-if="error" class="health-error">
        <i class="mdi mdi-alert-circle-outline"></i>
        <span>Health check unavailable</span>
        <button @click="loadHealth" class="btn btn-sm btn-outline-light mt-2">
          <i class="mdi mdi-refresh me-1"></i>Retry
        </button>
      </div>

      <div v-else class="health-content">
        <div class="health-summary">
          <div class="summary-item"><i class="mdi mdi-clock-outline"></i><span>Uptime: {{ healthData.uptime }}</span></div>
          <div class="summary-item"><i class="mdi mdi-update"></i><span>Last check: {{ formatTime(healthData.timestamp) }}</span></div>
        </div>

        <div class="overall-message" :class="overallStatusClass">
          <i :class="getStatusIcon(healthData.overall_status)"></i>
          <span>{{ healthData.summary }}</span>
        </div>

        <div v-if="issueChecks.length" class="issue-strip">
          <button
            v-for="check in issueChecks"
            :key="check.name"
            type="button"
            class="issue-chip"
            :class="check.status"
            @click="showCheckDetails(check)"
          >
            <i :class="getStatusIcon(check.status)"></i>
            <span>{{ check.name }}</span>
          </button>
        </div>

        <div class="health-checks">
          <div
            v-for="check in normalizedChecks"
            :key="check.name"
            class="health-check-item"
            :class="[check.status, { 'has-issue': check.status === 'critical' || check.status === 'warning' }]"
            @click="showCheckDetails(check)"
          >
            <div class="check-header">
              <i :class="getCheckIcon(check.name)"></i>
              <span class="check-name">{{ check.name }}</span>
              <span v-if="check.status === 'critical'" class="issue-badge critical">Critical</span>
              <span v-else-if="check.status === 'warning'" class="issue-badge warning">Warning</span>
            </div>
            <div class="check-message">{{ check.message }}</div>
            <div class="check-time">{{ check.duration }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="selectedCheck" class="modal-overlay" @click="selectedCheck = null">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h6><i :class="getCheckIcon(selectedCheck.name)"></i>{{ selectedCheck.name }}</h6>
          <button @click="selectedCheck = null" class="btn-close"><i class="mdi mdi-close"></i></button>
        </div>
        <div class="modal-body">
          <div class="check-status" :class="selectedCheck.status">
            <i :class="getStatusIcon(selectedCheck.status)"></i>
            <span>{{ selectedCheck.status }}</span>
          </div>
          <div class="check-message-full">{{ selectedCheck.message }}</div>
          <div class="check-meta">
            <div><strong>Last Checked:</strong> {{ formatTime(selectedCheck.last_checked) }}</div>
            <div><strong>Duration:</strong> {{ selectedCheck.duration }}</div>
          </div>

          <div v-if="selectedCheck.status === 'critical' || selectedCheck.status === 'warning'" class="fix-section">
            <h6><i class="mdi mdi-wrench-outline"></i> Remedies</h6>
            <div v-if="selectedRemedy" class="remedy-text">{{ selectedRemedy }}</div>
            <div v-if="selectedCommand" class="remedy-command">
              <div class="command-label">Command:</div>
              <pre class="command-text">{{ selectedCommand }}</pre>
              <button class="btn btn-sm btn-outline-light mt-2" @click="copyCommand(selectedCommand)">
                <i class="mdi mdi-content-copy"></i> Copy
              </button>
            </div>
            <div v-if="fixResponse" class="fix-response" :class="fixResponse.success ? 'success' : 'error'">
              {{ fixResponse.message }}
            </div>
            <div class="fix-actions">
              <button class="btn btn-sm btn-sc-primary" @click="fixIssue('auto')" :disabled="fixing">
                <i v-if="fixing" class="mdi mdi-loading mdi-spin"></i>
                <i v-else class="mdi mdi-auto-fix"></i>
                Auto Fix
              </button>
              <button class="btn btn-sm btn-outline-light" @click="fixIssue('manual')" :disabled="fixing">
                <i class="mdi mdi-console"></i>
                Show Commands
              </button>
            </div>
          </div>

          <div v-if="selectedCheck.details" class="check-details mt-3">
            <h6>Details</h6>
            <pre>{{ JSON.stringify(selectedCheck.details, null, 2) }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'HealthCheck',
  data() {
    return {
      healthData: { overall_status: 'unknown', score: 0, checks: [], summary: '', timestamp: null, uptime: '' },
      loading: false,
      error: false,
      selectedCheck: null,
      refreshTimer: null,
      fixing: false,
      fixResponse: null
    }
  },
  computed: {
    overallStatusClass() {
      return `status-${this.healthData.overall_status || 'unknown'}`
    },
    overallStatusText() {
      return String(this.healthData.overall_status || 'unknown').toUpperCase()
    },
    normalizedChecks() {
      const mapStatus = s => {
        if (s === 'healthy' || s === 'pass') return 'healthy'
        if (s === 'warning' || s === 'warn') return 'warning'
        if (s === 'critical' || s === 'fail') return 'critical'
        return 'unknown'
      }
      if (!Array.isArray(this.healthData.checks)) return []
      return this.healthData.checks.map(c => ({
        name: c?.name || 'Unknown check',
        status: mapStatus(c?.status),
        message: c?.message || 'No message',
        duration: c?.duration || '0ms',
        last_checked: c?.last_checked || '',
        details: c?.details && typeof c.details === 'object' ? c.details : null
      }))
    },
    issueChecks() {
      return this.normalizedChecks.filter(c => c.status === 'critical' || c.status === 'warning')
    },
    selectedRemedy() {
      return this.selectedCheck?.details?.remedy || this.fixResponse?.remedy || ''
    },
    selectedCommand() {
      return this.selectedCheck?.details?.command || this.fixResponse?.command || ''
    }
  },
  async mounted() {
    await this.loadHealth()
    this.refreshTimer = setInterval(() => this.loadHealth(), 60000)
  },
  beforeUnmount() {
    clearInterval(this.refreshTimer)
  },
  methods: {
    async loadHealth() {
      if (!this.$store.getters['auth/loggedIn']) return
      this.loading = true
      this.error = false
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getHealth()
        this.healthData = data || this.healthData
      } catch (err) {
        if (err.response?.status !== 401) this.error = true
      } finally {
        this.loading = false
      }
    },
    getCheckIcon(name) {
      return {
        'System Information': 'mdi mdi-desktop-classic',
        'Binary Status': 'mdi mdi-application',
        Database: 'mdi mdi-database',
        'Service Status': 'mdi mdi-cog',
        'Sudoers Configuration': 'mdi mdi-shield-key',
        'File Permissions': 'mdi mdi-lock',
        'Network Connectivity': 'mdi mdi-network',
        'Disk Space': 'mdi mdi-harddisk',
        Dependencies: 'mdi mdi-puzzle'
      }[name] || 'mdi mdi-information-outline'
    },
    getStatusIcon(status) {
      return {
        healthy: 'mdi mdi-check-circle',
        warning: 'mdi mdi-alert-circle',
        critical: 'mdi mdi-alert-octagon',
        unknown: 'mdi mdi-help-circle'
      }[status] || 'mdi mdi-help-circle'
    },
    showCheckDetails(check) {
      this.selectedCheck = check
      this.fixResponse = null
    },
    async fixIssue(action) {
      if (!this.selectedCheck) return
      this.fixing = true
      this.fixResponse = null
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.fixHealthIssue({ check_name: this.selectedCheck.name, action })
        this.fixResponse = data
        if (action === 'auto') setTimeout(() => this.loadHealth(), 1500)
      } catch (err) {
        this.fixResponse = {
          success: false,
          message: err.response?.data?.error || err.message || 'Failed to run fix',
          remedy: 'Run remediation command manually on the server.'
        }
      } finally {
        this.fixing = false
      }
    },
    copyCommand(command) {
      navigator.clipboard.writeText(command).catch(() => {})
    },
    formatTime(ts) {
      if (!ts) return 'Unknown'
      return new Date(ts).toLocaleTimeString()
    }
  }
}
</script>

<style scoped>
.health-card-body { min-height: 420px; }
.health-loading { display: flex; flex-direction: column; gap: 8px; }
.shimmer-line { height: 16px; border-radius: 4px; background: linear-gradient(90deg, #1e2a3a 25%, #263040 50%, #1e2a3a 75%); background-size: 200% 100%; animation: shimmer-sweep 1.4s infinite; }
@keyframes shimmer-sweep { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
.health-error { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 2rem 0; color: #5a7499; font-size: 0.82rem; }
.health-content { display: flex; flex-direction: column; gap: 10px; }
.health-summary { display: flex; justify-content: space-between; font-size: 0.75rem; color: #5a7499; padding: 8px 0; border-bottom: 1px solid #2a3f5f; }
.summary-item { display: flex; align-items: center; gap: 4px; }
.overall-message { display: flex; align-items: center; gap: 8px; padding: 10px; border-radius: 6px; font-size: 0.82rem; }
.status-healthy { background: rgba(40,167,69,0.12); color: #28a745; border: 1px solid rgba(40,167,69,0.25); }
.status-warning { background: rgba(255,193,7,0.12); color: #ffc107; border: 1px solid rgba(255,193,7,0.25); }
.status-critical { background: rgba(220,53,69,0.12); color: #dc3545; border: 1px solid rgba(220,53,69,0.25); }
.status-unknown { background: rgba(108,117,125,0.12); color: #6c757d; border: 1px solid rgba(108,117,125,0.25); }
.score-badge { background: rgba(74,158,255,0.1); color: #4a9eff; border: 1px solid rgba(74,158,255,0.3); padding: 2px 6px; border-radius: 10px; font-size: 0.72rem; }
.issue-strip { display: flex; flex-wrap: wrap; gap: 6px; }
.issue-chip { border: 1px solid transparent; border-radius: 999px; background: rgba(90,116,153,0.16); color: #8aa4c8; font-size: 0.7rem; padding: 4px 8px; display: inline-flex; align-items: center; gap: 4px; cursor: pointer; }
.issue-chip.warning { border-color: rgba(245,166,35,0.45); color: #f5a623; }
.issue-chip.critical { border-color: rgba(240,64,64,0.45); color: #f04040; }
.health-checks { display: flex; flex-direction: column; gap: 6px; max-height: 270px; overflow-y: auto; padding-right: 4px; }
.health-check-item { display: flex; flex-direction: column; gap: 3px; padding: 8px 10px; border-radius: 6px; border: 1px solid #2a3f5f; cursor: pointer; }
.health-check-item:hover { background: #1e2a3a; }
.health-check-item.healthy { border-left: 3px solid #28a745; }
.health-check-item.warning { border-left: 3px solid #ffc107; }
.health-check-item.critical { border-left: 3px solid #dc3545; }
.health-check-item.unknown { border-left: 3px solid #6c757d; }
.health-check-item.has-issue { background: rgba(220,53,69,0.05); }
.check-header { display: flex; align-items: center; gap: 8px; }
.check-name { font-size: 0.82rem; font-weight: 500; flex: 1; }
.check-message { font-size: 0.75rem; color: #8ca2c0; }
.check-time { font-size: 0.7rem; color: #5a7499; text-align: right; }
.issue-badge { font-size: 0.62rem; padding: 2px 6px; border-radius: 4px; }
.issue-badge.critical { background: rgba(220,53,69,0.2); color: #dc3545; }
.issue-badge.warning { background: rgba(255,193,7,0.2); color: #ffc107; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-content { background: #1a2332; border: 1px solid #2a3f5f; border-radius: 8px; max-width: 640px; max-height: 82vh; overflow: hidden; margin: 20px; }
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 14px 18px; border-bottom: 1px solid #2a3f5f; }
.modal-header h6 { margin: 0; display: flex; align-items: center; gap: 8px; }
.btn-close { background: none; border: none; color: #5a7499; }
.modal-body { padding: 16px; max-height: 65vh; overflow-y: auto; }
.check-status { display: inline-flex; align-items: center; gap: 8px; padding: 7px 10px; border-radius: 6px; margin-bottom: 12px; }
.check-message-full { color: #8ca2c0; margin-bottom: 10px; }
.check-meta { font-size: 0.82rem; color: #5a7499; margin-bottom: 12px; }
.fix-section { margin-top: 12px; padding: 12px; border: 1px solid rgba(74,158,255,0.2); border-radius: 6px; background: rgba(74,158,255,0.05); }
.fix-section h6 { margin: 0 0 8px 0; color: #4a9eff; }
.fix-actions { display: flex; gap: 8px; margin-top: 10px; }
.fix-response { padding: 8px; border-radius: 6px; margin-top: 10px; font-size: 0.82rem; }
.fix-response.success { color: #28a745; background: rgba(40,167,69,0.12); }
.fix-response.error { color: #dc3545; background: rgba(220,53,69,0.12); }
.remedy-text { color: #c9d8f0; font-size: 0.84rem; margin-bottom: 8px; }
.command-label { color: #8aa4c8; font-size: 0.8rem; margin-bottom: 6px; }
.command-text, .check-details pre { background: #0f1419; border: 1px solid #2a3f5f; border-radius: 4px; padding: 10px; font-size: 0.75rem; color: #8ca2c0; overflow-x: auto; }
</style>
