<template>
  <div>
    <PageHeader title="Alerts" icon="mdi mdi-bell" :items="[{text:'Alerts',active:true,icon:'mdi mdi-bell-alert'}]">
      <template #actions>
        <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff" @click="loadAlerts">
          <i class="mdi mdi-refresh me-1"></i> Refresh
        </button>
        <button class="btn btn-sm ms-2" style="background:rgba(74,158,255,0.12);color:#4a9eff" @click="markAllRead">
          <i class="mdi mdi-check-all me-1"></i> Mark All Read
        </button>
      </template>
    </PageHeader>

    <!-- Severity tabs -->
    <div class="d-flex gap-2 mb-4">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="btn btn-sm"
        :style="`background:rgba(${tab.rgb},${activeTab===tab.key?'0.2':'0.08'});color:rgba(${tab.rgb},1);border:1px solid rgba(${tab.rgb},${activeTab===tab.key?'0.4':'0.15'})`"
        @click="activeTab = tab.key"
      >
        <i :class="tab.icon" class="me-1"></i>
        {{ tab.label }}
        <span class="ms-1 badge rounded-pill" :style="`background:rgba(${tab.rgb},0.2);color:rgba(${tab.rgb},1);font-size:0.62rem`">{{ countBySeverity(tab.key) }}</span>
      </button>
    </div>

    <!-- Stats row -->
    <div class="row g-3 mb-4">
      <div class="col-xl-3 col-md-6">
        <StatCard label="Emergency" :value="countBySeverity('emergency')" icon="mdi mdi-alert-octagon" icon-color="#f04040" icon-bg="rgba(240,64,64,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Critical" :value="countBySeverity('critical')" icon="mdi mdi-alert-circle" icon-color="#f5a623" icon-bg="rgba(245,166,35,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Warning" :value="countBySeverity('warning')" icon="mdi mdi-alert" icon-color="#f5a623" icon-bg="rgba(245,166,35,0.08)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Info" :value="countBySeverity('info')" icon="mdi mdi-information" icon-color="#4a9eff" icon-bg="rgba(74,158,255,0.12)" />
      </div>
    </div>

    <!-- Alert list -->
    <div class="card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-bell-alert me-2" style="color:#f5a623"></i>{{ activeTab === 'all' ? 'All Alerts' : activeTab + ' alerts' }}</h6>
        <div class="d-flex gap-2">
          <select v-model="readFilter" class="form-select form-select-sm" style="width:130px">
            <option value="all">All states</option>
            <option value="unread">Unread</option>
            <option value="read">Read</option>
          </select>
          <select v-model="sourceFilter" class="form-select form-select-sm" style="width:180px">
            <option value="">All services</option>
            <option v-for="s in sourceOptions" :key="s" :value="s">{{ s }}</option>
          </select>
          <input v-model="searchText" class="form-control form-control-sm" placeholder="Search message/type/source/IP/user…" style="width:280px" />
        </div>
      </div>
      <div class="card-body p-0">
        <div v-if="loading" class="text-center py-4" style="color:#5a7499">
          <i class="mdi mdi-loading mdi-spin me-2"></i>Loading alerts…
        </div>
        <div v-else-if="filteredAlerts.length === 0" class="text-center py-4" style="color:#5a7499">
          No alerts matching criteria
        </div>
        <div
          v-for="alert in paginatedAlerts"
          :key="alert.id"
          class="border-bottom"
          :class="{ 'opacity-50': alert.read }"
          style="border-color:rgba(30,45,74,0.5)!important;transition:opacity 0.2s"
        >
          <div class="d-flex align-items-start gap-3 p-3">
            <div class="flex-shrink-0 mt-1" :style="`width:36px;height:36px;border-radius:8px;background:rgba(${severityRgb(alert.severity)},0.12);display:flex;align-items:center;justify-content:center`">
              <i :class="severityIcon(alert.severity)" :style="`color:rgba(${severityRgb(alert.severity)},1);font-size:1rem`"></i>
            </div>
            <div class="flex-fill">
              <div class="d-flex align-items-center gap-2 mb-1">
                <span :style="`font-size:0.65rem;font-weight:700;text-transform:uppercase;letter-spacing:.06em;padding:2px 6px;border-radius:3px;background:rgba(${severityRgb(alert.severity)},0.12);color:rgba(${severityRgb(alert.severity)},1)`">{{ alert.severity }}</span>
                <span style="font-size:0.8rem;font-weight:600;color:#c9d8f0">{{ alert.type || alert.source }}</span>
                <span v-if="!alert.read" class="ms-auto badge" style="background:rgba(74,158,255,0.15);color:#4a9eff;font-size:0.6rem">NEW</span>
              </div>
              <div style="font-size:0.78rem;color:#8aa4c8;margin-bottom:4px">{{ alert.message }}</div>
              <div class="d-flex align-items-center gap-3">
                <span style="font-size:0.7rem;color:#5a7499"><i class="mdi mdi-tag-outline me-1"></i>{{ alert.source }}</span>
                <span style="font-size:0.7rem;color:#5a7499"><i class="mdi mdi-clock-outline me-1"></i>{{ timeAgo(alert.ts) }}</span>
                <button class="btn btn-sm ms-auto" style="background:transparent;color:#4a9eff;font-size:0.72rem;padding:0 4px" @click="markRead(alert.id)">Dismiss</button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="filteredAlerts.length > 0" class="d-flex align-items-center justify-content-between px-3 py-2" style="border-top:1px solid var(--sc-border)">
          <div class="d-flex align-items-center gap-2">
            <span style="font-size:0.72rem;color:var(--sc-text-muted)">{{ filteredAlerts.length }} records</span>
            <select v-model.number="pageSize" class="form-select form-select-sm" style="width:120px">
              <option :value="10">10 / page</option>
              <option :value="25">25 / page</option>
              <option :value="50">50 / page</option>
              <option :value="100">100 / page</option>
            </select>
          </div>
          <div class="d-flex align-items-center gap-2">
            <span style="font-size:0.72rem;color:var(--sc-text-muted)">Page {{ currentPage }} / {{ totalPages }}</span>
            <button class="btn btn-sm" style="background:rgba(74,158,255,0.08);color:#4a9eff;padding:2px 8px;font-size:0.72rem" :disabled="currentPage===1" @click="currentPage--">
              <i class="mdi mdi-chevron-left"></i>
            </button>
            <button class="btn btn-sm" style="background:rgba(74,158,255,0.08);color:#4a9eff;padding:2px 8px;font-size:0.72rem" :disabled="currentPage===totalPages" @click="currentPage++">
              <i class="mdi mdi-chevron-right"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard   from '@/components/widgets/stat-card.vue'

