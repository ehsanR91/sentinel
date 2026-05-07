<template>
  <div>
    <PageHeader title="Updates" icon="mdi mdi-download" :items="[{text:'Updates',active:true,icon:'mdi mdi-package-variant-closed'}]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" :disabled="loading" @click="fetchUpdates">
          <span v-if="loading" class="spinner-border spinner-border-sm me-2"></span>
          <i v-else class="mdi mdi-refresh me-1"></i>
          {{ loading ? 'Checking…' : 'Check Now' }}
        </button>
      </template>
    </PageHeader>

    <!-- apt not available notice -->
    <div
      v-if="!aptAvailable && !loading"
      class="alert d-flex align-items-center gap-2 mb-4 py-2 update-notice-alert"
    >
      <i class="mdi mdi-information-outline" style="font-size:1rem"></i>
      apt not available on this system
    </div>

    <!-- Status bar -->
    <div
      v-else
      class="card mb-4"
      :style="`border-color:${packages.length ? 'rgba(245,166,35,0.3)' : 'rgba(34,214,124,0.3)'}`"
    >
      <div class="card-body py-2 d-flex align-items-center gap-3">
        <i
          v-if="loading"
          class="mdi mdi-loading mdi-spin update-status-icon"
        ></i>
        <i
          v-else
          :class="`mdi ${packages.length ? 'mdi-alert-circle' : 'mdi-check-circle'} update-status-icon`"
          :style="`color:${packages.length ? 'var(--sc-amber)' : 'var(--sc-green)'}`"
        ></i>
        <div>
          <div class="update-status-text">
            <template v-if="loading">Checking for updates…</template>
            <template v-else-if="packages.length">
              {{ packages.length }} package update{{ packages.length !== 1 ? 's' : '' }} available
            </template>
            <template v-else>System is up to date</template>
          </div>
          <div class="update-last-checked">Last checked: {{ lastChecked }}</div>
          <div class="update-last-checked">Last updated: {{ formatLastUpdated(lastUpdated) }}</div>
        </div>
        <div class="ms-auto d-flex gap-2">
          <button
            class="btn btn-sm update-install-btn"
            :disabled="!selectedCount || installing || loading"
            @click="installSelected"
          >
            <span v-if="installing" class="spinner-border spinner-border-sm me-1"></span>
            <i v-else class="mdi mdi-download me-1"></i>Install Selected
          </button>
          <button
            class="btn btn-sm btn-sc-primary"
            :disabled="!packages.length || installing || loading"
            @click="installAll"
          >
            <span v-if="installing" class="spinner-border spinner-border-sm me-1"></span>
            <i v-else class="mdi mdi-download-multiple me-1"></i>Install All
          </button>
        </div>
      </div>
    </div>

    <!-- Live Update Log Window -->
    <div v-if="showLogWindow" class="card mb-4 update-log-card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-terminal me-2" style="color:var(--sc-cyan)"></i>Live Update Log</h6>
        <div class="d-flex gap-2">
          <span class="status-dot" :class="installing ? 'online' : 'offline'"></span>
          <button class="btn btn-sm btn-sc-danger" @click="stopUpdate">
            <i class="mdi mdi-stop me-1"></i>Stop
          </button>
          <button class="btn btn-sm" style="background:rgba(90,116,153,.1);color:#5a7499" @click="showLogWindow=false">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
      </div>
      <div class="card-body p-0">
        <div class="update-log-window" ref="updateLogWindow">
          <div v-for="(line, idx) in updateLogs" :key="idx" class="update-log-line font-mono">
            <span class="log-timestamp">{{ line.ts }}</span>
            <span :class="`log-content ${line.type}`">{{ line.text }}</span>
          </div>
          <div v-if="installing" class="update-log-line font-mono log-cursor">
            <span class="log-timestamp">—</span>
            <span class="log-content">running...</span>
          </div>
        </div>
      </div>
      <div class="card-footer d-flex align-items-center justify-content-between">
        <button class="btn btn-sm" style="background:rgba(74,158,255,.1);color:#4a9eff;font-size:.72rem" @click="clearLogs">
          <i class="mdi mdi-delete-empty me-1"></i>Clear
        </button>
        <span style="font-size:.72rem;color:var(--sc-text-muted)">{{ updateLogs.length }} lines</span>
      </div>
    </div>

    <!-- Stat cards -->
    <div class="row g-3 mb-4">
      <div class="col-xl-3 col-md-6">
        <StatCard
          label="Available"
          :value="packages.length"
          sub="system packages"
          icon="mdi mdi-package-variant"
          icon-color="#f5a623"
          icon-bg="rgba(245,166,35,0.12)"
        />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard
          label="Security"
          :value="securityCount"
          sub="security patches"
          icon="mdi mdi-shield-alert"
          icon-color="#f04040"
          icon-bg="rgba(240,64,64,0.12)"
        />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard
          label="SentinelCore"
          value="v1.0.0"
          sub="current version"
          icon="mdi mdi-shield-half-full"
          icon-color="#4a9eff"
          icon-bg="rgba(74,158,255,0.12)"
        />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard
          label="Kernel"
          :value="kernelVersion || '—'"
          sub="current"
          icon="mdi mdi-linux"
          icon-color="#a78bfa"
          icon-bg="rgba(167,139,250,0.12)"
        />
      </div>
    </div>

    <!-- Package table -->
    <div class="card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-package-variant me-2" style="color:#f5a623"></i>Available Packages</h6>
        <div class="d-flex align-items-center gap-2">
          <span class="badge" style="background:rgba(240,64,64,0.12);color:#f04040;font-size:0.65rem">
            {{ securityCount }} security
          </span>
          <input
            v-model="pkgFilter"
            class="form-control form-control-sm"
            placeholder="Filter…"
            style="width:160px"
          />
        </div>
      </div>
      <div class="card-body p-0" style="max-height:360px;overflow-y:auto">
        <!-- Loading state -->
        <div v-if="loading" class="text-center py-4 update-empty-state">
          <span class="spinner-border spinner-border-sm me-2"></span>Loading…
        </div>

        <!-- No results from filter -->
        <div
          v-else-if="pkgFilter && !filteredPkgs.length"
          class="text-center py-4 update-empty-state"
        >
          No packages match "{{ pkgFilter }}"
        </div>

        <!-- Empty / up to date -->
        <div
          v-else-if="!packages.length"
          class="text-center py-4 update-empty-state"
        >
          <i class="mdi mdi-check-circle me-1" style="color:var(--sc-green)"></i>No updates available
        </div>

        <!-- Package list -->
        <table v-else class="table mb-0">
          <thead>
            <tr>
              <th style="width:40px">
                <input type="checkbox" :checked="allSelected" @change="toggleAll" />
              </th>
              <th>Package</th>
              <th>Current</th>
              <th>New Version</th>
              <th>Type</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="pkg in filteredPkgs" :key="pkg.name">
              <td><input type="checkbox" v-model="pkg.selected" /></td>
              <td class="font-mono update-pkg-name">{{ pkg.name }}</td>
              <td class="font-mono update-pkg-old">{{ pkg.old_ver }}</td>
              <td class="font-mono update-pkg-new">{{ pkg.new_ver }}</td>
              <td>
                <span
                  v-if="pkg.is_security"
                  class="badge rounded-pill badge-offline"
                  style="font-size:0.62rem"
                >security</span>
                <span v-else class="badge rounded-pill badge-info" style="font-size:0.62rem">standard</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard   from '@/components/widgets/stat-card.vue'

