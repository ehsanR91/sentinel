<template>
  <section class="service-health sc-focus-ring">
    <div class="service-health__header">
      <div>
        <div class="service-health__eyebrow">Service Health</div>
        <h3 class="service-health__title">{{ summaryLine }}</h3>
      </div>
      <div class="service-health__actions">
        <div class="service-health__filters" role="tablist" aria-label="Service health filters">
          <button
            v-for="option in filters"
            :key="option.key"
            type="button"
            class="service-health__filter"
            :class="{ 'is-active': activeFilter === option.key }"
            @click="activeFilter = option.key"
          >
            {{ option.label }}
            <span>{{ counts[option.key] }}</span>
          </button>
        </div>
        <AppButton variant="secondary" size="sm" icon="mdi mdi-restart" label="Restart all stopped" :disabled="!counts.stopped || busyAll" @click="restartStopped" />
        <AppButton variant="secondary" size="sm" icon="mdi mdi-refresh" label="Reload panel" :loading="loading" @click="refreshServices" />
        <button type="button" class="service-health__filter" style="padding: 0 8px; margin-left: 8px;" @click="isCollapsed = !isCollapsed"><i class="mdi" :class="isCollapsed ? 'mdi-chevron-down' : 'mdi-chevron-up'"></i></button>
      </div>
    </div>
    <div v-show="!isCollapsed">

    <div v-if="loading && !services.length" class="service-health__grid" aria-hidden="true">
      <div v-for="n in 8" :key="`svc-skeleton-${n}`" class="service-health__tile service-health__tile--skeleton"></div>
    </div>

    <div v-else-if="error && !services.length" class="service-health__empty">
      <i class="mdi mdi-alert-circle-outline"></i>
      <span>Service state could not be loaded.</span>
    </div>

    <div v-else class="service-health__grid">
      <article
        v-for="service in filteredServices"
        :key="service.name"
        class="service-health__tile"
        :class="`service-health__tile--${service.state}`"
        @click="openServicePage(service)"
      >
        <div class="service-health__tile-top">
          <div class="service-health__identity">
            <span class="service-health__dot" :class="`service-health__dot--${service.state}`"></span>
              <div class="service-health__name-group">
                <Tooltip :label="`Service: ${service.label || service.name}`" as-child>
                  <div class="service-health__name">{{ service.label || service.name }}</div>
                </Tooltip>
                <div class="service-health__subtext">{{ serviceDetail(service) }}</div>
              </div>
            </div>
            <Tooltip
              v-if="service.state === 'disabled'"
              label="Service disabled"
              :description="`${service.label || service.name} is not installed or is currently unavailable on this host.`"
              variant="rich"
              as-child
            >
              <span class="service-health__disabled-marker" aria-label="Service disabled">
                <i class="mdi mdi-power-plug-off-outline"></i>
              </span>
            </Tooltip>
            <span v-else class="service-health__pill" :class="`service-health__pill--${service.state}`">{{ service.state }}</span>
          <UptimeBar :history="service.history" :max-segments="24" />
        </div>

        <div class="service-health__tile-actions">
          <Tooltip :label="service.running ? 'Restart' : 'Start'" as-child>
            <button type="button" class="service-health__action-icon" @click.stop="performAction(service, service.running ? 'restart' : 'start')" :disabled="!!busy[service.name]">
              <i :class="busy[service.name] ? 'mdi mdi-loading mdi-spin' : `mdi mdi-${service.running ? 'restart' : 'play'}`"></i>
            </button>
          </Tooltip>
          <Tooltip label="Logs" as-child>
            <button type="button" class="service-health__action-icon" @click.stop="openLogs(service)">
              <i class="mdi mdi-file-document-outline"></i>
            </button>
          </Tooltip>
          <Tooltip :label="service.running ? 'Stop' : 'Options'" as-child>
            <button type="button" class="service-health__action-icon" @click.stop="performAction(service, 'stop')" :disabled="!service.running || !!busy[service.name]">
              <i class="mdi mdi-stop"></i>
            </button>
          </Tooltip>
          <Tooltip label="Open panel" as-child>
            <button type="button" class="service-health__action-icon" @click.stop="openServicePage(service)">
              <i class="mdi mdi-arrow-top-right"></i>
            </button>
          </Tooltip>
        </div>
      </article>
    </div>
    </div>
  </section>
</template>

<script>
import AppButton from '@/components/ui/app-button.vue'
import Tooltip from '@/components/ui/tooltip.vue'
import UptimeBar from '@/components/ui/uptime-bar.vue'
import api from '@/services/api'

const HISTORY_LIMIT = 24

function cloneHistory(history = []) {
  return history.slice(-HISTORY_LIMIT)
}

