<template>
  <div class="card h-100">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h6><i class="mdi mdi-cog-outline me-2" style="color:#4a9eff"></i>Service Health</h6>
      <span style="font-size:0.72rem;color:#5a7499">{{ activeCount }}/{{ services.length }} active</span>
    </div>
    <div class="card-body">

      <!-- Loading shimmer (initial load only) -->
      <div v-if="loading" class="service-grid">
        <div v-for="n in 8" :key="n" class="service-tile shimmer"></div>
      </div>

      <!-- Error / empty state -->
      <div v-else-if="error" class="svc-empty-state">
        <i class="mdi mdi-alert-circle-outline"></i>
        <span>Service data unavailable</span>
      </div>

      <!-- Service tiles -->
      <div v-else class="service-grid">
        <Tooltip
          v-for="svc in displayedServices"
          :key="svc.name"
          :label="svc.label"
          :description="svc.name"
          variant="rich"
          as-child
        >
          <div
            class="service-tile"
            :class="svc.status"
          >
            <i :class="svc.icon"></i>
            <div class="svc-name">{{ svc.label }}</div>
            <div class="svc-status">{{ statusLabel(svc.status) }}</div>
          </div>
        </Tooltip>
      </div>

    </div>
  </div>
</template>

<script>
import Tooltip from '@/components/ui/tooltip.vue'
import config from '@/app.config.json'
import api from '@/services/api'
import { useAuthStore } from '@/stores/auth'

const ICONS = {
  'ufw':           'mdi mdi-wall',
  'fail2ban':      'mdi mdi-lock-check',
  'crowdsec':      'mdi mdi-shield-lock',
  'psad':          'mdi mdi-radar',
  'clamav-daemon': 'mdi mdi-virus-off',
  'auditd':        'mdi mdi-file-eye-outline',
  'apparmor':      'mdi mdi-shield-half-full',
  'docker':        'mdi mdi-docker',
  'aide':          'mdi mdi-database-check-outline',
  'rkhunter':      'mdi mdi-magnify-scan',
  'nginx':         'mdi mdi-web',
  'ssh':           'mdi mdi-ssh',
  'sshd':          'mdi mdi-ssh',
}

const LABELS = {
  'ufw':                 'UFW',
  'fail2ban':            'fail2ban',
  'crowdsec':            'CrowdSec',
  'psad':                'psad',
  'clamav-daemon':       'ClamAV',
  'auditd':              'auditd',
  'apparmor':            'AppArmor',
  'docker':              'Docker',
  'netdata':             'Netdata',
  'unattended-upgrades': 'Auto-Update',
  'aide':                'AIDE',
  'rkhunter':            'rkhunter',
  'nginx':               'nginx',
  'ssh':                 'SSH',
  'sshd':                'sshd',
}

function mapService(svc) {
  const name = svc.name
  return {
    name,
    label: LABELS[name] ?? (name.charAt(0).toUpperCase() + name.slice(1)),
    icon:  ICONS[name]  ?? 'mdi mdi-cog-outline',
    status: svc.active_state === 'active' ? 'active' : 'inactive',
  }
}

export default {
  name: 'ServiceStatus',
  setup() {
    return {
      authStore: useAuthStore()
    }
  },
  components: { Tooltip },
  props: {
    compact: { type: Boolean, default: false }
  },

  data() {
    return {
      services:       [],
      loading:        false,
      error:          false,
      refreshTimer:  null,
    }
  },

  computed: {
    displayedServices() {
      return this.compact ? this.services.slice(0, 6) : this.services
    },
    activeCount() {
      return this.services.filter(s => s.status === 'active').length
    },
  },

  async mounted() {
    await this.loadServices()
    this.refreshTimer = setInterval(() => this.loadServices(), config.pollIntervalMs)
  },

  beforeUnmount() {
    clearInterval(this.refreshTimer)
  },

  methods: {
    async loadServices() {
      if (!this.authStore.loggedIn) return
      // Show shimmer only on the very first load (no data yet)
      if (this.services.length === 0) this.loading = true
      try {
        const { data } = await api.getServices()
        this.services = Array.isArray(data) ? data.map(mapService) : []
        this.error = false
      } catch (err) {
        if (err.response?.status !== 401) {
          console.error('Failed to load services:', err)
        }
      } finally {
        this.loading = false
      }
    },

    statusLabel(s) {
      return s === 'active' ? 'Running' : s === 'warning' ? 'Warning' : 'Stopped'
    },
  },
}
</script>

<style scoped>
/* Shimmer placeholder tiles */
.shimmer {
  background: linear-gradient(90deg, #1e2a3a 25%, #263040 50%, #1e2a3a 75%);
  background-size: 200% 100%;
  animation: shimmer-sweep 1.4s infinite;
  min-height: 72px;
  border-radius: 6px;
}

@keyframes shimmer-sweep {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Empty / error state */
.svc-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 2rem 0;
  color: #5a7499;
  font-size: 0.82rem;
}

.svc-empty-state i {
  font-size: 1.6rem;
}
</style>
