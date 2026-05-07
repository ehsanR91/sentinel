<template>
<div class="sc-view sc-view-apps">
  <PageHeader title="Apps" icon="mdi mdi-apps" :items="[{ text: 'Apps', active: true, icon: 'mdi mdi-apps' }]">
    <template #actions>
      <button class="btn btn-sm btn-sc-primary" :disabled="loading" @click="refreshApps">
        <i :class="`mdi ${loading ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>
        Refresh
      </button>
    </template>
  </PageHeader>

  <!-- Live Operation Log Window -->
  <div v-if="showLogWindow" class="card mb-4 app-log-card">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6>
        <i class="mdi mdi-terminal me-2" style="color:var(--sc-cyan)"></i>
        {{ opKindLabel }} Log
        <span class="ms-2 font-mono" style="font-size:0.72rem;color:var(--sc-text-muted)">{{ currentOpApp }}</span>
      </h6>
      <div class="d-flex gap-2 align-items-center">
        <span class="status-dot" :class="opRunning ? 'online pulsing' : (opError ? 'offline' : 'warn')"></span>
        <span style="font-size:0.72rem;color:var(--sc-text-muted)">
          {{ opRunning ? 'running...' : (opError ? 'failed' : 'done') }}
        </span>
        <button class="btn btn-sm" style="background:rgba(90,116,153,.1);color:#5a7499" @click="closeLogWindow">
          <i class="mdi mdi-close"></i>
        </button>
      </div>
    </div>
    <div class="card-body p-0">
      <div class="app-log-window" ref="logWindow">
        <div v-for="(line, idx) in opLogs" :key="idx" class="app-log-line font-mono">
          <span class="log-ts">{{ line.ts }}</span>
          <span :class="`log-text ${line.type}`">{{ line.text }}</span>
        </div>
        <div v-if="opRunning" class="app-log-line font-mono log-cursor-line">
          <span class="log-ts">—</span>
          <span class="log-text">running...</span>
        </div>
        <div v-if="opError" class="app-log-line font-mono">
          <span class="log-ts">ERR</span>
          <span class="log-text error">{{ opError }}</span>
        </div>
      </div>
    </div>
    <div class="card-footer d-flex align-items-center justify-content-between">
      <button class="btn btn-sm" style="background:rgba(74,158,255,.1);color:#4a9eff;font-size:.72rem" @click="opLogs = []">
        <i class="mdi mdi-delete-empty me-1"></i>Clear
      </button>
      <span style="font-size:.72rem;color:var(--sc-text-muted)">{{ opLogs.length }} lines</span>
    </div>
  </div>

  <!-- Stat cards -->
  <div class="row g-3 mb-4">
    <div class="col-xl-3 col-md-6">
      <StatCard label="Total Apps" :value="apps.length" sub="in catalog" icon="mdi mdi-apps" icon-color="#4a9eff" icon-bg="rgba(74,158,255,.12)" />
    </div>
    <div class="col-xl-3 col-md-6">
      <StatCard label="Installed" :value="installedCount" sub="on this server" icon="mdi mdi-check-circle-outline" icon-color="#22d67c" icon-bg="rgba(34,214,124,.12)" />
    </div>
    <div class="col-xl-3 col-md-6">
      <StatCard label="Updates Available" :value="updatesCount" sub="can be upgraded" icon="mdi mdi-update" icon-color="#f5a623" icon-bg="rgba(245,166,35,.12)" />
    </div>
    <div class="col-xl-3 col-md-6">
      <StatCard label="Not Installed" :value="notInstalledCount" sub="available to install" icon="mdi mdi-package-variant" icon-color="#a78bfa" icon-bg="rgba(167,139,250,.12)" />
    </div>
  </div>

  <!-- Filter + category tabs -->
  <div class="card sc-panel-card mb-4">
  <div class="card-header d-flex align-items-center justify-content-between flex-wrap gap-2">
  <div class="d-flex gap-2 flex-wrap align-items-center">
  <h6 class="mb-0"><i class="mdi mdi-package-variant-closed me-2" style="color:var(--sc-blue)"></i>Managed Apps</h6>
  <div class="category-pills-wrapper">
  <button
  v-for="cat in allCategories" :key="cat"
  class="btn btn-sm cat-tab"
  :class="{ active: activeCategory === cat }"
  @click="activeCategory = cat"
  >
  <i :class="`mdi ${categoryIcon(cat)} me-1`"></i>{{ cat === 'all' ? 'All' : catLabel(cat) }}
  <span v-if="cat !== 'all'" class="cat-count">{{ countByCategory(cat) }}</span>
  </button>
  </div>
  </div>
  <input v-model="q" class="form-control form-control-sm" placeholder="Search apps..." style="width:200px" />
  </div>
  
  <!-- Installed / All toggle -->
  <div class="card-body border-bottom p-2" style="background:rgba(10,22,40,.5)">
  <div class="d-flex align-items-center gap-2">
  <span class="filter-label" style="font-size:0.85rem;color:var(--sc-text-muted)">Show:</span>
  <button
  class="btn btn-sm"
  :class="showInstalledOnly ? 'btn-sc-primary' : ''"
  style="background:rgba(90,116,153,.1);color:var(--sc-text-muted)"
  @click="showInstalledOnly = !showInstalledOnly"
  >
  <i :class="`mdi me-1 ${showInstalledOnly ? 'mdi-check-circle' : 'mdi-apps'}`"></i>
  {{ showInstalledOnly ? 'Installed Only' : 'All Apps' }}
  </button>
  </div>
  </div>

    <div class="card-body p-0">
      <!-- Loading skeleton -->
      <div v-if="loading" class="p-4 text-center sc-text-muted">
        <i class="mdi mdi-loading mdi-spin me-2"></i>Loading apps and checking versions...
      </div>

      <div v-else-if="!filtered.length" class="p-4 text-center sc-text-muted">
        No apps match your filter.
      </div>

      <!-- App cards grid -->
      <div v-else class="app-grid p-3">
        <div
          v-for="app in filtered"
          :key="app.name"
          class="app-card"
          :class="{
            'app-card--installed': app.installed,
            'app-card--update': app.update_avail,
            'app-card--busy': isBusy(app),
          }"
        >
          <!-- Header row -->
          <div class="app-card-header">
            <div class="app-icon" :class="`app-icon--${app.category}`">
              <i :class="`mdi ${categoryIcon(app.category)}`"></i>
            </div>
            <div class="app-meta">
              <div class="app-label">{{ app.label }}</div>
              <div class="d-flex gap-1 flex-wrap mt-1">
                <span class="badge app-cat-badge" :class="`cat-${app.category}`">{{ catLabel(app.category) }}</span>
                <span class="badge app-method-badge">{{ methodLabel(app.install_method) }}</span>
              </div>
            </div>
            <div class="app-status-wrap">
              <span class="badge rounded-pill app-status-badge" :class="statusClass(app)">
                <i :class="`mdi ${statusIcon(app)} me-1`"></i>
                {{ statusLabel(app) }}
              </span>
            </div>
          </div>

          <!-- Description -->
          <p class="app-desc">{{ app.description }}</p>

          <!-- Version row -->
          <div class="app-version-row" v-if="app.installed || app.new_version">
            <div v-if="app.version" class="version-chip installed-ver">
              <i class="mdi mdi-tag-outline me-1"></i>
              <span>{{ app.version }}</span>
              <span class="ver-label">installed</span>
            </div>
            <div v-if="app.new_version" class="version-chip" :class="app.update_avail ? 'new-ver-avail' : 'new-ver-same'">
              <i class="mdi mdi-arrow-up-circle-outline me-1"></i>
              <span>{{ app.new_version }}</span>
              <span class="ver-label">{{ app.update_avail ? 'update available' : 'latest' }}</span>
            </div>
          </div>

          <!-- Actions -->
          <div class="app-actions">
            <!-- Install -->
            <button
              v-if="!app.installed"
              class="btn btn-sm btn-app-install"
              :disabled="isBusy(app) || anyOpRunning"
              @click="installApp(app)"
            >
              <i :class="`mdi ${isBusy(app) ? 'mdi-loading mdi-spin' : 'mdi-package-down'} me-1`"></i>
              {{ isBusy(app) ? 'Installing...' : 'Install' }}
            </button>

            <!-- Update -->
            <button
              v-if="app.installed && app.update_avail"
              class="btn btn-sm btn-app-update"
              :disabled="isBusy(app) || anyOpRunning"
              @click="updateApp(app)"
            >
              <i :class="`mdi ${isBusy(app) && app.status === 'updating' ? 'mdi-loading mdi-spin' : 'mdi-arrow-up-circle'} me-1`"></i>
              {{ isBusy(app) && app.status === 'updating' ? 'Updating...' : 'Update' }}
            </button>

            <!-- Up to date indicator -->
            <span v-if="app.installed && !app.update_avail && app.new_version" class="app-uptodate">
              <i class="mdi mdi-check-circle me-1"></i>Up to date
            </span>

            <!-- View logs button (when this app is being operated on) -->
            <button
              v-if="isBusy(app)"
              class="btn btn-sm btn-app-logs"
              @click="showLogWindow = true"
            >
              <i class="mdi mdi-text-box-outline me-1"></i>View Log
            </button>

            <!-- Homepage link -->
            <a
              v-if="app.homepage"
              :href="app.homepage"
              target="_blank"
              rel="noopener"
              class="btn btn-sm btn-app-info"
              title="Open homepage"
            >
              <i class="mdi mdi-open-in-new"></i>
            </a>

            <!-- Uninstall (installed apps only) -->
            <button
              v-if="app.installed"
              class="btn btn-sm btn-app-remove"
              :disabled="isBusy(app) || anyOpRunning"
              @click="uninstallApp(app)"
              title="Uninstall"
            >
              <i :class="`mdi ${isBusy(app) && app.status === 'uninstalling' ? 'mdi-loading mdi-spin' : 'mdi-delete-outline'}`"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard from '@/components/widgets/stat-card.vue'