export default {
  name: 'DashboardServiceHealthPanel',
  components: { AppButton, Tooltip, UptimeBar },
  data() {
    return {
      isCollapsed: false,
      services: [],
      serviceHistory: {},
      activeFilter: 'all',
      loading: false,
      error: false,
      busy: {},
      busyAll: false,
      refreshTimer: null
    }
  },
  computed: {
    filters() {
      return [
        { key: 'all', label: 'All' },
        { key: 'running', label: 'Running' },
        { key: 'stopped', label: 'Stopped' },
        { key: 'disabled', label: 'Disabled' },
        { key: 'failed', label: 'Failed' }
      ]
    },
    counts() {
      const counts = { all: this.services.length, running: 0, stopped: 0, disabled: 0, failed: 0 }
      this.services.forEach(service => {
        counts[service.state] += 1
      })
      return counts
    },
    summaryLine() {
      return `${this.counts.running}/${this.counts.all} active · ${this.counts.stopped} stopped · ${this.counts.disabled} disabled · ${this.counts.failed} failed`
    },
    filteredServices() {
      const list = this.activeFilter === 'all'
        ? this.services
        : this.services.filter(service => service.state === this.activeFilter)
      return [...list].sort((left, right) => this.sortWeight(left) - this.sortWeight(right) || left.label.localeCompare(right.label))
    }
  },
  async mounted() {
    await this.refreshServices()
    this.refreshTimer = setInterval(() => this.refreshServices(), 60000)
  },
  beforeUnmount() {
    clearInterval(this.refreshTimer)
  },
  methods: {
    normalizeService(service) {
      const state = this.serviceState(service)
      const previous = this.serviceHistory[service.name] || []
      const nextHistory = cloneHistory([...previous, state === 'running'])
      this.serviceHistory = {
        ...this.serviceHistory,
        [service.name]: nextHistory
      }
      return {
        ...service,
        state,
        history: nextHistory
      }
    },
    serviceState(service) {
      if (!service.installed) return 'disabled'
      if (service.active_state === 'failed' || service.sub_state === 'failed') return 'failed'
      if (service.running || (service.active_state === 'active' && service.sub_state === 'exited')) return 'running'
      return 'stopped'
    },
    sortWeight(service) {
      return { failed: 0, stopped: 1, disabled: 2, running: 3 }[service.state] ?? 4
    },
    serviceDetail(service) {
      if (!service.installed) return 'Not installed'
      if (service.running) return `${service.active_state} · ${service.sub_state}`
      return `${service.active_state || 'inactive'} · ${service.sub_state || 'stopped'}`
    },
    async refreshServices() {
      this.loading = true
      this.error = false
      try {
        const { data } = await api.getServices()
        const services = Array.isArray(data) ? data : []
        this.services = services.map(service => this.normalizeService(service))
      } catch (error) {
        this.error = true
      } finally {
        this.loading = false
      }
    },
    async performAction(service, action) {
      this.busy = { ...this.busy, [service.name]: true }
      try {
        await api.serviceAction(service.name, action)
        await this.refreshServices()
      } catch (_) {
        this.error = true
      } finally {
        const nextBusy = { ...this.busy }
        delete nextBusy[service.name]
        this.busy = nextBusy
      }
    },
    async restartStopped() {
      const stopped = this.services.filter(service => service.state === 'stopped')
      if (!stopped.length) return
      if (!window.confirm(`Start ${stopped.length} stopped services?`)) return
      this.busyAll = true
      try {
        for (const service of stopped) {
          await api.serviceAction(service.name, 'start')
        }
        await this.refreshServices()
      } finally {
        this.busyAll = false
      }
    },
    openLogs(service) {
      this.$router.push({ path: '/services', query: { service: service.name, panel: 'logs' } })
    },
    openServicePage(service) {
      this.$router.push({ path: '/services', query: { service: service.name } })
    }
  }
}
</script>

<style scoped>
.service-health {
  border-radius: 22px;
  border: 1px solid var(--dashboard-panel-border);
  background: var(--dashboard-panel-bg);
  box-shadow: var(--shadow-md);
  padding: 18px;
}

.service-health__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.service-health__eyebrow {
  color: var(--text-tertiary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 700;
}

.service-health__title {
  margin: 4px 0 0;
  font-size: 18px;
  color: var(--text-primary);
}

.service-health__actions,
.service-health__filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

.service-health__filter {
  border-radius: 999px;
  border: 1px solid var(--dashboard-panel-border);
  background: transparent;
  color: var(--text-secondary);
  padding: 8px 12px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 700;
}

.service-health__filter span {
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  background: var(--border-subtle);
  display: grid;
  place-items: center;
  font-size: 10px;
}

.service-health__filter.is-active {
  background: var(--accent-muted);
  color: var(--text-primary);
}

.service-health__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 14px;
}

