<template>
  <div class="sc-view sc-view-security">
    <PageHeader title="Security Center" icon="mdi mdi-shield" :items="[{text:'Security Center',active:true,icon:'mdi mdi-shield-check'}]">
      <template #actions>
        <span v-if="loading" class="spinner-border spinner-border-sm text-info me-2" role="status"></span>
        <button class="btn btn-sm btn-sc-primary" @click="loadData">
          <i class="mdi mdi-refresh me-1"></i> Refresh
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="alert alert-danger mb-4">{{ error }}</div>

    <!-- Security score + stat cards -->
    <div class="row g-3 mb-4">
      <!-- Score ring -->
      <div class="col-xl-3 col-md-6">
        <div class="card sc-panel-card text-center py-3">
          <div class="security-score">
            <div class="score-ring">
              <svg width="120" height="120" viewBox="0 0 120 120">
                <circle cx="60" cy="60" r="50" fill="none" stroke="var(--sc-border)" stroke-width="10"/>
                <circle cx="60" cy="60" r="50" fill="none"
                  :stroke="scoreColor"
                  stroke-width="10"
                  stroke-linecap="round"
                  :stroke-dasharray="`${securityScore * 3.14} 314`"
                />
              </svg>
              <div class="score-text">
                <div class="score-number" :style="`color:${scoreColor}`">{{ securityScore }}</div>
                <div class="score-label">/ 100</div>
              </div>
            </div>
          </div>
          <div class="mt-2" style="font-size:0.75rem">
            <span :style="`color:${scoreColor};font-weight:600`">{{ scoreLabel }}</span>
            <div class="sc-meta-sub">Security posture</div>
          </div>
        </div>
      </div>

      <div class="col-xl-3 col-md-6">
        <StatCard label="Active Bans" :value="activeBansCount" sub="all sources" icon="mdi mdi-shield-lock" icon-color="#f04040" icon-bg="rgba(240,64,64,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Failed Logins (24h)" :value="failedLogins" sub="recent attempts" icon="mdi mdi-account-alert" icon-color="#f5a623" icon-bg="rgba(245,166,35,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard
          label="Firewall (UFW)"
          :value="ufwActive ? 'Active' : 'Inactive'"
          sub="network protection"
          icon="mdi mdi-wall"
          :icon-color="ufwActive ? '#22d67c' : '#f04040'"
          :icon-bg="ufwActive ? 'rgba(34,214,124,0.12)' : 'rgba(240,64,64,0.12)'"
        />
      </div>
    </div>

    <!-- Security Services grid -->
    <div class="card sc-panel-card mb-4">
      <div class="card-header">
        <h6><i class="mdi mdi-server-security me-2" style="color:#4a9eff"></i>Security Services</h6>
      </div>
      <div class="card-body">
        <div v-if="services.length === 0 && !loading" class="sc-empty-hint" style="font-size:0.82rem;padding:0.5rem 0">
          No service data available.
        </div>
        <div class="row g-2">
          <div v-for="svc in serviceCards" :key="svc.name" class="col-xl-3 col-md-4 col-6">
            <div class="d-flex align-items-center justify-content-between p-2 rounded" style="background:var(--sc-bg-secondary);border:1px solid var(--sc-border)">
              <div>
                <div style="font-size:0.8rem;font-weight:600;color:var(--sc-text);font-family:monospace;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ svc.label }}</div>
                <div style="font-size:0.68rem;color:var(--sc-text-muted)">{{ svc.stateText }}</div>
              </div>
              <div class="d-flex align-items-center gap-1">
                <span class="badge rounded-pill" :class="svc.healthClass" style="font-size:0.62rem">
                  {{ svc.statusText }}
                </span>
                <button
                  v-if="svc.canForceStart"
                  class="btn btn-sm"
                  style="background:rgba(245,166,35,0.12);color:#f5a623;border:1px solid rgba(245,166,35,0.2);font-size:0.65rem;padding:4px 8px"
                  @click.prevent="forceStartService(svc)"
                  title="Force start {{ svc.label }}"
                >
                  <i class="mdi mdi-play-circle-outline me-1"></i>Start
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Active banned IPs table -->
    <div class="card sc-panel-card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-block-helper me-2" style="color:#f04040"></i>Active Bans</h6>
        <input v-model="banFilter" class="form-control form-control-sm" placeholder="Filter IP…" style="width:160px" />
      </div>
      <div class="card-body p-0" style="max-height:300px;overflow-y:auto">
        <table class="table mb-0">
          <thead><tr><th>IP Address</th><th>Source</th><th>Reason</th><th>Banned By</th><th>Since</th><th>Actions</th></tr></thead>
          <tbody>
            <tr v-if="filteredBans.length === 0">
              <td colspan="6" class="text-center sc-empty-hint" style="font-size:0.8rem;padding:1.5rem">No active bans</td>
            </tr>
            <tr v-for="ban in filteredBans" :key="ban.ip">
              <td class="font-mono sc-danger-mono" style="font-size:0.78rem">{{ ban.ip }}</td>
              <td><span class="badge badge-info">{{ ban.source }}</span></td>
              <td class="sc-cell-main" style="font-size:0.75rem">{{ ban.reason }}</td>
              <td class="sc-cell-main" style="font-size:0.75rem">{{ ban.banned_by || '—' }}</td>
              <td class="sc-cell-muted" style="font-size:0.72rem">{{ timeAgo(ban.ts) }}</td>
              <td>
                <button class="btn btn-sm btn-sc-danger" style="font-size:0.68rem;padding:2px 8px" @click="unban(ban.ip)">
                  <i class="mdi mdi-lock-open-outline me-1"></i>Unban
                </button>
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
import api from '@/services/api'