export default {
  name: 'UpdatesPage',
  components: { PageHeader, StatCard },

  data() {
    return {
      loading: false,
      installing: false,
      lastChecked: '—',
      kernelVersion: '',
      lastUpdated: null,
      aptAvailable: true,
      pkgFilter: '',
      packages: [],
      // Live log window
      showLogWindow: false,
      updateLogs: [],
      logPollTimer: null
    }
  },

  computed: {
    securityCount() {
      return this.packages.filter(p => p.is_security).length
    },
    selectedCount() {
      return this.packages.filter(p => p.selected).length
    },
    allSelected() {
      return this.packages.length > 0 && this.packages.every(p => p.selected)
    },
    filteredPkgs() {
      if (!this.pkgFilter) return this.packages
      const q = this.pkgFilter.toLowerCase()
      return this.packages.filter(p => p.name.toLowerCase().includes(q))
    }
  },

  async mounted() {
    await this.fetchUpdates()
  },

  methods: {
    async fetchUpdates() {
      this.loading = true
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getUpdates()
        this.aptAvailable = data.apt_available !== false
        this.kernelVersion = data.kernel || ''
        this.lastUpdated = data.last_updated || null
        this.packages = (data.packages || []).map(p => ({ ...p, selected: false }))
        this.lastChecked = new Date().toLocaleTimeString()
      } catch (err) {
        const msg = err.response?.data?.error || 'Failed to check for updates'
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: msg, showConfirmButton: false, timer: 3000 })
      } finally {
        this.loading = false
      }
    },

    toggleAll(e) {
      this.packages.forEach(p => (p.selected = e.target.checked))
    },

    async _runInstall(pkgNames) {
      this.installing = true
      this.showLogWindow = true
      this.updateLogs = []
      this.addLogLine('Starting update process...', 'info')
      try {
        const api = (await import('@/services/api')).default
        await api.installUpdates(pkgNames)
        this.addLogLine('Update request sent to server', 'success')
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Update started in the background', showConfirmButton: false, timer: 3000 })
        // Start polling for logs
        this.startLogPolling()
      } catch (err) {
        const msg = err.response?.data?.error || 'Failed to start update'
        this.addLogLine(`Error: ${msg}`, 'error')
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: msg, showConfirmButton: false, timer: 3000 })
        this.installing = false
      }
    },

    startLogPolling() {
      this.logPollTimer = setInterval(() => {
        this.fetchUpdateLogs()
      }, 2000)
    },

    async fetchUpdateLogs() {
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getUpdateLogs()
        if (data.logs && data.logs.length > 0) {
          const newLogs = data.logs.filter(log => !this.updateLogs.find(el => el.text === log.text))
          newLogs.forEach(log => {
            const type = log.text.toLowerCase().includes('error') ? 'error' :
                        log.text.toLowerCase().includes('ok') ? 'success' : 'info'
            this.addLogLine(log.text, type)
          })
        }
        if (data.done) {
          this.addLogLine('Update process completed', 'success')
          if (data.last_updated) {
            this.lastUpdated = data.last_updated
            this.addLogLine(`Last successful update: ${this.formatLastUpdated(data.last_updated)}`, 'info')
          }
          this.stopLogPolling()
          this.installing = false
          await this.fetchUpdates()
        }
      } catch (err) {
        // Silent fail for polling
      }
    },

    stopLogPolling() {
      if (this.logPollTimer) {
        clearInterval(this.logPollTimer)
        this.logPollTimer = null
      }
    },

    addLogLine(text, type = 'info') {
      const now = new Date()
      const ts = now.toLocaleTimeString()
      this.updateLogs.push({ ts, text, type })
      // Auto-scroll to bottom
      this.$nextTick(() => {
        const container = this.$refs.updateLogWindow
        if (container) {
          container.scrollTop = container.scrollHeight
        }
      })
    },

    clearLogs() {
      this.updateLogs = []
    },

    stopUpdate() {
      this.stopLogPolling()
      this.installing = false
      this.addLogLine('Update stopped by user', 'warn')
    },

    installSelected() {
      const names = this.packages.filter(p => p.selected).map(p => p.name)
      if (!names.length) return
      this.$swal({
        title: `Install ${names.length} selected package${names.length !== 1 ? 's' : ''}?`,
        icon: 'question',
        showCancelButton: true,
        confirmButtonText: 'Install',
        confirmButtonColor: '#4a9eff'
      }).then(r => { if (r.isConfirmed) this._runInstall(names) })
    },

    installAll() {
      this.$swal({
        title: 'Install all updates?',
        text: 'This will run apt-get upgrade in the background.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Install All',
        confirmButtonColor: '#4a9eff'
      }).then(r => { if (r.isConfirmed) this._runInstall(this.packages.map(p => p.name)) })
    },

    formatLastUpdated(ts) {
      if (!ts) return 'Never'
      return new Date(ts).toLocaleString()
    }
  }
}
</script>