import api from '@/services/api'

const CAT_LABELS = {
  all: 'All',
  cli: 'CLI Tools',
  runtime: 'Runtimes',
  web: 'Web Server',
  database: 'Database',
  build: 'Build Tools',
  devtool: 'Dev Tools',
  shell: 'Shell',
}

const CAT_ICONS = {
  all: 'mdi-apps',
  cli: 'mdi-console-line',
  runtime: 'mdi-code-braces',
  web: 'mdi-web',
  database: 'mdi-database',
  build: 'mdi-hammer-wrench',
  devtool: 'mdi-tools',
  shell: 'mdi-bash',
}

const METHOD_LABELS = {
  pkg: 'Package',
  script: 'Script',
  binary: 'Binary',
  rustup: 'rustup',
}

export default {
  name: 'AppsPage',
  components: { PageHeader, StatCard },
  data() {
    return {
      loading: false,
      apps: [],
      q: '',
      activeCategory: 'all',
      showInstalledOnly: false,
      // Operation state
      showLogWindow: false,
      opRunning: false,
      opLogs: [],
      opError: '',
      opKind: '',
      currentOpApp: '',
      opSeenCount: 0,
      opPollTimer: null,
      // Per-app busy map (optimistic UI)
      busyApps: {},
    }
  },
  computed: {
    filtered() {
      let list = this.apps
      if (this.activeCategory !== 'all') {
        list = list.filter(a => a.category === this.activeCategory)
      }
      if (this.showInstalledOnly) {
        list = list.filter(a => a.installed)
      }
      if (this.q) {
        const t = this.q.toLowerCase()
        list = list.filter(a =>
          a.name.toLowerCase().includes(t) ||
          a.label.toLowerCase().includes(t) ||
          a.description.toLowerCase().includes(t) ||
          a.category.toLowerCase().includes(t)
        )
      }
      return list
    },
    installedCount() {
      return this.apps.filter(a => a.installed).length
    },
    updatesCount() {
      return this.apps.filter(a => a.update_avail).length
    },
    notInstalledCount() {
      return this.apps.filter(a => !a.installed).length
    },
    allCategories() {
      const cats = ['all', ...new Set(this.apps.map(a => a.category))]
      return cats
    },
    anyOpRunning() {
      return this.opRunning || Object.values(this.busyApps).some(Boolean)
    },
    opKindLabel() {
      const map = { install: 'Installation', update: 'Update', uninstall: 'Uninstall' }
      return map[this.opKind] || 'Operation'
    },
  },
  async mounted() {
    await this.loadApps()
    // Resume polling if server says an op is running
    const { data } = await api.getAppOpLogs().catch(() => ({ data: {} }))
    if (data.running) {
      this.opRunning = true
      this.currentOpApp = data.app || ''
      this.opKind = data.kind || ''
      this.showLogWindow = true
      this.startPolling()
    }
  },
  beforeUnmount() {
    this.stopPolling()
  },
  methods: {
    // ── Data loading ────────────────────────────────────────────────────────
    async loadApps() {
      this.loading = true
      try {
        // Add cache-busting timestamp to ensure fresh data
        const { data } = await api.getApps()
        this.apps = data || []
        // Sync busyApps from server state
        this.apps.forEach(a => {
          if (['installing', 'updating', 'uninstalling'].includes(a.status)) {
            this.busyApps[a.name] = true
            if (!this.opRunning) {
              this.opRunning = true
              this.currentOpApp = a.label
              this.opKind = a.status.replace('ing', '')
              this.showLogWindow = true
              this.startPolling()
            }
          }
        })
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Failed to load apps', text: e.response?.data?.error || e.message })
      } finally {
        this.loading = false
      }
    },
  
    // Force refresh with cache invalidation
    async refreshApps() {
      // Clear apps first to force fresh render
      this.apps = []
      await this.loadApps()
    },

    // ── Category helpers ────────────────────────────────────────────────────
    catLabel(cat) { return CAT_LABELS[cat] || cat },
    categoryIcon(cat) { return CAT_ICONS[cat] || 'mdi-package' },
    methodLabel(m) { return METHOD_LABELS[m] || m },
    countByCategory(cat) {
      return this.apps.filter(a => a.category === cat).length
    },

    // ── Status display ──────────────────────────────────────────────────────
    statusLabel(app) {
      if (app.status === 'installing') return 'Installing...'
      if (app.status === 'updating') return 'Updating...'
      if (app.status === 'uninstalling') return 'Removing...'
      if (app.status === 'failed') return 'Failed'
      if (app.installed && app.update_avail) return 'Update Available'
      if (app.installed) return 'Installed'
      return 'Not Installed'
    },
    statusClass(app) {
      if (['installing', 'updating', 'uninstalling'].includes(app.status)) return 'badge-installing'
      if (app.status === 'failed') return 'badge-offline'
      if (app.installed && app.update_avail) return 'badge-update'
      if (app.installed) return 'badge-online'
      return 'badge-warning'
    },
    statusIcon(app) {
      if (['installing', 'updating', 'uninstalling'].includes(app.status)) return 'mdi-loading mdi-spin'
      if (app.status === 'failed') return 'mdi-alert-circle'
      if (app.installed && app.update_avail) return 'mdi-arrow-up-circle'
      if (app.installed) return 'mdi-check-circle'
      return 'mdi-minus-circle-outline'
    },
    isBusy(app) {
      return this.busyApps[app.name] ||
        ['installing', 'updating', 'uninstalling'].includes(app.status)
    },

    // ── Actions ─────────────────────────────────────────────────────────────
    async installApp(app) {
      const r = await this.$swal({
        title: `Install ${app.label}?`,
        html: `<div style="font-size:.9rem;color:#8aa4c8">${app.description}</div>`,
        icon: 'question',
        showCancelButton: true,
        confirmButtonText: 'Install',
      })
      if (!r.isConfirmed) return
      this.beginOp(app, 'install')
      try {
        await api.installApp(app.name)
        this.addLog(`Started installation of ${app.label}`, 'info')
        this.startPolling()
      } catch (e) {
        const msg = e.response?.data?.error || e.message
        this.addLog(`Error: ${msg}`, 'error')
        this.endOp(app, msg)
        this.$swal({ icon: 'error', title: `Install failed: ${app.label}`, text: msg })
      }
    },

    async updateApp(app) {
      const r = await this.$swal({
        title: `Update ${app.label}?`,
        html: `<div style="font-size:.9rem;color:#8aa4c8">
          Installed: <b>${app.version || '?'}</b> → Available: <b>${app.new_version || '?'}</b>
        </div>`,
        icon: 'question',
        showCancelButton: true,
        confirmButtonText: 'Update',
      })
      if (!r.isConfirmed) return
      this.beginOp(app, 'update')
      try {
        await api.updateApp(app.name)
        this.addLog(`Started update of ${app.label}`, 'info')
        this.startPolling()
      } catch (e) {
        const msg = e.response?.data?.error || e.message
        this.addLog(`Error: ${msg}`, 'error')
        this.endOp(app, msg)
        this.$swal({ icon: 'error', title: `Update failed: ${app.label}`, text: msg })
      }
    },

    async uninstallApp(app) {
      const r = await this.$swal({
        title: `Uninstall ${app.label}?`,
        text: 'This will remove the package from the system. Configuration files may be preserved.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Uninstall',
        confirmButtonColor: '#f04040',
      })
      if (!r.isConfirmed) return
      this.beginOp(app, 'uninstall')
      try {
        await api.uninstallApp(app.name)
        this.addLog(`Started uninstall of ${app.label}`, 'info')
        this.startPolling()
      } catch (e) {
        const msg = e.response?.data?.error || e.message
        this.addLog(`Error: ${msg}`, 'error')
        this.endOp(app, msg)
        this.$swal({ icon: 'error', title: `Uninstall failed: ${app.label}`, text: msg })
      }
    },

    // ── Op helpers ───────────────────────────────────────────────────────────
    beginOp(app, kind) {
      this.busyApps = { ...this.busyApps, [app.name]: true }
      // Optimistically update the app's status in the list
      const idx = this.apps.findIndex(a => a.name === app.name)
      if (idx >= 0) {
        this.apps[idx] = { ...this.apps[idx], status: kind + 'ing' }
      }
      this.opRunning = true
      this.opKind = kind
      this.currentOpApp = app.label
      this.opLogs = []
      this.opSeenCount = 0
      this.opError = ''
      this.showLogWindow = true
    },

    endOp(app, errMsg = '') {
      this.busyApps = { ...this.busyApps, [app.name]: false }
      this.opRunning = false
      this.opError = errMsg
      this.stopPolling()
    },

    // ── Log polling ──────────────────────────────────────────────────────────
    startPolling() {
      this.stopPolling()
      this.opPollTimer = setInterval(() => this.pollLogs(), 1500)
    },

    stopPolling() {
      if (this.opPollTimer) {
        clearInterval(this.opPollTimer)
        this.opPollTimer = null
      }
    },

    async pollLogs() {
      try {
        const { data } = await api.getAppOpLogs()
        // Append new log lines
        if (data.logs && data.logs.length > this.opSeenCount) {
          const newLines = data.logs.slice(this.opSeenCount)
          this.opSeenCount = data.logs.length
          newLines.forEach(line => {
            const lower = line.toLowerCase()
            const type = lower.includes('error') || lower.includes('fail') || lower.includes('err]') ? 'error'
              : lower.includes('===') || lower.includes('success') || lower.includes('installed') || lower.includes('ok') ? 'success'
              : lower.includes('warn') ? 'warn'
              : 'info'
            this.addLog(line, type)
          })
        }

        if (data.done) {
          this.stopPolling()
          this.opRunning = false

          if (data.error) {
            this.opError = data.error
            this.addLog(`Operation failed: ${data.error}`, 'error')
            this.$swal({
              icon: 'error',
              title: `${this.opKindLabel} failed`,
              text: data.error,
            })
          } else {
            this.addLog(`${this.opKindLabel} completed successfully`, 'success')
            this.$swal({
              toast: true,
              position: 'top-end',
              icon: 'success',
              title: `${this.currentOpApp} — ${this.opKindLabel} complete`,
              showConfirmButton: false,
              timer: 3000,
            })
          }

          // Clear busy map and reload fresh status
          this.busyApps = {}
          await this.loadApps()
        }
      } catch {
        // Silent fail for polling
      }
    },

    addLog(text, type = 'info') {
      const ts = new Date().toLocaleTimeString()
      this.opLogs.push({ ts, text, type })
      this.$nextTick(() => {
        const el = this.$refs.logWindow
        if (el) el.scrollTop = el.scrollHeight
      })
    },

    closeLogWindow() {
      this.showLogWindow = false
    },
  },
}
</script>

