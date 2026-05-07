<template>
  <div>
    <PageHeader title="Audit Logs" icon="mdi mdi-history" :items="[{text:'Audit Logs',active:true,icon:'mdi mdi-file-document-check'}]">
      <template #actions>
        <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff" @click="exportLogs">
          <i class="mdi mdi-download me-1"></i> Export
        </button>
      </template>
    </PageHeader>

    <!-- IP Stats Summary -->
    <div class="row g-3 mb-4">
      <div class="col-md-3">
        <div class="card">
          <div class="card-body py-2 text-center">
            <div class="text-muted" style="font-size:0.7rem">Most Frequent IP</div>
            <div class="fw-bold" style="font-size:0.9rem;color:#4a9eff;cursor:pointer" @click="showIpModal(topIp.ip)">{{ topIp.ip || '—' }}</div>
            <div class="text-muted" style="font-size:0.7rem">{{ topIp.count }} attempts</div>
          </div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card">
          <div class="card-body py-2 text-center">
            <div class="text-muted" style="font-size:0.7rem">Successful IPs</div>
            <div class="fw-bold" style="font-size:0.9rem;color:#22c55e">{{ successIps.length }}</div>
            <div class="text-muted" style="font-size:0.7rem">unique IPs</div>
          </div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card">
          <div class="card-body py-2 text-center">
            <div class="text-muted" style="font-size:0.7rem">Failed IPs</div>
            <div class="fw-bold" style="font-size:0.9rem;color:#f87171">{{ failedIps.length }}</div>
            <div class="text-muted" style="font-size:0.7rem">unique IPs</div>
          </div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="card">
          <div class="card-body py-2 text-center">
            <div class="text-muted" style="font-size:0.7rem">Brute-force Detected</div>
            <div class="fw-bold" style="font-size:0.9rem;color:#f59e0b">{{ bruteForceIps.length }}</div>
            <div class="text-muted" style="font-size:0.7rem">IPs flagged</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Filters -->
    <div class="card mb-4">
      <div class="card-body py-2">
        <div class="row g-2 align-items-center">
          <div class="col-md-4">
            <input v-model="search" class="form-control form-control-sm" placeholder="Search user, reason, IP, user agent…" />
          </div>
          <div class="col-md-3">
            <select v-model="filterUser" class="form-select form-select-sm">
              <option value="">All users</option>
              <option v-for="u in uniqueUsers" :key="u" :value="u">{{ u }}</option>
            </select>
          </div>
          <div class="col-md-3">
            <select v-model="filterResult" class="form-select form-select-sm">
              <option value="">All results</option>
              <option value="success">Success</option>
              <option value="failure">Failure</option>
            </select>
          </div>
          <div class="col-md-2 text-end">
            <span style="font-size:0.78rem;color:#5a7499">{{ filteredLogs.length }} records</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Table -->
    <div class="card">
      <div class="card-body p-0">
        <div v-if="loading" class="text-center py-4" style="color:#5a7499">
          <i class="mdi mdi-loading mdi-spin me-2"></i>Loading audit logs…
        </div>
        <div v-else-if="!loading && logs.length === 0" class="text-center py-4" style="color:#5a7499">
          No audit log entries found
        </div>
        <table v-else class="table mb-0">
          <thead>
            <tr>
              <th>Time</th><th>User</th><th>IP</th><th>Reason</th><th>User Agent</th><th>Result</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="filteredLogs.length === 0">
              <td colspan="6" class="text-center py-3" style="color:#5a7499">No records matching filter</td>
            </tr>
            <tr v-for="log in filteredLogs" :key="log.id">
              <td class="font-mono" style="font-size:0.72rem;color:#5a7499;white-space:nowrap">{{ formatTs(log.ts) }}</td>
              <td>
                <span class="badge" style="background:rgba(74,158,255,0.12);color:#4a9eff;font-size:0.65rem">{{ log.username }}</span>
              </td>
              <td class="font-mono" style="font-size:0.72rem;color:#4a9eff;cursor:pointer;text-decoration:underline" @click="showIpModal(log.ip)">{{ log.ip }}</td>
              <td style="font-size:0.72rem;color:#8aa4c8;max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" :title="log.reason">{{ log.reason || '—' }}</td>
              <td style="font-size:0.72rem;color:#8aa4c8;max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" :title="log.user_agent">{{ log.user_agent || '—' }}</td>
              <td>
                <span class="badge rounded-pill" :class="log.success ? 'badge-online' : 'badge-offline'">{{ log.success ? 'success' : 'failed' }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

    <!-- IP Details Modal -->
    <div v-if="ipModal.show" class="modal fade show d-block" tabindex="-1" style="background-color:rgba(0,0,0,0.5)">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">IP Details: {{ ipModal.ip }}</h5>
            <button type="button" class="btn-close btn-close-white" @click="ipModal.show = false"></button>
          </div>
          <div class="modal-body">
            <div v-if="ipModal.loading" class="text-center py-3">
              <i class="mdi mdi-loading mdi-spin me-2"></i>Loading IP info…
            </div>
            <div v-else-if="ipModal.error" class="text-center py-3 text-danger">
              {{ ipModal.error }}
            </div>
            <div v-else>
              <div class="mb-3">
                <strong>Country:</strong> {{ ipModal.data.country || '—' }} ({{ ipModal.data.countryCode || '—' }})
              </div>
              <div class="mb-3">
                <strong>Region:</strong> {{ ipModal.data.region || '—' }} ({{ ipModal.data.regionCode || '—' }})
              </div>
              <div class="mb-3">
                <strong>City:</strong> {{ ipModal.data.city || '—' }}
              </div>
              <div class="mb-3">
                <strong>ISP:</strong> {{ ipModal.data.org || ipModal.data.isp || '—' }}
              </div>
              <div class="mb-3">
                <strong>Location:</strong> {{ ipModal.data.loc ? `${ipModal.data.lat}, ${ipModal.data.lon}` : '—' }}
              </div>
              <div class="mb-3">
                <strong>Timezone:</strong> {{ ipModal.data.timezone || '—' }}
              </div>
              <div class="mb-3">
                <strong>ASN:</strong> {{ ipModal.data.asn || '—' }}
              </div>
              <div v-if="ipModal.data.isProxy || ipModal.data.isTor" class="alert alert-warning py-2">
                <i class="mdi mdi-alert me-1"></i>
                <span v-if="ipModal.data.isProxy">Proxy detected</span>
                <span v-if="ipModal.data.isTor">Tor exit node detected</span>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="ipModal.show = false">Close</button>
          </div>
        </div>
      </div>
    </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'

export default {
  name: 'AuditLogsPage',
  components: { PageHeader },
  data() {
    return {
      search: '',
      filterUser: '',
      filterResult: '',
      loading: false,
      logs: [],
      ipModal: { show: false, ip: '', loading: false, data: {}, error: '' }
    }
  },

  computed: {
    uniqueUsers() {
      return [...new Set(this.logs.map(l => l.username).filter(Boolean))].sort()
    },
    filteredLogs() {
      let list = this.logs
      if (this.filterUser) list = list.filter(l => l.username === this.filterUser)
      if (this.filterResult) {
        const isSuccess = this.filterResult === 'success'
        list = list.filter(l => l.success === isSuccess)
      }
      if (this.search) {
        const s = this.search.toLowerCase()
        list = list.filter(l =>
          (l.username   || '').toLowerCase().includes(s) ||
          (l.ip         || '').includes(s) ||
          (l.reason     || '').toLowerCase().includes(s) ||
          (l.user_agent || '').toLowerCase().includes(s)
        )
      }
      return list
    },
    ipCounts() {
      const counts = {}
      this.logs.forEach(l => {
        if (l.ip) {
          counts[l.ip] = (counts[l.ip] || 0) + 1
        }
      })
      return counts
    },
    topIp() {
      const entries = Object.entries(this.ipCounts)
      if (!entries.length) return { ip: null, count: 0 }
      const [ip, count] = entries.reduce((a, b) => (b[1] > a[1] ? b : a))
      return { ip, count }
    },
    successIps() {
      const set = new Set()
      this.logs.filter(l => l.success && l.ip).forEach(l => set.add(l.ip))
      return Array.from(set)
    },
    failedIps() {
      const set = new Set()
      this.logs.filter(l => !l.success && l.ip).forEach(l => set.add(l.ip))
      return Array.from(set)
    },
    bruteForceIps() {
      const set = new Set()
      this.logs.filter(l => l.reason === 'rate_limited' && l.ip).forEach(l => set.add(l.ip))
      return Array.from(set)
    }
  },

  mounted() {
    this.loadAuditLogs()
  },

  methods: {
    async showIpModal(ip) {
      if (!ip) return
      this.ipModal = { show: true, ip, loading: true, data: {}, error: '' }
      try {
        const token = import.meta.env.VITE_IPINFO_TOKEN || ''
        const url = token ? `https://ipinfo.io/${ip}/json?token=${token}` : `https://ipinfo.io/${ip}/json`
        const resp = await fetch(url)
        if (!resp.ok) throw new Error('Failed to fetch IP info')
        const data = await resp.json()
        this.ipModal.data = data
      } catch (e) {
        this.ipModal.error = 'Unable to fetch IP details. Service may be rate-limited.'
      } finally {
        this.ipModal.loading = false
      }
    },
    async loadAuditLogs() {
      const api = (await import('@/services/api')).default
      this.loading = true
      try {
        const { data } = await api.getAuditLogs({ limit: 200 })
        this.logs = Array.isArray(data) ? data : (data.logs || [])
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Failed to load audit logs', text: e.response?.data?.detail || e.message })
      } finally {
        this.loading = false
      }
    },
    formatTs(ts) {
      if (!ts) return '—'
      return new Date(ts * 1000).toISOString().replace('T', ' ').slice(0, 19)
    },
    exportLogs() {
      const rows = this.filteredLogs.map(l =>
        [this.formatTs(l.ts), l.username, l.ip, l.reason, l.user_agent, l.success ? 'success' : 'failed'].join('\t')
      )
      const blob = new Blob(
        [['Time', 'User', 'IP', 'Reason', 'User Agent', 'Result'].join('\t') + '\n' + rows.join('\n')],
        { type: 'text/tsv' }
      )
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `audit-${Date.now()}.tsv`
      a.click()
    }
  }
}
</script>
