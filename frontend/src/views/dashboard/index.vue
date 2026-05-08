<template>
  <div ref="pageEl">
    <!-- Pull-to-refresh indicator (mobile) -->
    <div class="ptr-bar" :class="{ pulling: isPulling, refreshing: isRefreshing }"
         :style="{ height: `${Math.min(pullDist, 64)}px`, opacity: Math.min(pullDist / 64, 1) }">
      <i class="mdi" :class="isRefreshing ? 'mdi-loading mdi-spin' : pullDist >= 64 ? 'mdi-refresh' : 'mdi-arrow-down'"></i>
    </div>

    <PageHeader title="Dashboard" icon="mdi mdi-view-dashboard" :items="breadcrumbs">
      <template #actions>
        <span v-if="isRefreshing" class="spinner-border spinner-border-sm text-info me-2" role="status"></span>
        <div class="btn-group btn-group-sm me-2">
          <button class="btn" :class="layoutMode === 'flexible' ? 'btn-sc-primary' : 'btn-outline-secondary'" @click="setLayoutMode('flexible')" title="Flexible layout">
            <i class="mdi mdi-application"></i>
          </button>
          <button class="btn" :class="layoutMode === 'compact' ? 'btn-sc-primary' : 'btn-outline-secondary'" @click="setLayoutMode('compact')" title="Compact layout">
            <i class="mdi mdi-application-outline"></i>
          </button>
          <button class="btn btn-sc-primary" @click="refreshAll" title="Refresh all widgets">
            <i class="mdi mdi-refresh"></i>
          </button>
        </div>
      </template>
    </PageHeader>

    <!-- ── Stat row ─────────────────────────────────────────────────────── -->
    <draggable
      v-model="statWidgets"
      class="row g-3 mb-4"
      item-key="id"
      handle=".drag-handle"
      :animation="200"
      ghost-class="drag-ghost"
      chosen-class="drag-chosen"
      @end="saveStatWidgetOrder"
    >
      <template #item="{ element }">
        <div :class="element.colClass">
          <div class="widget-wrapper">
            <div class="drag-handle" title="Drag to rearrange">
              <i class="mdi mdi-drag-vertical"></i>
            </div>
            <StatCard
              v-if="element.id === 'cpu'"
              label="CPU Usage"
              :value="`${cpu}%`"
              :sub="`Load avg: ${loadAvg}`"
              icon="mdi mdi-chip"
              icon-color="#4a9eff"
              icon-bg="rgba(74,158,255,0.12)"
              :progress="cpu"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <StatCard
              v-else-if="element.id === 'memory'"
              label="Memory"
              :value="`${ram}%`"
              :sub="`${ramUsed} / ${ramTotal} used`"
              icon="mdi mdi-memory"
              icon-color="#a78bfa"
              icon-bg="rgba(167,139,250,0.12)"
              :progress="ram"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <StatCard
              v-else-if="element.id === 'disk'"
              label="Disk / (root)"
              :value="`${disk}%`"
              :sub="`${diskUsed} / ${diskTotal}`"
              icon="mdi mdi-harddisk"
              icon-color="#f5a623"
              icon-bg="rgba(245,166,35,0.12)"
              :progress="disk"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <StatCard
              v-else-if="element.id === 'network'"
              label="Network Out"
              :value="netOut"
              :sub="`In: ${netIn}`"
              icon="mdi mdi-swap-vertical"
              icon-color="#22d67c"
              icon-bg="rgba(34,214,124,0.12)"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <div
              v-else-if="element.id === 'cleanup'"
              class="card"
              style="cursor:pointer"
              @click="openCleaner"
            >
              <div class="card-body d-flex align-items-center justify-content-between py-3">
                <div>
                  <div class="text-uppercase" style="font-size:0.68rem;color:#5a7499;letter-spacing:.05em">Junk Cleanable</div>
                  <div style="font-size:1.35rem;font-weight:700;color:#22d67c">{{ cleanupStats.estimated_junk_human || '—' }}</div>
                  <div style="font-size:0.72rem;color:#8aa4c8">Tap to run cleaner animation</div>
                </div>
                <div style="width:46px;height:46px;border-radius:14px;background:rgba(34,214,124,0.15);display:flex;align-items:center;justify-content:center;color:#22d67c">
                  <i class="mdi mdi-broom" style="font-size:1.3rem"></i>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </draggable>

    <div v-if="showCleaner" class="cleaner-modal" @click.self="closeCleaner">
      <div class="cleaner-card">
        <h5 class="mb-2"><i class="mdi mdi-broom me-1"></i>System Cleaner</h5>
        <p class="mb-3" style="font-size:0.82rem;color:#8aa4c8">Reclaiming junk files and optimizing cache...</p>
        <div class="cleaner-bubble">
          <div class="cleaner-fill" :style="{ height: cleanerProgress + '%' }"></div>
          <div class="cleaner-value">{{ cleanerProgress }}%</div>
        </div>
        <div class="mt-3" style="font-size:0.8rem;color:#5a7499">Freed: {{ cleanupStats.last_freed_human || cleanupStats.estimated_junk_human || '0 B' }}</div>
        <div class="d-flex gap-2 mt-3">
          <button class="btn btn-sm btn-sc-primary" :disabled="cleanerRunning" @click="runCleaner">
            <i :class="cleanerRunning ? 'mdi mdi-loading mdi-spin me-1' : 'mdi mdi-play me-1'"></i>{{ cleanerRunning ? 'Cleaning…' : 'Start Cleaner' }}
          </button>
          <button class="btn btn-sm btn-outline-secondary" @click="closeCleaner">Close</button>
        </div>
      </div>
    </div>

    <!-- ── Security summary row ──────────────────────────────────────────── -->
    <draggable
      v-model="securityWidgets"
      class="row g-3 mb-4"
      item-key="id"
      handle=".drag-handle"
      :animation="200"
      ghost-class="drag-ghost"
      chosen-class="drag-chosen"
      @end="saveSecurityWidgetOrder"
    >
      <template #item="{ element }">
        <div :class="element.colClass">
          <div class="widget-wrapper">
            <div class="drag-handle" title="Drag to rearrange">
              <i class="mdi mdi-drag-vertical"></i>
            </div>
            <StatCard
              v-if="element.id === 'bans'"
              label="Active Bans"
              :value="secStats.activeBans"
              sub="fail2ban + CrowdSec"
              icon="mdi mdi-shield-lock"
              icon-color="#f04040"
              icon-bg="rgba(240,64,64,0.12)"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <StatCard
              v-else-if="element.id === 'logins24h'"
              label="Failed Logins (24h)"
              :value="secStats.failedLogins"
              sub="all sources"
              icon="mdi mdi-lock-alert"
              icon-color="#f5a623"
              icon-bg="rgba(245,166,35,0.12)"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <StatCard
              v-else-if="element.id === 'docker'"
              label="Docker Containers"
              :value="`${dockerInfo.containers_running}/${dockerInfo.containers_total}`"
              sub="running / total"
              icon="mdi mdi-docker"
              icon-color="#22d3ee"
              icon-bg="rgba(34,211,238,0.12)"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
            <StatCard
              v-else-if="element.id === 'uptime'"
              label="Uptime"
              :value="uptime"
              sub="since last reboot"
              icon="mdi mdi-clock-outline"
              icon-color="#22d67c"
              icon-bg="rgba(34,214,124,0.12)"
              @click="navigateTo(element.link)"
              style="cursor:pointer"
            />
          </div>
        </div>
      </template>
    </draggable>

    <!-- ── Charts row ─────────────────────────────────────────────────────── -->
    <div class="row g-3 mb-4">
      <div class="col-xl-6">
        <div class="widget-wrapper">
          <div class="drag-handle" title="Drag to rearrange">
            <i class="mdi mdi-drag-vertical"></i>
          </div>
          <div class="card">
            <div class="card-header d-flex align-items-center justify-content-between">
              <h6><i class="mdi mdi-chip me-2" style="color:#4a9eff"></i>CPU Usage (last 60s)</h6>
              <span class="badge" style="background:rgba(74,158,255,0.15);color:#4a9eff;font-size:0.7rem">LIVE</span>
            </div>
            <div class="card-body py-2">
              <apexchart type="area" height="180" :options="cpuChart.options" :series="cpuChart.series" />
            </div>
          </div>
        </div>
      </div>
      <div class="col-xl-6">
        <div class="widget-wrapper">
          <div class="drag-handle" title="Drag to rearrange">
            <i class="mdi mdi-drag-vertical"></i>
          </div>
          <div class="card">
            <div class="card-header d-flex align-items-center justify-content-between">
              <h6><i class="mdi mdi-swap-vertical me-2" style="color:#22d67c"></i>Network Traffic (last 60s)</h6>
              <span class="badge" style="background:rgba(34,214,124,0.15);color:#22d67c;font-size:0.7rem">LIVE</span>
            </div>
            <div class="card-body py-2">
              <apexchart type="area" height="180" :options="netChart.options" :series="netChart.series" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Draggable widget row ──────────────────────────────────────────── -->
    <draggable
      v-model="widgets"
      class="row g-3"
      item-key="id"
      handle=".drag-handle"
      :animation="200"
      ghost-class="drag-ghost"
      chosen-class="drag-chosen"
      @end="saveWidgetOrder"
    >
      <template #item="{ element }">
        <div :class="widgetColClass">
          <div class="widget-wrapper h-100">
            <div class="drag-handle" title="Drag to rearrange">
              <i class="mdi mdi-drag-vertical"></i>
            </div>
            <HealthCheck   v-if="element.id === 'health'" class="h-100" />
            <AlertFeed    v-else-if="element.id === 'alerts'" class="h-100" :compact="layoutMode === 'compact'" />
            <ServiceStatus v-else-if="element.id === 'services'" class="h-100" :compact="layoutMode === 'compact'" />
            <div v-else-if="element.id === 'logins'" class="card h-100">
              <div class="card-header d-flex align-items-center justify-content-between">
                <h6><i class="mdi mdi-login me-2" style="color:#a78bfa"></i>Recent Login Attempts</h6>
              </div>
              <div class="card-body" :style="layoutMode === 'compact' ? 'max-height:200px;overflow-y:auto;padding:0.75rem' : 'max-height:300px;overflow-y:auto;padding:0.75rem'">
                <div style="overflow-x:auto">
                  <table class="table mb-0" style="min-width:100%">
                    <thead><tr><th style="font-size:0.72rem;padding:0.5rem 0.75rem">IP</th><th style="font-size:0.72rem;padding:0.5rem 0.75rem">User</th><th style="font-size:0.72rem;padding:0.5rem 0.75rem">Status</th><th style="font-size:0.72rem;padding:0.5rem 0.75rem">Time</th></tr></thead>
                    <tbody>
                      <tr v-if="!loginAttempts.length">
                        <td colspan="4" class="text-center" style="color:#5a7499;font-size:0.8rem;padding:1.5rem">No login attempts recorded</td>
                      </tr>
                      <tr v-for="(a, i) in loginAttempts" :key="i">
                        <td class="font-mono" style="font-size:0.75rem;padding:0.5rem 0.75rem;word-break:break-all">{{ a.ip }}</td>
                        <td style="font-size:0.78rem;padding:0.5rem 0.75rem">{{ a.username }}</td>
                        <td style="padding:0.5rem 0.75rem">
                          <span class="badge rounded-pill" :class="a.success ? 'badge-online' : 'badge-offline'">
                            {{ a.success ? 'OK' : 'FAIL' }}
                          </span>
                        </td>
                        <td style="font-size:0.72rem;color:#5a7499;padding:0.5rem 0.75rem;white-space:nowrap">{{ new Date(a.ts * 1000).toLocaleTimeString() }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </draggable>
  </div>
</template>

<script>
import PageHeader    from '@/components/page-header.vue'
import StatCard      from '@/components/widgets/stat-card.vue'
import AlertFeed     from '@/components/widgets/alert-feed.vue'
import ServiceStatus from '@/components/widgets/service-status.vue'
import HealthCheck   from '@/components/widgets/health-check.vue'
import draggable     from 'vuedraggable'
import { mapGetters } from 'vuex'

function sanitizeSeries(data = []) {
  return data.map(value => {
    const num = Number(value)
    return Number.isFinite(num) ? num : null
  })
}

function assertFiniteSeries(series, name) {
  if (!import.meta.env.DEV) return
  series.forEach((value, index) => {
    if (value !== null && !Number.isFinite(value)) {
      console.error('Non-finite chart value in series', { name, index, value })
    }
  })
}

const DEFAULT_WIDGETS = [
  { id: 'health' },
  { id: 'alerts' },
  { id: 'services' },
  { id: 'logins' }
]

function fmtBytes (b) {
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  if (b >= 1048576)    return (b / 1048576).toFixed(1) + ' MB'
  if (b >= 1024)       return (b / 1024).toFixed(0) + ' KB'
  return b + ' B'
}

function fmtUptime (seconds) {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

const CHART_OPTS_BASE = (color) => ({
  chart:  { type: 'area', toolbar: { show: false }, animations: { enabled: true, easing: 'easeinout', speed: 400 }, background: 'transparent' },
  theme:  { mode: 'dark' },
  colors: [color],
  stroke: { curve: 'smooth', width: 2 },
  fill:   { type: 'gradient', gradient: { shadeIntensity: 1, opacityFrom: 0.3, opacityTo: 0.02 } },
  grid:   { borderColor: '#1e2d4a', strokeDashArray: 3 },
  xaxis:  { labels: { show: false }, axisBorder: { show: false }, axisTicks: { show: false } },
  dataLabels: { enabled: false }
})

export default {
  name: 'DashboardPage',
  components: { PageHeader, StatCard, AlertFeed, ServiceStatus, HealthCheck, draggable },

  data() {
    const savedOrder = (() => {
      try { return JSON.parse(localStorage.getItem('sc_widget_order') || 'null') } catch { return null }
    })()
    const savedLayout = (() => {
      try { return localStorage.getItem('sc_layout_mode') || 'flexible' } catch { return 'flexible' }
    })()
    return {
      breadcrumbs:  [{ text: 'Dashboard', active: true, icon: 'mdi mdi-view-dashboard' }],
      dockerInfo:   { containers_running: 0, containers_total: 0 },
      loginAttempts: [],
      secStats:     { activeBans: '—', failedLogins: '—' },
      cleanupStats: { estimated_junk_bytes: 0, estimated_junk_human: '0 B', last_freed_human: '0 B' },
      isRefreshing: false,
      widgets:      savedOrder || DEFAULT_WIDGETS,
      layoutMode:   savedLayout,
      statWidgets: [
        { id: 'cpu', colClass: 'col-xl-3 col-md-6', link: '/monitoring' },
        { id: 'memory', colClass: 'col-xl-3 col-md-6', link: '/monitoring' },
        { id: 'disk', colClass: 'col-xl-3 col-md-6', link: '/monitoring' },
        { id: 'network', colClass: 'col-xl-3 col-md-6', link: '/monitoring' },
        { id: 'cleanup', colClass: 'col-xl-3 col-md-6', link: '' }
      ],
      securityWidgets: [
        { id: 'bans', colClass: 'col-xl-3 col-md-6', link: '/security' },
        { id: 'logins24h', colClass: 'col-xl-3 col-md-6', link: '/audit-logs' },
        { id: 'docker', colClass: 'col-xl-3 col-md-6', link: '/containers' },
        { id: 'uptime', colClass: 'col-xl-3 col-md-6', link: '/monitoring' }
      ],

      // pull-to-refresh
      pullStartY:  0,
      pullDist:    0,
      isPulling:   false,
      showCleaner: false,
      cleanerRunning: false,
      cleanerProgress: 0,
      cleanerTimer: null,
    }
  },

  computed: {
    ...mapGetters('metrics', ['snap', 'cpuHistory', 'ramHistory', 'netRxHistory', 'netTxHistory']),

    widgetColClass() {
      return this.layoutMode === 'compact' ? 'col-xl-4 col-md-4' : 'col-xl-4 col-md-6'
    },

    cpu()       { return this.snap.cpu_pct },
    ram()       { return this.snap.ram_pct },
    disk()      { return this.snap.disk_pct },
    loadAvg()   { return `${this.snap.load1}, ${this.snap.load5}, ${this.snap.load15}` },
    ramUsed()   { return fmtBytes(this.snap.ram_used) },
    ramTotal()  { return fmtBytes(this.snap.ram_total) },
    diskUsed()  { return fmtBytes(this.snap.disk_used) },
    diskTotal() { return fmtBytes(this.snap.disk_total) },
    netOut()    { return fmtBytes(this.snap.net_tx_rate) + '/s' },
    netIn()     { return fmtBytes(this.snap.net_rx_rate) + '/s' },
    uptime()    { return fmtUptime(this.snap.uptime) },

    cpuChart() {
      const series = sanitizeSeries(this.cpuHistory)
      assertFiniteSeries(series, 'cpuHistory')
      return {
        series: [{ name: 'CPU %', data: series }],
        options: {
          ...CHART_OPTS_BASE('#4a9eff'),
          yaxis: { min: 0, max: 100, labels: { style: { colors: '#5a7499', fontSize: '11px' }, formatter: v => `${v}%` } },
          tooltip: { theme: 'dark', y: { formatter: v => `${v}%` } }
        }
      }
    },

    netChart() {
      const rxSeries = sanitizeSeries(this.netRxHistory)
      const txSeries = sanitizeSeries(this.netTxHistory)
      assertFiniteSeries(rxSeries, 'netRxHistory')
      assertFiniteSeries(txSeries, 'netTxHistory')
      return {
        series: [
          { name: 'RX', data: rxSeries },
          { name: 'TX', data: txSeries }
        ],
        options: {
          ...CHART_OPTS_BASE('#22d67c'),
          colors: ['#4a9eff', '#22d67c'],
          fill: { type: 'gradient', gradient: { shadeIntensity: 1, opacityFrom: 0.25, opacityTo: 0.02 } },
          yaxis: { labels: { style: { colors: '#5a7499', fontSize: '11px' }, formatter: v => fmtBytes(v) + '/s' } },
          tooltip: { theme: 'dark', shared: true, y: { formatter: v => fmtBytes(v) + '/s' } },
          legend: { position: 'top', labels: { colors: '#8aa4c8' } }
        }
      }
    }
  },

  async mounted() {
    this.$store.dispatch('metrics/startLive')
    await this.loadDashboardLayoutFromDb()
    await this.loadAll()
    window.dispatchEvent(new CustomEvent('sentinel:dashboard-ready'))
    this.registerPullToRefresh()
  },

  beforeUnmount() {
    this.unregisterPullToRefresh()
    if (this.cleanerTimer) {
      clearInterval(this.cleanerTimer)
      this.cleanerTimer = null
    }
  },

  methods: {
    async loadAll() {
      if (!this.$store.getters['auth/loggedIn']) return
      try {
        const api = (await import('@/services/api')).default
        const [docker, secStatus, logins, cleanup] = await Promise.allSettled([
          api.getDockerInfo(),
          api.getSecurityStatus(),
          api.getDashboardLoginAttempts(),
          api.getCleanupStats()
        ])
        if (docker.status === 'fulfilled') this.dockerInfo = docker.value.data
        if (secStatus.status === 'fulfilled') {
          const s = secStatus.value.data
          this.secStats = { activeBans: s.active_bans ?? '—', failedLogins: s.failed_logins ?? '—' }
        }
        if (logins.status === 'fulfilled') this.loginAttempts = logins.value.data || []
        if (cleanup.status === 'fulfilled') this.cleanupStats = cleanup.value.data || this.cleanupStats
      } catch {}
    },

    async refreshAll() {
      if (this.isRefreshing) return
      this.isRefreshing = true
      try {
        const api = (await import('@/services/api')).default
        const metrics = await api.getMetrics()
        this.$store.commit('metrics/SET_SNAP', metrics.data)
      } catch {}
      await this.loadAll()
      this.isRefreshing = false
    },

    saveWidgetOrder() {
      localStorage.setItem('sc_widget_order', JSON.stringify(this.widgets))
      this.saveWidgetOrderToDb()
    },

    saveStatWidgetOrder() {
      localStorage.setItem('sc_stat_widgets', JSON.stringify(this.statWidgets))
    },

    saveSecurityWidgetOrder() {
      localStorage.setItem('sc_security_widgets', JSON.stringify(this.securityWidgets))
    },

    saveLayoutMode() {
      localStorage.setItem('sc_layout_mode', this.layoutMode)
      this.saveLayoutModeToDb()
    },

    navigateTo(link) {
      if (link) {
        this.$router.push(link)
      }
    },

    async saveWidgetOrderToDb() {
      try {
        const api = (await import('@/services/api')).default
        await api.saveDashboardLayout({ widgets: this.widgets, layoutMode: this.layoutMode })
      } catch (err) {
        console.error('Failed to save widget order to database:', err)
      }
    },

    async saveLayoutModeToDb() {
      try {
        const api = (await import('@/services/api')).default
        await api.saveDashboardLayout({ widgets: this.widgets, layoutMode: this.layoutMode })
      } catch (err) {
        console.error('Failed to save layout mode to database:', err)
      }
    },

    async loadDashboardLayoutFromDb() {
      if (!this.$store.getters['auth/loggedIn']) return
      try {
        const api = (await import('@/services/api')).default
        const res = await api.getDashboardLayout()
        if (res.data && res.data.widgets) {
          // Ensure health widget is always included
          const hasHealth = res.data.widgets.some(w => w.id === 'health')
          if (!hasHealth) {
            res.data.widgets = [{ id: 'health' }, ...res.data.widgets]
          }
          this.widgets = res.data.widgets
          if (res.data.layoutMode) {
            this.layoutMode = res.data.layoutMode
          }
        }
      } catch (err) {
        if (err.response?.status !== 401) {
          console.error('Failed to load dashboard layout from database:', err)
        }
      }
    },

    registerPullToRefresh() {
      const el = this.$refs.pageEl
      if (!el) return
      el.addEventListener('touchstart', this.onTouchStart, { passive: true })
      el.addEventListener('touchmove',  this.onTouchMove,  { passive: true })
      el.addEventListener('touchend',   this.onTouchEnd,   { passive: true })
    },

    unregisterPullToRefresh() {
      const el = this.$refs.pageEl
      if (!el) return
      el.removeEventListener('touchstart', this.onTouchStart)
      el.removeEventListener('touchmove',  this.onTouchMove)
      el.removeEventListener('touchend',   this.onTouchEnd)
    },

    onTouchStart(e) {
      const scrollEl = document.querySelector('.page-content') || window
      const scrollTop = scrollEl === window ? window.scrollY : scrollEl.scrollTop
      if (scrollTop > 0) return
      this.pullStartY = e.touches[0].clientY
      this.isPulling  = true
    },

    onTouchMove(e) {
      if (!this.isPulling) return
      const dy = e.touches[0].clientY - this.pullStartY
      this.pullDist = dy > 0 ? dy : 0
    },

    async onTouchEnd() {
      if (!this.isPulling) return
      this.isPulling = false
      if (this.pullDist >= 64) {
        this.isRefreshing = true
        this.pullDist = 0
        await this.refreshAll()
        this.isRefreshing = false
      } else {
        this.pullDist = 0
      }
    },

    setLayoutMode(mode) {
      this.layoutMode = mode
      this.saveLayoutMode()
    },
    openCleaner() {
      this.showCleaner = true
      this.cleanerProgress = 0
    },
    closeCleaner() {
      this.showCleaner = false
      this.cleanerRunning = false
      if (this.cleanerTimer) {
        clearInterval(this.cleanerTimer)
        this.cleanerTimer = null
      }
    },
    async runCleaner() {
      if (this.cleanerRunning) return
      this.cleanerRunning = true
      this.cleanerProgress = 0
      const api = (await import('@/services/api')).default
      await api.runCleanup().catch(() => {})

      if (this.cleanerTimer) clearInterval(this.cleanerTimer)
      this.cleanerTimer = setInterval(async () => {
        this.cleanerProgress = Math.min(98, this.cleanerProgress + Math.floor(Math.random() * 8 + 3))
        const { data } = await api.getCleanupLogs().catch(() => ({ data: null }))
        if (data?.done) {
          this.cleanupStats = { ...this.cleanupStats, ...data }
          this.cleanerProgress = 100
          this.cleanerRunning = false
          clearInterval(this.cleanerTimer)
          this.cleanerTimer = null
          await this.loadAll()
        }
      }, 700)
    }
  }
}
</script>

<style scoped>
/* Pull-to-refresh bar */
.ptr-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: rgba(74, 158, 255, 0.08);
  border-bottom: 1px solid rgba(74, 158, 255, 0.2);
  color: #4a9eff;
  font-size: 1.1rem;
  transition: height 0.2s ease, opacity 0.2s ease;
  height: 0;
  opacity: 0;
}

/* Draggable */
.widget-wrapper {
  position: relative;
}

.drag-handle {
  position: absolute;
  top: 8px;
  right: 12px;
  z-index: 10;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #5a7499;
  cursor: grab;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
  padding: 4px;
}

.drag-handle:hover {
  color: #8aa4c8;
  background: rgba(255,255,255,0.06);
}

.drag-handle:active {
  cursor: grabbing;
}

.drag-ghost {
  opacity: 0.4;
}

.drag-chosen {
  box-shadow: 0 8px 32px rgba(74,158,255,0.18);
}

/* Add padding to cards so content doesn't overlap drag handle */
.widget-wrapper .card {
  padding-right: 40px;
}

.cleaner-modal {
  position: fixed;
  inset: 0;
  z-index: 1060;
  display: grid;
  place-items: center;
  background: rgba(8, 14, 24, 0.72);
  backdrop-filter: blur(8px);
}

.cleaner-card {
  width: min(92vw, 420px);
  border-radius: 16px;
  border: 1px solid rgba(34, 214, 124, 0.35);
  background: rgba(15, 22, 41, 0.92);
  padding: 1.2rem;
}

.cleaner-bubble {
  width: 170px;
  height: 170px;
  border-radius: 50%;
  margin: 0 auto;
  position: relative;
  overflow: hidden;
  border: 2px solid rgba(34, 214, 124, 0.4);
  box-shadow: inset 0 0 18px rgba(34, 214, 124, 0.25);
}

.cleaner-fill {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(180deg, rgba(34,214,124,0.9), rgba(34,214,124,0.45));
  transition: height 0.6s cubic-bezier(.22,.61,.36,1);
}

.cleaner-value {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d9fce9;
  font-size: 1.9rem;
  font-weight: 700;
}
</style>
