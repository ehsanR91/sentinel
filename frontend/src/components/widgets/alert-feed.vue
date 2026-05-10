<template>
  <div class="card h-100">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6><i class="mdi mdi-bell-outline me-2" style="color:#f5a623"></i>Recent Alerts</h6>
      <router-link to="/alerts" style="font-size:0.75rem;color:#4a9eff">View all</router-link>
    </div>
    <div class="card-body p-0">
      <div class="alert-feed px-3">
        <div v-if="loading" class="text-center py-4" style="color:#5a7499;font-size:0.8rem">
          <i class="mdi mdi-loading mdi-spin me-1"></i>Loading…
        </div>
        <div v-else-if="alerts.length === 0" class="text-center py-4" style="color:#5a7499;font-size:0.8rem">
          No recent alerts.
        </div>
        <div
          v-for="alert in alerts"
          :key="alert.id"
          class="feed-item"
        >
          <div class="feed-icon" :style="`background:rgba(${severityRgb(alert.severity)},0.12)`">
            <i :class="severityIcon(alert.severity)" :style="`color:rgba(${severityRgb(alert.severity)},1)`"></i>
          </div>
          <div class="feed-body">
            <div class="feed-msg">{{ alert.message }}</div>
            <div class="feed-time">
              <span class="badge rounded-pill me-1" :style="`background:rgba(${severityRgb(alert.severity)},0.15);color:rgba(${severityRgb(alert.severity)},1);font-size:0.6rem`">{{ alert.severity }}</span>
              {{ timeAgo(alert.ts) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '@/services/api'
import { useAuthStore } from '@/stores/auth'

export default {
  name: 'AlertFeed',
  setup() {
    return {
      authStore: useAuthStore()
    }
  },
  props: {
    compact: { type: Boolean, default: false }
  },
  data() {
    return {
      alerts: [],
      loading: false
    }
  },

  mounted() {
    this.loadAlerts()
  },

  methods: {
    async loadAlerts() {
      if (!this.authStore.loggedIn) return
      this.loading = true
      try {
        const { data } = await api.getAlerts()
        const list = Array.isArray(data) ? data : (data.alerts || [])
        this.alerts = list.slice(0, this.compact ? 3 : 5)
      } catch (err) {
        if (err.response?.status !== 401) {
          console.error('Failed to load alerts:', err)
        }
      } finally {
        this.loading = false
      }
    },
    severityRgb(s) {
      return { emergency: '240,64,64', critical: '240,64,64', warning: '245,166,35', info: '74,158,255' }[s] || '138,164,200'
    },
    severityIcon(s) {
      return {
        emergency: 'mdi mdi-alert-octagon',
        critical:  'mdi mdi-alert-circle',
        warning:   'mdi mdi-alert',
        info:      'mdi mdi-information'
      }[s] || 'mdi mdi-bell'
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