<style scoped>
/* Update Page Styles */
.update-notice-alert {
  background: rgba(90, 116, 153, 0.1);
  border: 1px solid rgba(90, 116, 153, 0.25);
  color: var(--sc-text-secondary);
  font-size: 0.82rem;
}

.update-status-icon {
  font-size: 1.3rem;
}

.update-status-text {
  font-weight: 600;
  font-size: 0.85rem;
  color: var(--sc-text);
}

.update-last-checked {
  font-size: 0.75rem;
  color: var(--sc-text-muted);
}

.update-install-btn {
  background: var(--sc-bg-primary-subtle);
  color: var(--sc-primary);
}

.update-empty-state {
  color: var(--sc-text-muted);
  font-size: 0.85rem;
}

.update-pkg-name {
  font-size: 0.78rem;
  color: var(--sc-text);
}

.update-pkg-old {
  font-size: 0.72rem;
  color: var(--sc-text-muted);
}

.update-pkg-new {
  font-size: 0.72rem;
  color: var(--sc-green);
}

/* Update Log Window */
.update-log-card {
  border-color: var(--sc-border);
}

.update-log-window {
  max-height: 300px;
  overflow-y: auto;
  background: var(--sc-bg-secondary);
  padding: 12px;
  font-size: 0.75rem;
}

.update-log-line {
  display: flex;
  gap: 12px;
  padding: 4px 0;
  border-bottom: 1px solid var(--sc-border);
}

.update-log-line:last-child {
  border-bottom: none;
}

.log-timestamp {
  color: var(--sc-text-muted);
  min-width: 80px;
  font-size: 0.72rem;
}

.log-content {
  color: var(--sc-text);
  flex: 1;
  word-break: break-all;
}

.log-content.info {
  color: var(--sc-text);
}

.log-content.success {
  color: var(--sc-green);
}

.log-content.error {
  color: var(--sc-red);
}

.log-content.warn {
  color: var(--sc-amber);
}

.log-cursor .log-content {
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  50% { opacity: 0; }
}
</style>