export default {
  name: 'AlertsPage',
  components: { PageHeader, StatCard },
  data() {
    return {
      activeTab: 'all',
      readFilter: 'all',
      sourceFilter: '',
      searchText: '',
      loading: false,
      pageSize: 25,
      currentPage: 1,
      tabs: [
        { key: 'all',       label: 'All',       icon: 'mdi mdi-bell-outline',       rgb: '138,164,200' },
        { key: 'emergency', label: 'Emergency', icon: 'mdi mdi-alert-octagon',      rgb: '240,64,64'   },
        { key: 'critical',  label: 'Critical',  icon: 'mdi mdi-alert-circle',       rgb: '240,64,64'   },
        { key: 'warning',   label: 'Warning',   icon: 'mdi mdi-alert',              rgb: '245,166,35'  },
        { key: 'info',      label: 'Info',      icon: 'mdi mdi-information-outline', rgb: '74,158,255' }
      ],
      alerts: []
    }
  },

  computed: {
    sourceOptions() {
      return [...new Set(this.alerts.map(a => a.source).filter(Boolean))].sort()
    },
    filteredAlerts() {
      let list = [...this.alerts]
      if (this.activeTab !== 'all') list = list.filter(a => a.severity === this.activeTab)
      if (this.readFilter === 'read') list = list.filter(a => a.read)
      if (this.readFilter === 'unread') list = list.filter(a => !a.read)
      if (this.sourceFilter) list = list.filter(a => a.source === this.sourceFilter)
      if (this.searchText) {
        const s = this.searchText.toLowerCase()
        list = list.filter(a =>
          (a.type    || '').toLowerCase().includes(s) ||
          (a.message || '').toLowerCase().includes(s) ||
          (a.source  || '').toLowerCase().includes(s) ||
          (a.ip      || '').toLowerCase().includes(s) ||
          (a.username || '').toLowerCase().includes(s)
        )
      }
      list.sort((a, b) => (b.ts || 0) - (a.ts || 0))
      return list
    },
    totalPages() {
      return Math.max(1, Math.ceil(this.filteredAlerts.length / this.pageSize))
    },
    paginatedAlerts() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.filteredAlerts.slice(start, start + this.pageSize)
    }
  },

  watch: {
    activeTab() { this.currentPage = 1 },
    readFilter() { this.currentPage = 1 },
    sourceFilter() { this.currentPage = 1 },
    searchText() { this.currentPage = 1 },
    pageSize() { this.currentPage = 1 },
    filteredAlerts() {
      if (this.currentPage > this.totalPages) this.currentPage = this.totalPages
    }
  },

  mounted() {
    this.loadAlerts()
  },

  methods: {
    async loadAlerts() {
      const api = (await import('@/services/api')).default
      this.loading = true
      try {
        const { data } = await api.getAlerts()
        this.alerts = Array.isArray(data) ? data : (data.alerts || [])
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Failed to load alerts', text: e.response?.data?.detail || e.message })
      } finally {
        this.loading = false
      }
    },
    async markRead(id) {
      const api = (await import('@/services/api')).default
      try {
        await api.markAlertRead(id)
        const alert = this.alerts.find(a => a.id === id)
        if (alert) alert.read = true
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Failed to dismiss alert', text: e.response?.data?.detail || e.message })
      }
    },
    async markAllRead() {
      const api = (await import('@/services/api')).default
      const unread = this.alerts.filter(a => !a.read)
      await Promise.all(unread.map(a => api.markAlertRead(a.id).catch(() => {})))
      this.alerts.forEach(a => { a.read = true })
    },
    countBySeverity(s) {
      return s === 'all' ? this.alerts.length : this.alerts.filter(a => a.severity === s).length
    },
    severityRgb(s) {
      return { emergency: '240,64,64', critical: '240,64,64', warning: '245,166,35', info: '74,158,255' }[s] || '138,164,200'
    },
    severityIcon(s) {
      return { emergency: 'mdi mdi-alert-octagon', critical: 'mdi mdi-alert-circle', warning: 'mdi mdi-alert', info: 'mdi mdi-information' }[s] || 'mdi mdi-bell'
    },
    timeAgo(ts) {
      if (!ts) return ''
      const diff = Math.floor(Date.now() / 1000) - ts
      if (diff < 60)    return `${diff}s ago`
      if (diff < 3600)  return `${Math.floor(diff / 60)}m ago`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
      return `${Math.floor(diff / 86400)}d ago`
    }
  }
}
</script>