export default {
  name: 'SecurityPage',
  components: { PageHeader, StatCard },

  data() {
    return {
      loading:         false,
      error:           null,
      banFilter:       '',

      securityScore:   0,
      failedLogins:    0,
      activeBansCount: 0,
      ufwActive:       false,
      services:        [],
      bansTable:       [],

      pollTimer:      null
    }
  },

  computed: {
    scoreColor() {
      if (this.securityScore >= 80) return '#22d67c'
      if (this.securityScore >= 60) return '#f5a623'
      return '#f04040'
    },
    scoreLabel() {
      if (this.securityScore >= 80) return 'Good'
      if (this.securityScore >= 60) return 'Fair'
      return 'At Risk'
    },
    filteredBans() {
      if (!this.banFilter) return this.bansTable
      return this.bansTable.filter(b => b.ip.includes(this.banFilter))
    },
    serviceCards() {
      return (this.services || []).map((svc) => {
        const name = svc.name || svc.Name || ''
        const label = svc.label || svc.Label || name || 'unknown'
        const activeState = svc.active_state || svc.ActiveState || 'inactive'
        const subState = svc.sub_state || svc.SubState || 'dead'
        const running = Boolean(svc.running ?? svc.IsRunning)
        const installed = Boolean(svc.installed)
        const healthy = activeState === 'active' && (running || (subState === 'exited' && (name === 'ufw' || name === 'apparmor')))
        return {
          name,
          label,
          stateText: `${activeState}/${subState}`,
          statusText: healthy ? 'Healthy' : (activeState === 'active' ? 'Active' : 'Inactive'),
          healthClass: healthy ? 'badge-online' : (activeState === 'active' ? 'badge-warning' : 'badge-offline'),
          canForceStart: installed && !healthy
        }
      })
    }
  },

  mounted() {
    this.loadData()
    this.pollTimer = setInterval(() => this.loadData(), 30000)
  },

  beforeUnmount() {
    clearInterval(this.pollTimer)
  },

  methods: {
    async loadData() {
      this.loading = true
      this.error   = null
      try {
        const [statusRes, bansRes] = await Promise.all([
          api.getSecurityStatus(),
          api.getBans()
        ])

        const s              = statusRes.data
        this.securityScore   = s.security_score ?? 0
        this.failedLogins    = s.failed_logins   ?? 0
        this.activeBansCount = s.active_bans      ?? 0
        this.ufwActive       = s.ufw_active       ?? false
        this.services        = s.services         ?? []

        this.bansTable = (bansRes.data ?? []).map(b => ({
          ip:       b.ip,
          source:   b.source,
          reason:   b.reason,
          banned_by: b.banned_by,
          ts:       b.ts
        }))
      } catch (err) {
        this.error = err.response?.data?.error || 'Failed to load security data'
      } finally {
        this.loading = false
      }
    },

    timeAgo(ts) {
      if (!ts) return '—'
      try {
        const diff = Date.now() - new Date(ts).getTime()
        const mins = Math.floor(diff / 60000)
        if (mins < 1)  return 'just now'
        if (mins < 60) return `${mins}m ago`
        const hrs = Math.floor(mins / 60)
        if (hrs < 24)  return `${hrs}h ago`
        return `${Math.floor(hrs / 24)}d ago`
      } catch {
        return ts
      }
    },

    async forceStartService(svc) {
      const result = await this.$swal({
        title: `Force start ${svc.label}?`,
        text: `This will attempt to start the ${svc.label} service unit.`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Start',
        confirmButtonColor: '#f5a623'
      })
      if (!result.isConfirmed) return
      try {
        await api.serviceAction(svc.name, 'start')
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${svc.label} starting`, showConfirmButton: false, timer: 2000 })
        await this.loadData()
      } catch (err) {
        this.$swal({ icon: 'error', title: 'Start failed', text: err.response?.data?.error || 'Could not start service' })
      }
    },

    async unban(ip) {
      const result = await this.$swal({
        title: `Unban ${ip}?`,
        text: 'This will remove the ban.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Unban',
        confirmButtonColor: '#f04040'
      })
      if (!result.isConfirmed) return
      try {
        await api.unban(ip)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${ip} unbanned`, showConfirmButton: false, timer: 2000 })
        await this.loadData()
      } catch (err) {
        this.$swal({ icon: 'error', title: 'Error', text: err.response?.data?.error || 'Failed to unban IP' })
      }
    }
  }
}
</script>

<style scoped>
.sc-panel-card {
  border-radius: 12px;
}

.sc-meta-sub {
  color: var(--sc-text-muted);
  font-size: 0.7rem;
  margin-top: 2px;
}

.sc-empty-hint {
  color: var(--sc-text-muted);
}

.sc-danger-mono {
  color: var(--sc-red);
}

.sc-cell-main {
  color: var(--sc-text-secondary);
}

.sc-cell-muted {
  color: var(--sc-text-muted);
}

.sc-view-security :deep(.card-header) {
  padding: 0.85rem 1rem;
}

.sc-view-security :deep(.card-body) {
  padding: 1rem;
}
</style>
