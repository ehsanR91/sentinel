<template>
  <section class="service-health sc-focus-ring">
    <div class="service-health__header">
      <div>
        <div class="service-health__eyebrow">Service Health</div>
        <h3 class="service-health__title">{{ summaryLine }}</h3>
      </div>
      <div class="service-health__actions">
        <div class="service-health__filters sc-pill-nav" role="tablist" aria-label="Service health filters">
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
    <div v-show="!isCollapsed" class="service-health__body">

    <div v-if="loading && !services.length" class="service-health__grid" aria-hidden="true">
      <div v-for="n in 8" :key="`svc-skeleton-${n}`" class="service-health__tile service-health__tile--skeleton"></div>
    </div>

    <div v-else-if="error && !services.length" class="service-health__empty">
      <i class="mdi mdi-alert-circle-outline"></i>
      <span>Service state could not be loaded.</span>
    </div>

    <div v-else class="service-health__grid">
      <!-- Not-installed: prompt card -->
      <article
        v-for="service in filteredServices"
        :key="service.name"
        class="service-health__tile"
        :class="`service-health__tile--${service.state}`"
        @click="service.state !== 'not-installed' && openServicePage(service)"
      >
        <template v-if="service.state === 'not-installed'">
          <div class="service-health__tile-header">
            <div class="service-health__name service-health__name--muted">{{ service.label || service.name }}</div>
            <span class="service-health__state-icon service-health__state-icon--not-installed">
              <i class="mdi mdi-package-variant-closed"></i>
            </span>
          </div>
          <div class="service-health__subtext">Not installed</div>
          <div class="service-health__install-cta">
            <button type="button" class="service-health__install-btn" @click.stop="openServicePage(service)">
              <i class="mdi mdi-plus-circle-outline"></i> Install
            </button>
          </div>
        </template>

        <template v-else>
          <div class="service-health__tile-header">
            <div class="service-health__name">{{ service.label || service.name }}</div>
            <Tooltip
              :label="`${service.state.charAt(0).toUpperCase() + service.state.slice(1)}`"
              :description="serviceDetail(service)"
              variant="rich"
              as-child
            >
              <span
                class="service-health__state-icon"
                :class="`service-health__state-icon--${service.state}`"
                :aria-label="`Status: ${service.state}`"
              >
                <i class="mdi" :class="stateIcon(service.state)"></i>
              </span>
            </Tooltip>
          </div>
          <div class="service-health__subtext">{{ serviceDetail(service) }}</div>

          <div class="service-health__uptime-strip">
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
        </template>
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
        { key: 'failed', label: 'Failed' },
        { key: 'not-installed', label: 'Not Installed' }
      ]
    },
    counts() {
      const counts = { all: this.services.length, running: 0, stopped: 0, disabled: 0, failed: 0, 'not-installed': 0 }
      this.services.forEach(service => {
        if (counts[service.state] !== undefined) counts[service.state] += 1
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
      if (!service.installed) return 'not-installed'
      if (service.active_state === 'failed' || service.sub_state === 'failed') return 'failed'
      if (service.running || (service.active_state === 'active' && service.sub_state === 'exited')) return 'running'
      // installed but not enabled/started = intentionally disabled
      if (service.active_state === 'inactive' && (!service.sub_state || service.sub_state === 'dead')) return 'disabled'
      return 'stopped'
    },
    sortWeight(service) {
      return { failed: 0, stopped: 1, disabled: 2, running: 3 }[service.state] ?? 4
    },
    serviceDetail(service) {
      if (!service.installed) return 'Not installed on this host'
      if (service.state === 'disabled') return 'Installed · intentionally off'
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
    },
    stateIcon(state) {
      return {
        running:         'mdi-check-circle',
        stopped:         'mdi-pause-circle',
        failed:          'mdi-alert-circle',
        disabled:        'mdi-power-plug-off-outline',
        'not-installed': 'mdi-package-variant-closed'
      }[state] || 'mdi-circle-outline'
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

.service-health__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
  align-items: center;
}

.service-health__filters {
  /* horizontal scroll handled by .sc-pill-nav */
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

.service-health__body {
  /* no scroll — grid wraps */
}

.service-health__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.service-health__tile {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 12px 18px;
  border-radius: 18px;
  border: 1px solid var(--dashboard-panel-border);
  background: var(--surface-2);
  position: relative;
  overflow: hidden;
  cursor: pointer;
  box-sizing: border-box;
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
  background: var(--surface-2);
  border-color: var(--border-subtle);
  opacity: 0.75;
}

.service-health__tile--not-installed {
  background: var(--surface-1);
  border: 1px dashed var(--border-subtle);
  cursor: default;
}

.service-health__tile--not-installed:hover {
  transform: none;
  background: var(--surface-1);
  border-color: color-mix(in srgb, var(--accent) 30%, var(--border-subtle));
}

.service-health__name--muted {
  color: var(--text-tertiary);
}

.service-health__install-cta {
  flex: 1;
  display: flex;
  align-items: flex-end;
}

.service-health__install-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 700;
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent) 24%, transparent);
  border-radius: 999px;
  padding: 4px 10px;
  cursor: pointer;
  transition: background 0.14s ease;
}

.service-health__install-btn:hover {
  background: color-mix(in srgb, var(--accent) 18%, transparent);
}

.service-health__tile--skeleton {
  min-height: 100px;
  background: linear-gradient(90deg, rgba(138, 164, 200, 0.14) 25%, rgba(138, 164, 200, 0.28) 50%, rgba(138, 164, 200, 0.14) 75%);
  background-size: 200% 100%;
  animation: service-shimmer 1.4s linear infinite;
}

.service-health__tile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-width: 0;
}

.service-health__uptime-strip {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 6px;
  overflow: hidden;
}

.service-health__uptime-strip :deep(.uptime-bar) {
  height: 6px;
  gap: 1px;
  border-radius: 0;
}

.service-health__uptime-strip :deep(.uptime-bar__segment) {
  border-radius: 0;
}

.service-health__state-icon {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  cursor: default;
}

.service-health__state-icon--running       { color: var(--state-ok); }
.service-health__state-icon--stopped       { color: var(--state-warn); }
.service-health__state-icon--failed        { color: var(--state-error); }
.service-health__state-icon--disabled      { color: var(--text-tertiary); }
.service-health__state-icon--not-installed { color: var(--border-strong, #4b5563); }

.service-health__name {
  color: var(--text-primary);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1 1 0;
  min-width: 0;
}

.service-health__subtext {
  color: var(--text-secondary);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.service-health__empty {
  color: var(--text-secondary);
  font-size: 12px;
}

.service-health__tile-actions {
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