<style scoped>
.sc-text-muted { color: var(--sc-text-muted, #5a7499); }

/* ── Category pills wrapper (scrollable) ── */
.category-pills-wrapper {
  display: flex;
  gap: 0.25rem;
  overflow-x: auto;
  white-space: nowrap;
  padding-bottom: 2px;
  max-width: 600px;
}
.category-pills-wrapper::-webkit-scrollbar {
  height: 4px;
}
.category-pills-wrapper::-webkit-scrollbar-track {
  background: rgba(90, 116, 153, 0.1);
  border-radius: 2px;
}
.category-pills-wrapper::-webkit-scrollbar-thumb {
  background: rgba(74, 158, 255, 0.3);
  border-radius: 2px;
}
.category-pills-wrapper::-webkit-scrollbar-thumb:hover {
  background: rgba(74, 158, 255, 0.5);
}
.cat-tab {
  flex-shrink: 0;
}
.cat-count {
  margin-left: 4px;
  font-size: 0.65rem;
  opacity: 0.8;
}

/* ── App grid ── */
.app-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 1rem;
}

/* ── App card ── */
.app-card {
  background: var(--sc-bg-secondary, #0a1628);
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 10px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.app-card:hover {
  border-color: rgba(74,158,255,.3);
  box-shadow: 0 2px 12px rgba(0,0,0,.25);
}
.app-card--installed {
  border-color: rgba(34,214,124,.2);
}
.app-card--update {
  border-color: rgba(245,166,35,.3);
  box-shadow: 0 0 0 1px rgba(245,166,35,.15);
}
.app-card--busy {
  border-color: rgba(74,158,255,.4);
  animation: app-card-pulse 2s infinite;
}
@keyframes app-card-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(74,158,255,.2); }
  50%       { box-shadow: 0 0 0 4px rgba(74,158,255,.1); }
}

/* ── Card header ── */
.app-card-header {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.app-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.2rem;
  flex-shrink: 0;
}
.app-icon--cli      { background: rgba(74,158,255,.12); color: #4a9eff; }
.app-icon--runtime  { background: rgba(34,214,124,.12); color: #22d67c; }
.app-icon--web      { background: rgba(167,139,250,.12); color: #a78bfa; }
.app-icon--database { background: rgba(245,166,35,.12); color: #f5a623; }
.app-icon--build    { background: rgba(240,64,64,.12); color: #f04040; }
.app-icon--devtool  { background: rgba(0,208,255,.12); color: #00d0ff; }
.app-icon--shell    { background: rgba(100,116,139,.12); color: #64748b; }

.app-meta {
  flex: 1;
  min-width: 0;
}
.app-label {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--sc-text, #e2ecff);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.app-status-wrap {
  flex-shrink: 0;
}

/* ── Category / method badges ── */
.app-cat-badge {
  font-size: 0.6rem !important;
  padding: 2px 6px !important;
  text-transform: uppercase;
  letter-spacing: .04em;
}
.cat-cli      { background: rgba(74,158,255,.12) !important; color: #4a9eff !important; }
.cat-runtime  { background: rgba(34,214,124,.12) !important; color: #22d67c !important; }
.cat-web      { background: rgba(167,139,250,.12) !important; color: #a78bfa !important; }
.cat-database { background: rgba(245,166,35,.12)  !important; color: #f5a623 !important; }
.cat-build    { background: rgba(240,64,64,.12)   !important; color: #f04040 !important; }
.cat-devtool  { background: rgba(0,208,255,.12)   !important; color: #00d0ff !important; }
.cat-shell    { background: rgba(100,116,139,.12) !important; color: #8aa4c8 !important; }

.app-method-badge {
  font-size: 0.6rem !important;
  padding: 2px 6px !important;
  background: rgba(90,116,153,.12) !important;
  color: var(--sc-text-muted, #5a7499) !important;
}

/* ── Status badges ── */
.app-status-badge { font-size: 0.65rem !important; padding: 3px 8px !important; }
.badge-installing { background: rgba(74,158,255,.18) !important; color: #4a9eff !important; }
.badge-update     { background: rgba(245,166,35,.18) !important; color: #f5a623 !important; }
.badge-online     { background: rgba(34,214,124,.18) !important; color: #22d67c !important; }
.badge-warning    { background: rgba(90,116,153,.18) !important; color: #8aa4c8 !important; }
.badge-offline    { background: rgba(240,64,64,.18)  !important; color: #f04040 !important; }

/* ── Description ── */
.app-desc {
  font-size: 0.78rem;
  color: var(--sc-text-secondary, #8aa4c8);
  line-height: 1.5;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ── Version chips ── */
.app-version-row {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.version-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.7rem;
  padding: 3px 8px;
  border-radius: 20px;
  font-family: monospace;
}
.installed-ver {
  background: rgba(34,214,124,.1);
  color: #22d67c;
  border: 1px solid rgba(34,214,124,.2);
}
.new-ver-avail {
  background: rgba(245,166,35,.1);
  color: #f5a623;
  border: 1px solid rgba(245,166,35,.2);
}
.new-ver-same {
  background: rgba(90,116,153,.08);
  color: var(--sc-text-muted);
  border: 1px solid rgba(90,116,153,.15);
}
.ver-label {
  font-size: 0.6rem;
  text-transform: uppercase;
  letter-spacing: .04em;
  opacity: .7;
}

/* ── Actions ── */
.app-actions {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  align-items: center;
  margin-top: auto;
}
.btn-app-install {
  background: rgba(245,166,35,.12) !important;
  color: #f5a623 !important;
  font-size: 0.72rem !important;
  padding: 3px 10px !important;
}
.btn-app-install:disabled { opacity: .5; cursor: not-allowed; }
.btn-app-update {
  background: rgba(74,158,255,.12) !important;
  color: #4a9eff !important;
  font-size: 0.72rem !important;
  padding: 3px 10px !important;
}
.btn-app-update:disabled { opacity: .5; cursor: not-allowed; }
.btn-app-logs {
  background: rgba(167,139,250,.1) !important;
  color: #a78bfa !important;
  font-size: 0.72rem !important;
  padding: 3px 10px !important;
}
.btn-app-info {
  background: rgba(90,116,153,.08) !important;
  color: var(--sc-text-muted) !important;
  font-size: 0.72rem !important;
  padding: 3px 8px !important;
  border: 1px solid var(--sc-border) !important;
}
.btn-app-info:hover { color: #4a9eff !important; border-color: #4a9eff !important; }
.btn-app-remove {
  background: rgba(240,64,64,.08) !important;
  color: #f04040 !important;
  font-size: 0.72rem !important;
  padding: 3px 8px !important;
  margin-left: auto;
}
.btn-app-remove:disabled { opacity: .5; cursor: not-allowed; }
.app-uptodate {
  font-size: 0.7rem;
  color: var(--sc-text-muted);
}

/* ── Category filter tabs ── */
.cat-tab {
  font-size: 0.7rem !important;
  padding: 3px 10px !important;
  background: transparent !important;
  color: var(--sc-text-muted) !important;
  border: 1px solid var(--sc-border, #1e2d4a) !important;
  border-radius: 20px !important;
}
.cat-tab:hover {
  background: rgba(74,158,255,.08) !important;
  color: #4a9eff !important;
  border-color: rgba(74,158,255,.3) !important;
}
.cat-tab.active {
  background: rgba(74,158,255,.15) !important;
  color: #4a9eff !important;
  border-color: rgba(74,158,255,.4) !important;
}
.cat-count {
  display: inline-block;
  background: rgba(255,255,255,.08);
  border-radius: 10px;
  padding: 0 5px;
  font-size: 0.6rem;
  margin-left: 3px;
}

/* ── Log window ── */
.app-log-card {
  border: 1px solid var(--sc-border);
}
.app-log-window {
  height: 260px;
  overflow-y: auto;
  background: var(--sc-bg-secondary);
  padding: 0.75rem;
  font-size: 0.77rem;
}
.app-log-line {
  display: flex;
  gap: 0.5rem;
  padding: 1px 0;
  white-space: pre-wrap;
  word-break: break-all;
}
.log-ts {
  color: var(--sc-text-muted);
  min-width: 72px;
  flex-shrink: 0;
}
.log-text         { color: var(--sc-text, #e2ecff); }
.log-text.error   { color: var(--sc-red, #f04040); }
.log-text.success { color: var(--sc-green, #22d67c); }
.log-text.warn    { color: var(--sc-amber, #f5a623); }
.log-cursor-line  { font-style: italic; color: var(--sc-text-muted); }

/* ── Pulsing dot ── */
@keyframes dot-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: .4; }
}
.status-dot.pulsing { animation: dot-pulse 1.2s infinite; }

/* ── Card header padding ── */
.sc-view-apps :deep(.card-header) { padding: 0.85rem 1rem; }
</style>