.service-health__tile {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 18px;
  border: 1px solid var(--dashboard-panel-border);
  background: var(--surface-2);
  position: relative;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.16s ease, background 0.16s ease, border-color 0.16s ease;
}

.service-health__tile:hover {
  transform: translateY(-1px);
  background: var(--surface-3);
  border-color: color-mix(in srgb, var(--accent) 26%, var(--dashboard-panel-border));
}

.service-health__tile--failed,
.service-health__tile--stopped {
  background: rgba(255, 106, 106, 0.05);
}

.service-health__tile--disabled {
  background:
    linear-gradient(135deg, rgba(255, 106, 106, 0.12), rgba(255, 106, 106, 0.035)),
    var(--surface-2);
  border-color: rgba(255, 106, 106, 0.24);
}

.service-health__tile--skeleton {
  min-height: 146px;
  background: linear-gradient(90deg, rgba(138, 164, 200, 0.14) 25%, rgba(138, 164, 200, 0.28) 50%, rgba(138, 164, 200, 0.14) 75%);
  background-size: 200% 100%;
  animation: service-shimmer 1.4s linear infinite;
}

.service-health__tile-top,
.service-health__identity,
.service-health__tile-actions {
  display: flex;
  justify-content: space-between;
  gap: 10px;
}

.service-health__tile-top {
  min-width: 0;
}

.service-health__name-group {
  min-width: 0;
}

.service-health__identity {
  align-items: flex-start;
  min-width: 0;
  flex: 1 1 auto;
}

.service-health__identity > div {
  min-width: 0;
}

.service-health__name {
  color: var(--text-primary);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.service-health__subtext,
.service-health__empty {
  color: var(--text-secondary);
  font-size: 12px;
}

.service-health__dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  margin-top: 4px;
}

.service-health__dot--running { background: var(--state-ok); }
.service-health__dot--stopped { background: var(--state-warn); }
.service-health__dot--disabled { background: var(--state-muted); }
.service-health__dot--failed { background: var(--state-error); }

.service-health__pill {
  border-radius: 999px;
  padding: 5px 8px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 10px;
  font-weight: 700;
  align-self: flex-start;
}

.service-health__pill--running { background: var(--state-ok-bg); color: var(--state-ok-fg); }
.service-health__pill--stopped { background: var(--state-warn-bg); color: var(--state-warn-fg); }
.service-health__pill--disabled { background: var(--state-muted-bg); color: var(--state-muted-fg); }
.service-health__pill--failed { background: var(--state-error-bg); color: var(--state-error-fg); }

.service-health__disabled-marker {
  width: 28px;
  height: 28px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  color: var(--state-error-fg);
  background: var(--state-error-bg);
  border: 1px solid rgba(255, 106, 106, 0.24);
}

.service-health__history {
  display: flex;
}

.service-health__history-segment {
  height: 8px;
  border-radius: 999px;
  background: var(--border-subtle);
}

.service-health__history-segment.is-up { background: var(--state-ok); opacity: 0.7; }
.service-health__history-segment.is-down { background: var(--state-error); opacity: 0.45; }

.service-health__tile-actions {
  margin-top: auto;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  transform: translateY(110%);
  opacity: 0;
  transition: transform 100ms ease, opacity 100ms ease;
}

.service-health__tile:hover .service-health__tile-actions,
.service-health__tile:focus-within .service-health__tile-actions {
  transform: translateY(0);
  opacity: 1;
}

.service-health__tile--failed .service-health__tile-actions,
.service-health__tile--stopped .service-health__tile-actions {
  transform: translateY(0);
  opacity: 1;
}

@media (hover: none) {
  .service-health__tile-actions {
    display: flex;
    transform: none;
    opacity: 1;
  }
}

.service-health__action-icon {
  width: 32px;
  height: 32px;
  border-radius: 12px;
  border: 1px solid var(--dashboard-panel-border);
  background: var(--surface-3);
  color: var(--text-secondary);
  display: inline-flex;
  justify-content: center;
  align-items: center;
  font-size: 16px;
  transition: background 0.12s ease, color 0.12s ease;
  flex: none;
}

.service-health__action-icon:hover {
  background: var(--accent-muted);
  color: var(--accent);
  border-color: color-mix(in srgb, var(--accent) 24%, var(--dashboard-panel-border));
}

.service-health__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 40px 18px;
}

@keyframes service-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

@media (max-width: 960px) {
  .service-health__header {
    flex-direction: column;
  }
}
</style>