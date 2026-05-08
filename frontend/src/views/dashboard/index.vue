<template>
  <div ref="pageEl" class="dashboard-page">
    <div
      class="ptr-bar"
      :class="{ pulling: isPulling, refreshing: isRefreshing }"
      :style="{ height: `${Math.min(pullDist, 64)}px`, opacity: Math.min(pullDist / 64, 1) }"
    >
      <i class="mdi" :class="isRefreshing ? 'mdi-loading mdi-spin' : pullDist >= 64 ? 'mdi-refresh' : 'mdi-arrow-down'"></i>
    </div>

    <PageHeader title="Dashboard" icon="mdi mdi-view-dashboard" :items="breadcrumbs">
      <template #actions>
        <div class="dashboard-header-actions">
          <AppButton
            variant="secondary"
            size="sm"
            :icon="layoutEditMode ? 'mdi mdi-lock-outline' : 'mdi mdi-pencil-ruler-outline'"
            :label="layoutEditMode ? 'Lock layout' : 'Edit layout'"
            @click="toggleLayoutEdit"
          />
          <AppButton
            variant="secondary"
            size="sm"
            :icon="isFullscreen ? 'mdi mdi-fullscreen-exit' : 'mdi mdi-fullscreen'"
            :label="isFullscreen ? 'Exit focus' : 'Focus mode'"
            @click="toggleFullscreen"
          />
          <label class="dashboard-select-wrap">
            <span>Preset</span>
            <select v-model="activePreset" class="dashboard-select" @change="applyPreset(activePreset)">
              <option v-for="preset in presetOptions" :key="preset.value" :value="preset.value">{{ preset.label }}</option>
            </select>
          </label>
          <label class="dashboard-select-wrap dashboard-select-wrap--compact">
            <span>Refresh</span>
            <select v-model.number="auxRefreshSec" class="dashboard-select" @change="persistDashboardState">
              <option :value="30">30s</option>
              <option :value="60">60s</option>
              <option :value="120">2m</option>
            </select>
          </label>
          <AppButton variant="secondary" size="sm" icon="mdi mdi-view-grid-plus-outline" label="Add widget" @click="showWidgetCatalog = true" />
          <div class="dashboard-refresh-block">
            <span class="dashboard-refresh-note">Last sync {{ formatRelativeFromNow(lastLoadedAt) }}</span>
            <AppButton variant="primary" size="sm" icon="mdi mdi-refresh" :loading="isRefreshing" label="Refresh" @click="refreshAll" />
          </div>
        </div>
      </template>
    </PageHeader>

    <section class="dashboard-hero-grid">
      <HealthRing
        class="dashboard-hero-grid__health"
        :health-data="healthData"
        :history="healthHistory"
        :loading="healthLoading"
        :stale="isAuxStale"
        @open="showHealthDrawer = true"
        @inspect-issue="openHealthIssue"
      />

      <article class="dashboard-panel dashboard-server-card">
        <div class="dashboard-panel__header">
          <div>
            <div class="dashboard-panel__eyebrow">Server Identity</div>
            <h2 class="dashboard-panel__title">{{ snap.hostname || 'Sentinel node' }}</h2>
          </div>
          <div class="dashboard-live-chip" :class="{ 'is-offline': !wsConnected || isMetricStale }">
            <span class="dashboard-live-chip__dot"></span>
            {{ connectionStateLabel }}
          </div>
        </div>

        <div class="dashboard-identity-grid">
          <div v-for="row in identityRows" :key="row.label" class="dashboard-identity-row">
            <span class="dashboard-identity-row__label">{{ row.label }}</span>
            <span class="dashboard-identity-row__value" :class="{ 'is-mono': row.mono }">
              <span>{{ row.value }}</span>
              <button
                v-if="row.copyValue"
                type="button"
                class="dashboard-identity-row__copy sc-focus-ring"
                :aria-label="`Copy ${row.label}`"
                @click="copyIdentityValue(row)"
              >
                <i class="mdi mdi-content-copy" aria-hidden="true"></i>
              </button>
            </span>
          </div>
        </div>
      </article>

      <article class="dashboard-panel dashboard-hero-side">
        <div class="dashboard-panel__header">
          <div>
            <div class="dashboard-panel__eyebrow">Critical Signals</div>
            <h2 class="dashboard-panel__title">Operator Actions</h2>
          </div>
        </div>

        <div class="dashboard-status-pills">
          <button
            v-for="pill in heroPills"
            :key="pill.label"
            type="button"
            class="dashboard-status-pill"
            :class="`dashboard-status-pill--${pill.tone}`"
            @click="navigateTo(pill.route)"
          >
            <span class="dashboard-status-pill__label">{{ pill.label }}</span>
            <span class="dashboard-status-pill__value" :class="`is-${pill.tone}`">
              <span class="dashboard-status-pill__marker" :class="`is-${pill.tone}`">{{ pill.marker }}</span>
              <span>{{ pill.value }}</span>
            </span>
          </button>
        </div>

        <div class="dashboard-actions-grid">
          <AppButton variant="primary" size="md" icon="mdi mdi-shield-search" label="Scan now" @click="runScanAction" />
          <AppButton variant="secondary" size="md" icon="mdi mdi-broom" :loading="cleanerRunning" label="Run cleaner" @click="openCleaner" />
          <AppButton variant="secondary" size="md" icon="mdi mdi-reload-alert" label="Reload services" @click="reloadServicesPanel" />
          <AppButton variant="secondary" size="md" icon="mdi mdi-console-line" label="Open terminal" @click="navigateTo('/terminal')" />
        </div>
      </article>
    </section>

    <draggable
      v-model="kpiWidgets"
      class="dashboard-kpi-grid"
      item-key="id"
      handle=".dashboard-edit-handle"
      :disabled="!layoutEditMode"
      :animation="180"
      ghost-class="drag-ghost"
      chosen-class="drag-chosen"
      @end="persistDashboardState"
    >
      <template #item="{ element }">
        <div class="dashboard-kpi-grid__item">
          <button v-if="layoutEditMode" type="button" class="dashboard-edit-handle" aria-label="Drag widget">
            <i class="mdi mdi-drag"></i>
          </button>
          <button
            v-if="layoutEditMode && kpiWidgets.length > 2"
            type="button"
            class="dashboard-hide-button"
            aria-label="Hide widget"
            @click="hideKpi(element.id)"
          >
            <i class="mdi mdi-eye-off-outline"></i>
          </button>
          <KPICard v-bind="kpiCardById(element.id)" @click="openKpiDrawer(element.id)" />
        </div>
      </template>
    </draggable>

    <draggable
      v-model="sectionWidgets"
      class="dashboard-section-stack"
      item-key="id"
      handle=".dashboard-section-handle"
      :disabled="!layoutEditMode"
      :animation="180"
      ghost-class="drag-ghost"
      chosen-class="drag-chosen"
      @end="persistDashboardState"
    >
      <template #item="{ element }">
        <section class="dashboard-section-shell">
          <div v-if="layoutEditMode" class="dashboard-section-tools">
            <button type="button" class="dashboard-section-handle" aria-label="Drag section">
              <i class="mdi mdi-drag"></i>
            </button>
            <button
              v-if="sectionWidgets.length > 1"
              type="button"
              class="dashboard-hide-button"
              aria-label="Hide section"
              @click="hideSection(element.id)"
            >
              <i class="mdi mdi-eye-off-outline"></i>
            </button>
          </div>

          <template v-if="element.id === 'telemetry'">
            <div class="dashboard-telemetry-grid">
              <TelemetryChart
                title="CPU + Load"
                description="Auto-scales around the current CPU band while preserving alert thresholds."
                icon="mdi mdi-chip"
                :live="wsConnected && !isMetricStale"
                :series="cpuTelemetrySeries"
                :formatter="formatPercentValue"
                :range-options="telemetryRanges"
                :thresholds="cpuTelemetryThresholds"
                :percent-scale="lockCpuToPercent"
              />
              <TelemetryChart
                title="Memory + Swap"
                description="RAM and swap pressure, stacked for fast saturation reads."
                icon="mdi mdi-memory"
                :live="wsConnected && !isMetricStale"
                :series="memoryTelemetrySeries"
                :formatter="formatPercentValue"
                :range-options="telemetryRanges"
                :thresholds="memoryTelemetryThresholds"
                :stacked="true"
                :percent-scale="true"
              />
              <TelemetryChart
                title="Network Traffic"
                description="Linked ingress and egress streams with shared crosshair timing."
                icon="mdi mdi-swap-vertical"
                :live="wsConnected && !isMetricStale"
                :series="networkTelemetrySeries"
                :formatter="formatRateValue"
                :range-options="telemetryRanges"
              />

              <article class="dashboard-panel dashboard-mount-card">
                <div class="dashboard-panel__header">
                  <div>
                    <div class="dashboard-panel__eyebrow">Disk Space by Mount</div>
                    <h3 class="dashboard-panel__title">Storage Pressure</h3>
                  </div>
                  <span class="dashboard-panel__hint">{{ mountRows.length }} mounts</span>
                </div>
                <div class="dashboard-mount-list">
                  <div v-if="!mountRows.length" class="dashboard-empty-inline">Collecting partition metrics…</div>
                  <div v-for="mount in mountRows" :key="mount.mount" class="dashboard-mount-row">
                    <div class="dashboard-mount-row__head">
                      <span class="dashboard-mount-row__mount">{{ mount.mount }}</span>
                      <span class="dashboard-mount-row__value">{{ mount.pct }}%</span>
                    </div>
                    <div class="dashboard-mount-row__meta">{{ mount.used }} / {{ mount.total }} · {{ mount.device }}</div>
                    <div class="dashboard-mount-row__bar">
                      <span class="dashboard-mount-row__fill" :style="{ width: `${mount.pct}%` }" :class="mount.tone"></span>
                    </div>
                  </div>
                </div>
              </article>
            </div>
          </template>

          <ServiceHealthPanel v-else-if="element.id === 'services'" ref="servicePanel" />
          <ActivityFeed v-else-if="element.id === 'activity'" :items-by-tab="activityItemsByTab" @open-item="openActivityItem" />
        </section>
      </template>
    </draggable>

    <section class="dashboard-footer-strip dashboard-panel">
      <div class="dashboard-footer-strip__group">
        <span class="dashboard-footer-strip__label">Agent build</span>
        <span class="dashboard-footer-strip__value">v{{ healthData.version || 'unknown' }}</span>
      </div>
      <div class="dashboard-footer-strip__group">
        <span class="dashboard-footer-strip__label">Last sync</span>
        <span class="dashboard-footer-strip__value">{{ formatRelativeFromNow(lastLoadedAt) }}</span>
      </div>
      <div class="dashboard-footer-strip__group">
        <span class="dashboard-footer-strip__label">Connection</span>
        <span class="dashboard-connection-pill" :class="`dashboard-connection-pill--${connectionTone}`">{{ connectionStateLabel }}</span>
      </div>
      <div class="dashboard-footer-strip__group">
        <span class="dashboard-footer-strip__label">Endpoint</span>
        <span class="dashboard-footer-strip__value dashboard-footer-strip__value--mono">{{ endpointLabel }}</span>
      </div>
      <div class="dashboard-footer-strip__group">
        <span class="dashboard-footer-strip__label">Operator IP</span>
        <span class="dashboard-footer-strip__value dashboard-footer-strip__value--mono">{{ userMeta.client_ip || 'Unavailable' }}</span>
      </div>
    </section>

    <DetailDrawer
      :model-value="showWidgetCatalog"
      title="Widget catalog"
      subtitle="Restore hidden dashboard widgets"
      @update:model-value="showWidgetCatalog = $event"
    >
      <div class="dashboard-drawer-grid">
        <div>
          <h4 class="dashboard-drawer-title">Hidden KPI cards</h4>
          <div class="dashboard-restore-list">
            <button
              v-for="widget in hiddenKpiEntries"
              :key="widget.id"
              type="button"
              class="dashboard-restore-row"
              @click="restoreKpi(widget.id)"
            >
              <span>
                <strong>{{ widget.label }}</strong>
                <small>{{ widget.description }}</small>
              </span>
              <i class="mdi mdi-plus"></i>
            </button>
            <div v-if="!hiddenKpiEntries.length" class="dashboard-empty-inline">No hidden KPI cards.</div>
          </div>
        </div>
        <div>
          <h4 class="dashboard-drawer-title">Hidden sections</h4>
          <div class="dashboard-restore-list">
            <button
              v-for="section in hiddenSectionEntries"
              :key="section.id"
              type="button"
              class="dashboard-restore-row"
              @click="restoreSection(section.id)"
            >
              <span>
                <strong>{{ section.label }}</strong>
                <small>{{ section.description }}</small>
              </span>
              <i class="mdi mdi-plus"></i>
            </button>
            <div v-if="!hiddenSectionEntries.length" class="dashboard-empty-inline">All major sections are visible.</div>
          </div>
        </div>
      </div>
    </DetailDrawer>

    <DetailDrawer
      :model-value="showHealthDrawer"
      :title="`${healthData.score || 0}/100 · ${healthHeadline}`"
      subtitle="Remediation and posture detail"
      @update:model-value="showHealthDrawer = $event"
    >
      <div class="dashboard-health-drawer">
        <div class="dashboard-health-drawer__summary">{{ healthData.summary || 'No health summary available.' }}</div>
        <div class="dashboard-health-drawer__categories">
          <div v-for="category in healthCategories" :key="category.key" class="dashboard-health-category">
            <span>{{ category.label }}</span>
            <strong>{{ category.score }}/100</strong>
          </div>
        </div>
        <div class="dashboard-health-drawer__issues">
          <div v-for="check in healthIssues" :key="check.name" class="dashboard-health-issue-card">
            <div>
              <div class="dashboard-health-issue-card__title">
                <span class="dashboard-health-badge" :class="check.status">{{ check.status }}</span>
                <strong>{{ check.name }}</strong>
              </div>
              <p>{{ check.message }}</p>
              <small>Checked {{ formatTimestamp(check.last_checked || healthData.timestamp) }}</small>
            </div>
            <div class="dashboard-health-issue-card__actions">
              <AppButton variant="secondary" size="sm" label="Fix" :loading="healthFixingName === check.name && healthFixMode === 'auto'" @click="fixHealthIssue(check, 'auto')" />
              <AppButton variant="ghost" size="sm" label="Guide" :loading="healthFixingName === check.name && healthFixMode === 'manual'" @click="fixHealthIssue(check, 'manual')" />
            </div>
          </div>
        </div>
        <div v-if="healthFixResponse" class="dashboard-health-response" :class="healthFixResponse.success ? 'is-success' : 'is-error'">
          <strong>{{ healthFixResponse.message }}</strong>
          <p v-if="healthFixResponse.remedy">{{ healthFixResponse.remedy }}</p>
          <pre v-if="healthFixResponse.command">{{ healthFixResponse.command }}</pre>
        </div>
      </div>
    </DetailDrawer>

    <DetailDrawer
      :model-value="showKpiDrawer"
      :title="selectedKpiDetail?.label || 'KPI detail'"
      subtitle="Current metric context"
      @update:model-value="showKpiDrawer = $event"
    >
      <div v-if="selectedKpiDetail" class="dashboard-kpi-drawer">
        <div class="dashboard-kpi-drawer__hero">
          <div>
            <div class="dashboard-kpi-drawer__value">{{ selectedKpiDetail.value }}</div>
            <div class="dashboard-kpi-drawer__delta" :class="`is-${selectedKpiDetail.deltaTone}`">{{ selectedKpiDetail.deltaLabel || 'No delta yet' }}</div>
          </div>
          <div class="dashboard-kpi-drawer__status">{{ selectedKpiDetail.statusLabel || selectedKpiDetail.rangeLabel }}</div>
        </div>
        <div class="dashboard-kpi-drawer__lines">
          <div v-for="line in selectedKpiDetail.contextLines" :key="line" class="dashboard-kpi-drawer__line">{{ line }}</div>
        </div>
        <div v-if="selectedKpiDetail.threshold" class="dashboard-kpi-drawer__thresholds">
          <div>Warn at {{ selectedKpiDetail.threshold.warn }}</div>
          <div>Critical at {{ selectedKpiDetail.threshold.crit }}</div>
          <div>Current {{ selectedKpiDetail.threshold.value }}</div>
        </div>
      </div>
    </DetailDrawer>

    <div v-if="showCleaner" class="dashboard-cleaner-overlay" @click.self="closeCleaner">
      <div class="dashboard-cleaner-card">
        <div class="dashboard-cleaner-card__header">
          <div>
            <div class="dashboard-panel__eyebrow">Maintenance</div>
            <h3 class="dashboard-panel__title">System Cleaner</h3>
          </div>
          <button type="button" class="dashboard-hide-button" aria-label="Close cleaner" @click="closeCleaner">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <p class="dashboard-cleaner-card__copy">Clear reclaimable package cache and stale temporary files, then sync the latest maintenance status back into the dashboard.</p>
        <div class="dashboard-cleaner-progress">
          <span class="dashboard-cleaner-progress__fill" :style="{ width: `${cleanerProgress}%` }"></span>
        </div>
        <div class="dashboard-cleaner-stats">
          <span>Progress {{ cleanerProgress }}%</span>
          <span>Last reclaimed {{ cleanupStats.last_freed_human || '0 B' }}</span>
        </div>
        <div class="dashboard-cleaner-card__actions">
          <AppButton variant="primary" size="md" icon="mdi mdi-play" :loading="cleanerRunning" :label="cleanerRunning ? 'Cleaning…' : 'Start cleaner'" @click="runCleaner" />
          <AppButton variant="secondary" size="md" label="Close" @click="closeCleaner" />
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import AppButton from '@/components/ui/app-button.vue'
import DetailDrawer from '@/components/ui/detail-drawer.vue'
import HealthRing from '@/components/dashboard/health-ring.vue'
import KPICard from '@/components/dashboard/kpi-card.vue'
import TelemetryChart from '@/components/dashboard/telemetry-chart.vue'
import ActivityFeed from '@/components/dashboard/activity-feed.vue'
import ServiceHealthPanel from '@/components/dashboard/service-health-panel.vue'
import draggable from 'vuedraggable'
import { mapGetters } from 'vuex'

const DASHBOARD_STATE_KEY = 'sc_dashboard_v2_layout'

const DEFAULT_KPI_WIDGETS = [
  { id: 'cpu' },
  { id: 'memory' },
  { id: 'disk' },
  { id: 'network' },
  { id: 'bans' },
  { id: 'logins24h' },
  { id: 'containers' },
  { id: 'uptime' }
]

const DEFAULT_SECTION_WIDGETS = [
  { id: 'telemetry' },
  { id: 'services' },
  { id: 'activity' }
]

const PRESETS = {
  operator: {
    kpis: ['cpu', 'memory', 'disk', 'network', 'bans', 'logins24h', 'containers', 'uptime'],
    sections: ['telemetry', 'services', 'activity']
  },
  security: {
    kpis: ['bans', 'logins24h', 'cpu', 'memory', 'disk', 'network', 'containers', 'uptime'],
    sections: ['activity', 'services', 'telemetry']
  },
  performance: {
    kpis: ['cpu', 'memory', 'disk', 'network', 'uptime', 'containers', 'bans', 'logins24h'],
    sections: ['telemetry', 'activity', 'services']
  },
  compact: {
    kpis: ['cpu', 'memory', 'disk', 'network', 'containers', 'uptime', 'bans', 'logins24h'],
    sections: ['telemetry', 'services', 'activity']
  }
}

function safeLocalState() {
  try {
    return JSON.parse(localStorage.getItem(DASHBOARD_STATE_KEY) || '{}')
  } catch {
    return {}
  }
}

function defaultDashboardState() {
  return {
    layoutEditMode: false,
    activePreset: 'operator',
    auxRefreshSec: 60,
    kpiWidgets: DEFAULT_KPI_WIDGETS,
    hiddenKpis: [],
    sectionWidgets: DEFAULT_SECTION_WIDGETS,
    hiddenSections: []
  }
}

function normalizeWidgetState(entries, fallback) {
  const source = Array.isArray(entries) ? entries : fallback
  const normalized = source
    .map(entry => (typeof entry === 'string' ? { id: entry } : entry))
    .filter(entry => entry && typeof entry.id === 'string' && entry.id)
  return normalized.length ? normalized : fallback
}

function normalizeIdList(entries, fallback = []) {
  if (!Array.isArray(entries)) return fallback
  return [...new Set(entries.filter(entry => typeof entry === 'string' && entry))]
}

function normalizeDashboardState(rawState, fallbackState = defaultDashboardState()) {
  const source = rawState && typeof rawState === 'object' ? rawState : {}
  const kpiWidgets = normalizeWidgetState(source.kpiWidgets, fallbackState.kpiWidgets)
  const sectionWidgets = normalizeWidgetState(source.sectionWidgets, fallbackState.sectionWidgets)
  const allKpiIds = DEFAULT_KPI_WIDGETS.map(item => item.id)
  const allSectionIds = DEFAULT_SECTION_WIDGETS.map(item => item.id)
  const activePreset = PRESETS[source.activePreset] ? source.activePreset : fallbackState.activePreset
  const auxRefreshSec = [30, 60, 120].includes(Number(source.auxRefreshSec))
    ? Number(source.auxRefreshSec)
    : Number(fallbackState.auxRefreshSec || 60)

  return {
    layoutEditMode: typeof source.layoutEditMode === 'boolean' ? source.layoutEditMode : !!fallbackState.layoutEditMode,
    activePreset,
    auxRefreshSec,
    kpiWidgets,
    hiddenKpis: normalizeIdList(source.hiddenKpis, allKpiIds.filter(id => !kpiWidgets.some(widget => widget.id === id))),
    sectionWidgets,
    hiddenSections: normalizeIdList(source.hiddenSections, allSectionIds.filter(id => !sectionWidgets.some(widget => widget.id === id)))
  }
}

function fmtBytes(bytes) {
  const value = Number(bytes || 0)
  if (value >= 1024 ** 4) return `${(value / 1024 ** 4).toFixed(1)} TB`
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(1)} GB`
  if (value >= 1024 ** 2) return `${(value / 1024 ** 2).toFixed(1)} MB`
  if (value >= 1024) return `${(value / 1024).toFixed(0)} KB`
  return `${value.toFixed(0)} B`
}

function fmtRate(bytes) {
  return `${fmtBytes(bytes)}/s`
}

function fmtPercent(value) {
  const numeric = Number(value || 0)
  return `${numeric.toFixed(numeric >= 10 ? 0 : 1)}%`
}

function fmtUptime(seconds) {
  const value = Number(seconds || 0)
  const days = Math.floor(value / 86400)
  const hours = Math.floor((value % 86400) / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function appendHistory(source = [], value, limit = 60) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return source.slice(-limit)
  return [...source, numeric].slice(-limit)
}

function lastFinite(values = []) {
  for (let index = values.length - 1; index >= 0; index -= 1) {
    const value = Number(values[index])
    if (Number.isFinite(value)) return value
  }
  return null
}

function previousFinite(values = []) {
  let found = 0
  for (let index = values.length - 1; index >= 0; index -= 1) {
    const value = Number(values[index])
    if (!Number.isFinite(value)) continue
    found += 1
    if (found === 2) return value
  }
  return null
}

function deriveDelta(values = [], options = {}) {
  const current = lastFinite(values)
  const previous = previousFinite(values)
  if (current === null || previous === null) {
    return { label: 'Collecting…', direction: 'neutral', tone: 'neutral' }
  }
  const rawChange = current - previous
  const baseline = Math.abs(previous) < 0.0001 ? 1 : Math.abs(previous)
  const pct = Math.abs((rawChange / baseline) * 100)
  const direction = rawChange > 0 ? 'up' : rawChange < 0 ? 'down' : 'neutral'
  const inverted = !!options.inverted
  let tone = 'neutral'
  if (direction !== 'neutral') {
    const improving = inverted ? rawChange < 0 : rawChange > 0
    tone = improving ? 'good' : 'bad'
  }
  return {
    label: `${direction === 'up' ? '▲' : direction === 'down' ? '▼' : '—'} ${pct.toFixed(pct >= 10 ? 0 : 1)}%`,
    direction,
    tone
  }
}

function thresholdTone(value, warn, crit) {
  if (value >= crit) return 'error'
  if (value >= warn) return 'warn'
  return 'ok'
}

function relativeTime(timestamp) {
  if (!timestamp) return 'just now'
  const delta = Math.max(0, Math.floor((Date.now() - Number(timestamp)) / 1000))
  if (delta < 60) return `${delta}s ago`
  if (delta < 3600) return `${Math.floor(delta / 60)}m ago`
  if (delta < 86400) return `${Math.floor(delta / 3600)}h ago`
  return `${Math.floor(delta / 86400)}d ago`
}

function formatTimestamp(value) {
  if (!value) return 'Unknown'
  return new Date(value).toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function healthCategoryScores(healthData) {
  const buckets = {
    security: { key: 'security', label: 'Security', total: 0, count: 0 },
    integrity: { key: 'integrity', label: 'Integrity', total: 0, count: 0 },
    availability: { key: 'availability', label: 'Availability', total: 0, count: 0 },
    performance: { key: 'performance', label: 'Performance', total: 0, count: 0 }
  }
  ;(healthData?.checks || []).forEach(check => {
    const name = String(check.name || '').toLowerCase()
    const group =
      /sudo|permission|apparmor|audit|network/.test(name) ? 'security' :
      /database|binary|file/.test(name) ? 'integrity' :
      /service|dependency/.test(name) ? 'availability' : 'performance'
    const value = check.status === 'healthy' ? 100 : check.status === 'warning' ? 60 : check.status === 'critical' ? 20 : 40
    buckets[group].total += value
    buckets[group].count += 1
  })
  return Object.values(buckets).map(bucket => ({
    ...bucket,
    score: bucket.count ? Math.round(bucket.total / bucket.count) : Number(healthData?.score || 0)
  }))
}

export default {
  name: 'DashboardPage',
  components: {
    PageHeader,
    AppButton,
    DetailDrawer,
    HealthRing,
    KPICard,
    TelemetryChart,
    ActivityFeed,
    ServiceHealthPanel,
    draggable
  },
  data() {
    const saved = normalizeDashboardState(safeLocalState())
    return {
      breadcrumbs: [{ text: 'Dashboard', active: true, icon: 'mdi mdi-view-dashboard' }],
      layoutEditMode: !!saved.layoutEditMode,
      activePreset: saved.activePreset || 'operator',
      auxRefreshSec: Number(saved.auxRefreshSec || 60),
      kpiWidgets: Array.isArray(saved.kpiWidgets) && saved.kpiWidgets.length ? saved.kpiWidgets : DEFAULT_KPI_WIDGETS,
      hiddenKpis: Array.isArray(saved.hiddenKpis) ? saved.hiddenKpis : [],
      sectionWidgets: Array.isArray(saved.sectionWidgets) && saved.sectionWidgets.length ? saved.sectionWidgets : DEFAULT_SECTION_WIDGETS,
      hiddenSections: Array.isArray(saved.hiddenSections) ? saved.hiddenSections : [],
      isRefreshing: false,
      isPulling: false,
      pullStartY: 0,
      pullDist: 0,
      isFullscreen: !!document.fullscreenElement,
      showWidgetCatalog: false,
      showHealthDrawer: false,
      showKpiDrawer: false,
      showCleaner: false,
      cleanerRunning: false,
      cleanerProgress: 0,
      cleanerTimer: null,
      selectedKpiId: '',
      healthFixResponse: null,
      healthFixMode: '',
      healthFixingName: '',
      userMeta: {},
      dockerInfo: { containers_running: 0, containers_total: 0 },
      secStats: { activeBans: 0, failedLogins: 0, ufwActive: false, services: [], securityScore: 0 },
      cleanupStats: { estimated_junk_human: '0 B', last_freed_human: '0 B', last_cleaned_at: null },
      loginAttempts: [],
      alerts: [],
      auditEntries: [],
      updates: { count: 0, last_updated: null },
      tasks: [],
      healthData: { overall_status: 'unknown', score: 0, checks: [], summary: '', timestamp: null, version: '2.1' },
      healthHistory: [],
      healthLoading: false,
      lastLoadedAt: 0,
      refreshTimer: null,
      persistLayoutTimer: null,
      derivedHistory: {
        activeBans: [],
        failedLogins: [],
        containersRunning: [],
        updatesPending: []
      }
    }
  },
  computed: {
    ...mapGetters('metrics', ['snap', 'cpuHistory', 'ramHistory', 'swapHistory', 'diskHistory', 'netRxHistory', 'netTxHistory', 'wsConnected', 'lastMetricTs']),
    presetOptions() {
      return [
        { value: 'operator', label: 'Operator' },
        { value: 'security', label: 'Security' },
        { value: 'performance', label: 'Performance' },
        { value: 'compact', label: 'Compact' }
      ]
    },
    telemetryRanges() {
      const enough = this.cpuHistory.filter(value => Number.isFinite(Number(value))).length >= 45
      return [
        { key: '1m', label: '1m', enabled: true },
        { key: '5m', label: '5m', enabled: enough },
        { key: '15m', label: '15m', enabled: false },
        { key: '1h', label: '1h', enabled: false }
      ]
    },
    lockCpuToPercent() {
      return false
    },
    endpointLabel() {
      return window.location.host || 'local'
    },
    metricFreshnessMs() {
      return this.lastMetricTs ? (Date.now() - this.lastMetricTs * 1000) : Number.POSITIVE_INFINITY
    },
    isMetricStale() {
      return this.metricFreshnessMs > 20000
    },
    isAuxStale() {
      if (!this.lastLoadedAt) return true
      return Date.now() - this.lastLoadedAt > this.auxRefreshSec * 2000
    },
    connectionTone() {
      if (!this.wsConnected) return 'error'
      if (this.isMetricStale) return 'warn'
      return 'ok'
    },
    connectionStateLabel() {
      if (!this.wsConnected) return 'Disconnected'
      if (this.isMetricStale) return 'Reconnecting'
      return 'Connected'
    },
    identityRows() {
      return [
        { label: 'Hostname', value: this.snap.hostname || 'Unavailable', mono: true },
        { label: 'Endpoint', value: this.endpointLabel, mono: true, copyValue: this.endpointLabel },
        { label: 'OS', value: this.prettyPlatformLabel(this.snap.os) || 'Unknown' },
        { label: 'Kernel', value: this.snap.kernel || 'Unknown', mono: true },
        { label: 'Uptime', value: fmtUptime(this.snap.uptime) },
        { label: 'Region', value: 'Self-hosted' },
        { label: 'Agent', value: `v${this.healthData.version || 'unknown'}`, mono: true },
        { label: 'Last tick', value: this.lastMetricTs ? this.formatLastTick(this.lastMetricTs * 1000) : 'Awaiting stream' }
      ]
    },
    backupTask() {
      return this.tasks.find(task => /backup/i.test(task.name || '') || /backup/i.test(task.command || '')) || null
    },
    idsService() {
      const services = Array.isArray(this.secStats.services) ? this.secStats.services : []
      return services.find(service => /crowdsec|psad/i.test(service.name || service.label || '')) || null
    },
    heroPills() {
      const idsIssue = !!(this.idsService && this.idsService.active_state !== 'active')
      return [
        {
          label: 'Firewall',
          value: this.secStats.ufwActive ? 'Active' : 'Inactive',
          tone: this.secStats.ufwActive ? 'ok' : 'error',
          marker: '●',
          route: '/firewall'
        },
        {
          label: 'IDS',
          value: idsIssue ? '1 stopped' : 'OK',
          tone: idsIssue ? 'warn' : 'ok',
          marker: idsIssue ? '⚠' : '●',
          route: idsIssue
            ? { path: '/services', query: { state: 'stopped', service: this.idsService?.name || 'crowdsec' } }
            : { path: '/services', query: { service: this.idsService?.name || 'crowdsec' } }
        },
        {
          label: 'Updates',
          value: this.updates.count ? `${this.updates.count} pending` : 'Current',
          tone: this.updates.count ? 'warn' : 'ok',
          marker: this.updates.count ? '⚠' : '●',
          route: '/updates'
        },
        {
          label: 'Backups',
          value: this.backupTask?.last_run?.started_at ? this.formatRelativeFromNow(new Date(this.backupTask.last_run.started_at).getTime()) : 'No recent run',
          tone: this.backupTask?.last_run?.started_at ? 'info' : 'warn',
          marker: this.backupTask?.last_run?.started_at ? '•' : '⚠',
          route: '/tasks'
        }
      ]
    },
    cpuTelemetrySeries() {
      return [{ name: 'CPU', data: this.cpuHistory, color: '#6ba8ff' }]
    },
    memoryTelemetrySeries() {
      return [
        { name: 'RAM', data: this.ramHistory, color: '#7c3aed' },
        { name: 'Swap', data: this.swapHistory, color: '#f3b54a' }
      ]
    },
    networkTelemetrySeries() {
      return [
        { name: 'Ingress', data: this.netRxHistory, color: '#6ba8ff' },
        { name: 'Egress', data: this.netTxHistory, color: '#3ad38a' }
      ]
    },
    cpuTelemetryThresholds() {
      return [
        { value: 70, label: 'Warn', color: '#f3b54a' },
        { value: 90, label: 'Crit', color: '#ff6a6a' }
      ]
    },
    memoryTelemetryThresholds() {
      return [
        { value: 80, label: 'Warn', color: '#f3b54a' },
        { value: 95, label: 'Crit', color: '#ff6a6a' }
      ]
    },
    mountRows() {
      return [...(this.snap.partitions || [])]
        .sort((left, right) => Number(right.pct || 0) - Number(left.pct || 0))
        .slice(0, 6)
        .map(partition => ({
          mount: partition.mount,
          pct: Number(partition.pct || 0).toFixed(0),
          used: fmtBytes(partition.used),
          total: fmtBytes(partition.total),
          device: partition.device || partition.fstype || 'disk',
          tone: thresholdTone(Number(partition.pct || 0), 80, 95)
        }))
    },
    activityItemsByTab() {
      const alerts = this.alerts.map(alert => ({
        id: `alert-${alert.id}`,
        type: 'alert',
        severity: alert.severity === 'critical' || alert.severity === 'emergency' ? 'critical' : alert.severity === 'warning' ? 'warning' : 'info',
        icon: alert.severity === 'critical' || alert.severity === 'emergency' ? 'mdi mdi-alert-octagon' : alert.severity === 'warning' ? 'mdi mdi-alert' : 'mdi mdi-information',
        summary: `${alert.message || 'Alert'} · ${alert.source || 'system'}`,
        meta: `Alerts · ${alert.severity || 'info'}`,
        ts: (alert.ts || 0) * 1000,
        route: '/alerts'
      }))
      const logins = this.loginAttempts.map((attempt, index) => ({
        id: `login-${index}-${attempt.ts || 0}`,
        type: 'login',
        severity: attempt.success ? 'info' : 'warning',
        icon: attempt.success ? 'mdi mdi-login-variant' : 'mdi mdi-lock-alert-outline',
        summary: `${attempt.success ? 'Successful' : 'Failed'} login · ${attempt.username || 'unknown user'}`,
        meta: `${attempt.ip || 'unknown IP'} · ${attempt.success ? 'Accepted' : 'Rejected'}`,
        ts: (attempt.ts || 0) * 1000,
        route: '/audit-logs'
      }))
      const audit = this.auditEntries.map((entry, index) => ({
        id: `audit-${entry.id || index}`,
        type: 'audit',
        severity: entry.success ? 'info' : 'warning',
        icon: entry.success ? 'mdi mdi-shield-check-outline' : 'mdi mdi-shield-alert-outline',
        summary: `${entry.username || 'Unknown user'} · ${entry.reason || 'event'}`,
        meta: `${entry.ip || 'unknown IP'} · ${entry.success ? 'Success' : 'Failure'}`,
        ts: (entry.ts || 0) * 1000,
        route: '/audit-logs'
      }))
      const system = []
      if (this.cleanupStats.last_cleaned_at) {
        system.push({
          id: 'system-cleaner',
          type: 'system',
          severity: 'info',
          icon: 'mdi mdi-broom',
          summary: `Maintenance cleanup completed · ${this.cleanupStats.last_freed_human || '0 B'} reclaimed`,
          meta: 'System · Maintenance',
          ts: new Date(this.cleanupStats.last_cleaned_at).getTime(),
          route: '/dashboard'
        })
      }
      if (this.updates.count) {
        system.push({
          id: 'system-updates',
          type: 'system',
          severity: this.updates.count > 10 ? 'warning' : 'info',
          icon: 'mdi mdi-package-up',
          summary: `${this.updates.count} updates pending`,
          meta: 'System · Packages',
          ts: this.updates.last_updated ? new Date(this.updates.last_updated).getTime() : Date.now(),
          route: '/updates'
        })
      }
      const all = [...alerts, ...logins, ...audit, ...system].sort((left, right) => right.ts - left.ts)
      return { all, alerts, logins, audit, system }
    },
    healthCategories() {
      return healthCategoryScores(this.healthData)
    },
    healthIssues() {
      return [...(this.healthData.checks || [])].filter(check => check.status !== 'healthy')
    },
    healthHeadline() {
      const score = Number(this.healthData.score || 0)
      if (score < 35) return 'Critical posture'
      if (score < 50) return 'Degraded posture'
      if (score < 80) return 'Healthy with issues'
      return 'Operationally healthy'
    },
    selectedKpiDetail() {
      return this.selectedKpiId ? this.kpiCardById(this.selectedKpiId) : null
    },
    hiddenKpiEntries() {
      return this.hiddenKpis.map(id => this.kpiCatalog(id)).filter(Boolean)
    },
    hiddenSectionEntries() {
      return this.hiddenSections.map(id => this.sectionCatalog(id)).filter(Boolean)
    }
  },
  watch: {
    auxRefreshSec() {
      this.scheduleRefreshTimer()
      this.persistDashboardState()
    }
  },
  async mounted() {
    document.addEventListener('fullscreenchange', this.onFullscreenChange)
    this.$store.dispatch('metrics/startLive')
    await this.loadDashboardState()
    this.scheduleRefreshTimer()
    await this.loadAll()
    this.registerPullToRefresh()
  },
  beforeUnmount() {
    document.removeEventListener('fullscreenchange', this.onFullscreenChange)
    this.unregisterPullToRefresh()
    clearInterval(this.refreshTimer)
    if (this.cleanerTimer) {
      clearInterval(this.cleanerTimer)
      this.cleanerTimer = null
    }
    if (this.persistLayoutTimer) {
      clearTimeout(this.persistLayoutTimer)
      this.persistLayoutTimer = null
    }
  },
  methods: {
    formatRelativeFromNow(timestamp) {
      return relativeTime(timestamp)
    },
    formatLastTick(timestamp) {
      const base = formatTimestamp(timestamp)
      try {
        const zone = new Intl.DateTimeFormat(undefined, { timeZoneName: 'shortOffset' })
          .formatToParts(new Date(timestamp))
          .find(part => part.type === 'timeZoneName')?.value
        return zone ? `${base} ${zone}` : base
      } catch {
        return base
      }
    },
    prettyPlatformLabel(value) {
      if (!value) return ''
      if (String(value).toLowerCase() === 'linux') return 'Linux'
      return String(value).replace(/\b\w/g, char => char.toUpperCase())
    },
    formatTimestamp,
    formatPercentValue(value) {
      return fmtPercent(value)
    },
    formatRateValue(value) {
      return fmtRate(value)
    },
    buildDashboardStatePayload() {
      return {
        layoutEditMode: this.layoutEditMode,
        activePreset: this.activePreset,
        auxRefreshSec: this.auxRefreshSec,
        kpiWidgets: this.kpiWidgets,
        hiddenKpis: this.hiddenKpis,
        sectionWidgets: this.sectionWidgets,
        hiddenSections: this.hiddenSections
      }
    },
    async loadDashboardState() {
      const fallback = normalizeDashboardState(safeLocalState())
      if (!this.$store.getters['auth/loggedIn']) {
        Object.assign(this, fallback)
        return
      }
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getDashboardLayout()
        const normalized = normalizeDashboardState(data, fallback)
        Object.assign(this, normalized)
        localStorage.setItem(DASHBOARD_STATE_KEY, JSON.stringify(normalized))
      } catch {
        Object.assign(this, fallback)
      }
    },
    scheduleDashboardPersist(payload) {
      if (this.persistLayoutTimer) {
        clearTimeout(this.persistLayoutTimer)
      }
      this.persistLayoutTimer = window.setTimeout(() => {
        this.saveDashboardState(payload)
      }, 250)
    },
    async saveDashboardState(payload) {
      if (!this.$store.getters['auth/loggedIn']) return
      try {
        const api = (await import('@/services/api')).default
        await api.saveDashboardLayout(payload)
      } catch {
        // Keep local fallback even when roaming persistence fails.
      }
    },
    persistDashboardState() {
      const payload = this.buildDashboardStatePayload()
      localStorage.setItem(DASHBOARD_STATE_KEY, JSON.stringify(payload))
      this.scheduleDashboardPersist(payload)
    },
    applyPreset(presetKey) {
      const preset = PRESETS[presetKey] || PRESETS.operator
      this.activePreset = presetKey
      this.kpiWidgets = preset.kpis.map(id => ({ id }))
      this.hiddenKpis = DEFAULT_KPI_WIDGETS.map(item => item.id).filter(id => !preset.kpis.includes(id))
      this.sectionWidgets = preset.sections.map(id => ({ id }))
      this.hiddenSections = DEFAULT_SECTION_WIDGETS.map(item => item.id).filter(id => !preset.sections.includes(id))
      this.persistDashboardState()
    },
    toggleLayoutEdit() {
      this.layoutEditMode = !this.layoutEditMode
      this.persistDashboardState()
    },
    async toggleFullscreen() {
      if (document.fullscreenElement) {
        await document.exitFullscreen().catch(() => {})
      } else {
        await this.$el?.requestFullscreen?.().catch(() => {})
      }
    },
    onFullscreenChange() {
      this.isFullscreen = !!document.fullscreenElement
    },
    scheduleRefreshTimer() {
      clearInterval(this.refreshTimer)
      this.refreshTimer = setInterval(() => {
        this.loadAll()
      }, this.auxRefreshSec * 1000)
    },
    async loadAll() {
      if (!this.$store.getters['auth/loggedIn']) return
      this.healthLoading = !this.lastLoadedAt
      try {
        const api = (await import('@/services/api')).default
        const [health, docker, secStatus, logins, cleanup, alerts, audit, updates, tasks, me] = await Promise.allSettled([
          api.getHealth(),
          api.getDockerInfo(),
          api.getSecurityStatus(),
          api.getDashboardLoginAttempts(),
          api.getCleanupStats(),
          api.getAlerts({ limit: 8 }),
          api.getAuditLogs({ limit: 12 }),
          api.getUpdates(),
          api.getTasks(),
          api.getMe()
        ])

        if (health.status === 'fulfilled') {
          this.healthData = health.value.data || this.healthData
          this.healthHistory = appendHistory(this.healthHistory, this.healthData.score, 30)
        }
        if (docker.status === 'fulfilled') {
          this.dockerInfo = docker.value.data || this.dockerInfo
          this.derivedHistory.containersRunning = appendHistory(this.derivedHistory.containersRunning, this.dockerInfo.containers_running || 0, 30)
        }
        if (secStatus.status === 'fulfilled') {
          const data = secStatus.value.data || {}
          this.secStats = {
            activeBans: data.active_bans || 0,
            failedLogins: data.failed_logins || 0,
            ufwActive: !!data.ufw_active,
            services: Array.isArray(data.services) ? data.services : [],
            securityScore: data.security_score || 0
          }
          this.derivedHistory.activeBans = appendHistory(this.derivedHistory.activeBans, this.secStats.activeBans, 30)
          this.derivedHistory.failedLogins = appendHistory(this.derivedHistory.failedLogins, this.secStats.failedLogins, 30)
        }
        if (logins.status === 'fulfilled') this.loginAttempts = Array.isArray(logins.value.data) ? logins.value.data : []
        if (cleanup.status === 'fulfilled') this.cleanupStats = cleanup.value.data || this.cleanupStats
        if (alerts.status === 'fulfilled') {
          const data = alerts.value.data
          this.alerts = Array.isArray(data) ? data : (data?.alerts || [])
        }
        if (audit.status === 'fulfilled') {
          const data = audit.value.data
          this.auditEntries = Array.isArray(data) ? data : (data?.logs || [])
        }
        if (updates.status === 'fulfilled') {
          this.updates = updates.value.data || this.updates
          this.derivedHistory.updatesPending = appendHistory(this.derivedHistory.updatesPending, this.updates.count || 0, 30)
        }
        if (tasks.status === 'fulfilled') {
          const data = tasks.value.data
          this.tasks = Array.isArray(data) ? data : (data?.tasks || [])
        }
        if (me.status === 'fulfilled') {
          this.userMeta = me.value.data || {}
        }
        this.lastLoadedAt = Date.now()
      } finally {
        this.healthLoading = false
      }
    },
    async refreshAll() {
      if (this.isRefreshing) return
      this.isRefreshing = true
      try {
        const api = (await import('@/services/api')).default
        const metrics = await api.getMetrics()
        this.$store.commit('metrics/SET_SNAP', metrics.data)
        await this.loadAll()
      } finally {
        this.isRefreshing = false
      }
    },
    kpiCatalog(id) {
      return {
        cpu: { id: 'cpu', label: 'CPU usage', description: 'Processor saturation, delta, and load average.' },
        memory: { id: 'memory', label: 'Memory', description: 'RAM pressure with swap context.' },
        disk: { id: 'disk', label: 'Disk root', description: 'Root filesystem utilization and free space.' },
        network: { id: 'network', label: 'Network I/O', description: 'Ingress and egress throughput.' },
        bans: { id: 'bans', label: 'Active bans', description: 'fail2ban and CrowdSec pressure.' },
        logins24h: { id: 'logins24h', label: 'Failed logins', description: 'Authentication failures over the last 24 hours.' },
        containers: { id: 'containers', label: 'Containers', description: 'Running containers versus total footprint.' },
        uptime: { id: 'uptime', label: 'Uptime', description: 'Host uptime and last-restart context.' }
      }[id] || null
    },
    sectionCatalog(id) {
      return {
        telemetry: { id: 'telemetry', label: 'Telemetry zone', description: 'Charts and mount usage.' },
        services: { id: 'services', label: 'Service health', description: 'Full-width service grid with actions.' },
        activity: { id: 'activity', label: 'Activity feed', description: 'Alerts, logins, audit, and system events.' }
      }[id] || null
    },
    hideKpi(id) {
      this.kpiWidgets = this.kpiWidgets.filter(widget => widget.id !== id)
      this.hiddenKpis = [...new Set([...this.hiddenKpis, id])]
      this.persistDashboardState()
    },
    restoreKpi(id) {
      if (!this.kpiWidgets.some(widget => widget.id === id)) {
        this.kpiWidgets = [...this.kpiWidgets, { id }]
      }
      this.hiddenKpis = this.hiddenKpis.filter(entry => entry !== id)
      this.persistDashboardState()
    },
    hideSection(id) {
      this.sectionWidgets = this.sectionWidgets.filter(widget => widget.id !== id)
      this.hiddenSections = [...new Set([...this.hiddenSections, id])]
      this.persistDashboardState()
    },
    restoreSection(id) {
      if (!this.sectionWidgets.some(widget => widget.id === id)) {
        this.sectionWidgets = [...this.sectionWidgets, { id }]
      }
      this.hiddenSections = this.hiddenSections.filter(entry => entry !== id)
      this.persistDashboardState()
    },
    kpiCardById(id) {
      const counts = this.dockerInfo.containers_total ? `${this.dockerInfo.containers_running}/${this.dockerInfo.containers_total}` : '0/0'
      const cpuDelta = deriveDelta(this.cpuHistory)
      const memoryDelta = deriveDelta(this.ramHistory, { inverted: false })
      const diskDelta = deriveDelta(this.diskHistory)
      const networkCombined = this.netRxHistory.map((value, index) => Number(value || 0) + Number(this.netTxHistory[index] || 0))
      const networkDelta = deriveDelta(networkCombined)
      const bansDelta = deriveDelta(this.derivedHistory.activeBans, { inverted: true })
      const loginDelta = deriveDelta(this.derivedHistory.failedLogins, { inverted: true })
      const containerDelta = deriveDelta(this.derivedHistory.containersRunning)

      const cards = {
        cpu: {
          label: 'CPU Usage',
          icon: 'mdi mdi-chip',
          value: fmtPercent(this.snap.cpu_pct),
          deltaLabel: cpuDelta.label,
          deltaDirection: cpuDelta.direction,
          deltaTone: cpuDelta.tone,
          sparkline: this.cpuHistory,
          contextLines: [
            `Load ${Number(this.snap.load1 || 0).toFixed(2)} · ${Number(this.snap.load5 || 0).toFixed(2)} · ${Number(this.snap.load15 || 0).toFixed(2)}`,
            `Updated ${this.formatRelativeFromNow(this.lastMetricTs * 1000)}`
          ],
          threshold: { value: Number(this.snap.cpu_pct || 0), warn: 70, crit: 90, max: 100 },
          live: this.wsConnected,
          stale: this.isMetricStale,
          rangeLabel: '1m live',
          tone: thresholdTone(Number(this.snap.cpu_pct || 0), 70, 90),
          statusLabel: this.isMetricStale ? 'Stale' : ''
        },
        memory: {
          label: 'Memory',
          icon: 'mdi mdi-memory',
          value: fmtPercent(this.snap.ram_pct),
          deltaLabel: memoryDelta.label,
          deltaDirection: memoryDelta.direction,
          deltaTone: memoryDelta.tone,
          sparkline: this.ramHistory,
          sparklineSecondary: this.swapHistory,
          contextLines: [
            `${fmtBytes(this.snap.ram_used)} / ${fmtBytes(this.snap.ram_total)} · swap ${fmtPercent(this.snap.swap_pct)}`,
            `${fmtBytes(this.snap.swap_used)} / ${fmtBytes(this.snap.swap_total)}`
          ],
          threshold: { value: Number(this.snap.ram_pct || 0), warn: 80, crit: 95, max: 100 },
          live: this.wsConnected,
          stale: this.isMetricStale,
          rangeLabel: '1m live',
          tone: thresholdTone(Number(this.snap.ram_pct || 0), 80, 95)
        },
        disk: {
          label: 'Disk (Root)',
          icon: 'mdi mdi-harddisk',
          value: fmtPercent(this.snap.disk_pct),
          deltaLabel: diskDelta.label,
          deltaDirection: diskDelta.direction,
          deltaTone: diskDelta.tone,
          sparkline: this.diskHistory,
          contextLines: [
            `${fmtBytes(this.snap.disk_used)} / ${fmtBytes(this.snap.disk_total)}`,
            `${fmtBytes(this.snap.disk_free)} free on /`
          ],
          threshold: { value: Number(this.snap.disk_pct || 0), warn: 80, crit: 95, max: 100 },
          live: this.wsConnected,
          stale: this.isMetricStale,
          rangeLabel: '1m live',
          tone: thresholdTone(Number(this.snap.disk_pct || 0), 80, 95)
        },
        network: {
          label: 'Network I/O',
          icon: 'mdi mdi-swap-vertical',
          value: fmtRate(Number(this.snap.net_rx_rate || 0) + Number(this.snap.net_tx_rate || 0)),
          deltaLabel: networkDelta.label,
          deltaDirection: networkDelta.direction,
          deltaTone: networkDelta.tone,
          sparkline: this.netRxHistory,
          sparklineSecondary: this.netTxHistory,
          contextLines: [
            `in ${fmtRate(this.snap.net_rx_rate)} · out ${fmtRate(this.snap.net_tx_rate)}`,
            `${fmtBytes(this.snap.net_rx_total)} rx · ${fmtBytes(this.snap.net_tx_total)} tx`
          ],
          live: this.wsConnected,
          stale: this.isMetricStale,
          rangeLabel: '1m live',
          tone: 'default',
          sparkColor: 'var(--dashboard-spark-line-alt)'
        },
        bans: {
          label: 'Active Bans',
          icon: 'mdi mdi-shield-lock-outline',
          value: this.secStats.activeBans,
          deltaLabel: bansDelta.label,
          deltaDirection: bansDelta.direction,
          deltaTone: bansDelta.tone,
          sparkline: this.derivedHistory.activeBans,
          contextLines: [
            'fail2ban + CrowdSec pressure',
            `${this.secStats.ufwActive ? 'Firewall active' : 'Firewall inactive'}`
          ],
          threshold: { value: Number(this.secStats.activeBans || 0), warn: 5, crit: 10, max: 15 },
          live: true,
          stale: this.isAuxStale,
          rangeLabel: '24h window',
          tone: thresholdTone(Number(this.secStats.activeBans || 0), 5, 10)
        },
        logins24h: {
          label: 'Failed Logins',
          icon: 'mdi mdi-lock-alert-outline',
          value: this.secStats.failedLogins,
          deltaLabel: loginDelta.label,
          deltaDirection: loginDelta.direction,
          deltaTone: loginDelta.tone,
          sparkline: this.derivedHistory.failedLogins,
          contextLines: [
            `Last attempt ${this.loginAttempts[0]?.ts ? this.formatRelativeFromNow(this.loginAttempts[0].ts * 1000) : 'unknown'}`,
            '24h aggregate across all auth sources'
          ],
          threshold: { value: Number(this.secStats.failedLogins || 0), warn: 10, crit: 50, max: 60 },
          live: true,
          stale: this.isAuxStale,
          rangeLabel: '24h window',
          tone: thresholdTone(Number(this.secStats.failedLogins || 0), 10, 50)
        },
        containers: {
          label: 'Containers',
          icon: 'mdi mdi-docker',
          value: counts,
          deltaLabel: containerDelta.label,
          deltaDirection: containerDelta.direction,
          deltaTone: containerDelta.tone,
          sparkline: this.derivedHistory.containersRunning,
          contextLines: [
            `${this.dockerInfo.containers_running || 0} running · ${this.dockerInfo.containers_total || 0} total`,
            `Updates ${this.updates.count || 0} pending`
          ],
          live: true,
          stale: this.isAuxStale,
          rangeLabel: 'service poll',
          tone: this.dockerInfo.containers_running < this.dockerInfo.containers_total ? 'warn' : 'ok'
        },
        uptime: {
          label: 'Uptime',
          icon: 'mdi mdi-timer-outline',
          value: fmtUptime(this.snap.uptime),
          deltaLabel: '— stable',
          deltaDirection: 'neutral',
          deltaTone: 'neutral',
          sparkline: [],
          contextLines: [
            `Host ${this.snap.hostname || 'node'} · kernel ${this.snap.kernel || 'unknown'}`,
            `Last sync ${this.formatRelativeFromNow(this.lastLoadedAt)}`
          ],
          live: this.wsConnected,
          stale: this.isMetricStale,
          rangeLabel: 'host lifetime',
          tone: 'ok'
        }
      }
      return cards[id] || null
    },
    openKpiDrawer(id) {
      this.selectedKpiId = id
      this.showKpiDrawer = true
    },
    async copyIdentityValue(row) {
      if (!row?.copyValue || !navigator.clipboard) return
      try {
        await navigator.clipboard.writeText(row.copyValue)
      } catch {
        // Ignore clipboard failures.
      }
    },
    navigateTo(route) {
      if (!route) return
      this.$router.push(route)
    },
    runScanAction() {
      this.$router.push('/security-tools')
    },
    async reloadServicesPanel() {
      const panelRef = Array.isArray(this.$refs.servicePanel) ? this.$refs.servicePanel[0] : this.$refs.servicePanel
      await panelRef?.refreshServices?.()
      await this.loadAll()
    },
    openActivityItem(item) {
      this.navigateTo(item.route)
    },
    openHealthIssue(check) {
      this.showHealthDrawer = true
      this.healthFixResponse = null
      this.$nextTick(() => {
        const target = Array.from(this.$el.querySelectorAll('.dashboard-health-issue-card')).find(card => card.textContent.includes(check.name))
        target?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
      })
    },
    async fixHealthIssue(check, mode) {
      this.healthFixMode = mode
      this.healthFixingName = check.name
      this.healthFixResponse = null
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.fixHealthIssue({ check_name: check.name, action: mode })
        this.healthFixResponse = data
        await this.loadAll()
      } catch (error) {
        this.healthFixResponse = {
          success: false,
          message: error.response?.data?.error || error.message || 'Fix request failed',
          remedy: 'Open the Services or Security surfaces for manual remediation.'
        }
      } finally {
        this.healthFixMode = ''
        this.healthFixingName = ''
      }
    },
    registerPullToRefresh() {
      const el = this.$refs.pageEl
      if (!el) return
      el.addEventListener('touchstart', this.onTouchStart, { passive: true })
      el.addEventListener('touchmove', this.onTouchMove, { passive: true })
      el.addEventListener('touchend', this.onTouchEnd, { passive: true })
    },
    unregisterPullToRefresh() {
      const el = this.$refs.pageEl
      if (!el) return
      el.removeEventListener('touchstart', this.onTouchStart)
      el.removeEventListener('touchmove', this.onTouchMove)
      el.removeEventListener('touchend', this.onTouchEnd)
    },
    onTouchStart(event) {
      const scrollEl = document.querySelector('.page-content') || window
      const scrollTop = scrollEl === window ? window.scrollY : scrollEl.scrollTop
      if (scrollTop > 0) return
      this.pullStartY = event.touches[0].clientY
      this.isPulling = true
    },
    onTouchMove(event) {
      if (!this.isPulling) return
      const delta = event.touches[0].clientY - this.pullStartY
      this.pullDist = delta > 0 ? delta : 0
    },
    async onTouchEnd() {
      if (!this.isPulling) return
      this.isPulling = false
      if (this.pullDist >= 64) {
        this.pullDist = 0
        await this.refreshAll()
      } else {
        this.pullDist = 0
      }
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
        this.cleanerProgress = Math.min(98, this.cleanerProgress + Math.floor(Math.random() * 12 + 4))
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
.dashboard-page {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  gap: 18px;
  padding-inline: clamp(2px, 0.35vw, 6px);
  padding-bottom: 18px;
  background:
    radial-gradient(circle at top left, rgba(107, 168, 255, 0.08), transparent 32%),
    linear-gradient(180deg, transparent, rgba(107, 168, 255, 0.02));
}

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

.dashboard-header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
}

.dashboard-select-wrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 999px;
  border: 1px solid var(--dashboard-panel-border);
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.dashboard-select {
  border: 0;
  background: transparent;
  color: var(--text-primary);
  font-size: 12px;
}

.dashboard-select-wrap--compact {
  min-width: 118px;
}

.dashboard-refresh-block {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.dashboard-refresh-note {
  color: var(--text-tertiary);
  font-size: 12px;
  white-space: nowrap;
}

.dashboard-hero-grid {
  display: grid;
  grid-template-columns: minmax(320px, 1.25fr) minmax(280px, 1fr) minmax(320px, 1.1fr);
  gap: 18px;
}

.dashboard-panel {
  border-radius: 22px;
  border: 1px solid var(--dashboard-panel-border);
  background: var(--dashboard-panel-bg);
  box-shadow: var(--shadow-md);
  padding: 18px;
  font-variant-numeric: tabular-nums;
}

.dashboard-panel__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.dashboard-panel__eyebrow,
.dashboard-footer-strip__label {
  color: var(--text-tertiary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 700;
}

.dashboard-panel__title {
  margin: 4px 0 0;
  font-size: 18px;
  color: var(--text-primary);
}

.dashboard-panel__hint {
  color: var(--text-secondary);
  font-size: 12px;
}

.dashboard-live-chip,
.dashboard-connection-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.dashboard-live-chip {
  background: var(--state-ok-bg);
  color: var(--state-ok-fg);
}

.dashboard-live-chip.is-offline {
  background: var(--state-warn-bg);
  color: var(--state-warn-fg);
}

.dashboard-live-chip__dot,
.dashboard-connection-pill::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: currentColor;
}

.dashboard-identity-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
}

.dashboard-identity-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dashboard-identity-row__label,
.dashboard-cleaner-card__copy,
.dashboard-empty-inline,
.dashboard-kpi-drawer__line,
.dashboard-health-issue-card p,
.dashboard-health-issue-card small,
.dashboard-mount-row__meta {
  color: var(--text-secondary);
  font-size: 12px;
}

.dashboard-identity-row__value {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.dashboard-identity-row__copy {
  width: 22px;
  height: 22px;
  display: inline-grid;
  place-items: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-tertiary);
  opacity: 0;
  transition: opacity 0.16s ease, background 0.16s ease, color 0.16s ease;
}

.dashboard-identity-row:hover .dashboard-identity-row__copy,
.dashboard-identity-row__copy:focus-visible {
  opacity: 1;
}

.dashboard-identity-row__copy:hover,
.dashboard-identity-row__copy:focus-visible {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-primary);
}

.dashboard-identity-row__value.is-mono,
.dashboard-footer-strip__value--mono {
  font-family: var(--font-family-monospace);
}

.dashboard-hero-side {
  display: flex;
  flex-direction: column;
}

.dashboard-status-pills {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.dashboard-status-pill {
  border: 1px solid var(--dashboard-panel-border);
  background: rgba(255, 255, 255, 0.03);
  border-radius: 16px;
  padding: 12px;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 6px;
  cursor: pointer;
  transition: transform 0.16s ease, background 0.16s ease, border-color 0.16s ease;
}

.dashboard-status-pill:hover,
.dashboard-status-pill:focus-visible {
  transform: translateY(-1px);
  background: rgba(255, 255, 255, 0.05);
  border-color: color-mix(in srgb, var(--accent) 26%, var(--dashboard-panel-border));
}

.dashboard-status-pill__label {
  color: var(--text-tertiary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-status-pill__value {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.dashboard-status-pill__marker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 12px;
  font-size: 13px;
}

.dashboard-status-pill__marker.is-ok {
  color: var(--dashboard-threshold-ok);
}

.dashboard-status-pill__value.is-ok {
  color: var(--dashboard-threshold-ok);
}

.dashboard-status-pill__marker.is-warn {
  color: var(--dashboard-threshold-warn);
}

.dashboard-status-pill__value.is-warn {
  color: var(--dashboard-threshold-warn);
}

.dashboard-status-pill__marker.is-error {
  color: var(--dashboard-threshold-crit);
}

.dashboard-status-pill__value.is-error {
  color: var(--dashboard-threshold-crit);
}

.dashboard-status-pill__marker.is-info {
  color: var(--text-secondary);
}

.dashboard-status-pill__value.is-info {
  color: var(--text-secondary);
}

.dashboard-status-pill--ok {
  box-shadow: var(--dashboard-glow-ok);
}

.dashboard-status-pill--warn {
  box-shadow: var(--dashboard-glow-warn);
}

.dashboard-status-pill--error {
  box-shadow: var(--dashboard-glow-error);
}

.dashboard-status-pill--info {
  box-shadow: var(--shadow-sm);
}

.dashboard-actions-grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.dashboard-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.dashboard-kpi-grid__item,
.dashboard-section-shell {
  position: relative;
}

.dashboard-edit-handle,
.dashboard-section-handle,
.dashboard-hide-button {
  position: absolute;
  z-index: 5;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  border: 1px solid var(--dashboard-panel-border);
  background: rgba(8, 15, 27, 0.82);
  color: var(--text-secondary);
}

.dashboard-edit-handle,
.dashboard-section-handle {
  top: 8px;
  right: 48px;
  cursor: grab;
}

.dashboard-hide-button {
  top: 8px;
  right: 8px;
}

.dashboard-section-stack {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.dashboard-section-tools {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 4;
  display: flex;
  gap: 8px;
}

.dashboard-telemetry-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.dashboard-mount-card {
  min-height: 100%;
}

.dashboard-mount-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dashboard-mount-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dashboard-mount-row__head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.dashboard-mount-row__mount,
.dashboard-mount-row__value,
.dashboard-footer-strip__value,
.dashboard-kpi-drawer__value,
.dashboard-health-category strong {
  color: var(--text-primary);
  font-weight: 600;
}

.dashboard-mount-row__bar,
.dashboard-cleaner-progress {
  height: 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  overflow: hidden;
}

.dashboard-mount-row__fill,
.dashboard-cleaner-progress__fill {
  display: block;
  height: 100%;
  border-radius: inherit;
}

.dashboard-mount-row__fill.ok {
  background: var(--dashboard-threshold-ok);
}

.dashboard-mount-row__fill.warn {
  background: var(--dashboard-threshold-warn);
}

.dashboard-mount-row__fill.error {
  background: var(--dashboard-threshold-error);
}

.dashboard-footer-strip {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  align-items: center;
}

.dashboard-footer-strip__group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dashboard-connection-pill--ok {
  background: var(--state-ok-bg);
  color: var(--state-ok-fg);
}

.dashboard-connection-pill--warn {
  background: var(--state-warn-bg);
  color: var(--state-warn-fg);
}

.dashboard-connection-pill--error {
  background: var(--state-error-bg);
  color: var(--state-error-fg);
}

.dashboard-drawer-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.dashboard-drawer-title {
  margin: 0 0 12px;
  color: var(--text-primary);
  font-size: 15px;
}

.dashboard-restore-list,
.dashboard-health-drawer,
.dashboard-kpi-drawer {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dashboard-restore-row,
.dashboard-health-issue-card {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 14px;
  border-radius: 16px;
  border: 1px solid var(--dashboard-panel-border);
  background: rgba(255, 255, 255, 0.03);
  text-align: left;
}

.dashboard-restore-row strong,
.dashboard-health-issue-card strong,
.dashboard-health-drawer__summary,
.dashboard-kpi-drawer__status,
.dashboard-kpi-drawer__delta,
.dashboard-cleaner-stats span {
  color: var(--text-primary);
}

.dashboard-restore-row small {
  display: block;
  margin-top: 4px;
  color: var(--text-secondary);
}

.dashboard-health-drawer__categories,
.dashboard-health-issue-card__actions,
.dashboard-cleaner-stats,
.dashboard-cleaner-card__actions,
.dashboard-health-issue-card__title,
.dashboard-kpi-drawer__hero,
.dashboard-kpi-drawer__thresholds {
  display: flex;
  gap: 10px;
  justify-content: space-between;
  align-items: flex-start;
}

.dashboard-health-category {
  flex: 1 1 0;
  padding: 12px 14px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--dashboard-panel-border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dashboard-health-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-health-badge.critical {
  background: var(--state-error-bg);
  color: var(--state-error-fg);
}

.dashboard-health-badge.warning {
  background: var(--state-warn-bg);
  color: var(--state-warn-fg);
}

.dashboard-health-response {
  padding: 14px;
  border-radius: 16px;
}

.dashboard-health-response.is-success {
  background: var(--state-ok-bg);
}

.dashboard-health-response.is-error {
  background: var(--state-error-bg);
}

.dashboard-health-response pre {
  margin: 10px 0 0;
  padding: 10px;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.22);
  color: var(--text-primary);
  white-space: pre-wrap;
}

.dashboard-kpi-drawer__value {
  font-size: 30px;
  line-height: 1;
}

.dashboard-kpi-drawer__delta.is-good {
  color: var(--state-ok-fg);
}

.dashboard-kpi-drawer__delta.is-bad {
  color: var(--state-error-fg);
}

.dashboard-kpi-drawer__delta.is-neutral {
  color: var(--text-secondary);
}

.dashboard-cleaner-overlay {
  position: fixed;
  inset: 0;
  z-index: 1050;
  display: grid;
  place-items: center;
  background: rgba(8, 14, 24, 0.72);
  backdrop-filter: blur(10px);
}

.dashboard-cleaner-card {
  position: relative;
  width: min(92vw, 520px);
  border-radius: 24px;
  border: 1px solid var(--dashboard-panel-border-strong);
  background: var(--dashboard-panel-bg-strong);
  box-shadow: var(--shadow-lg);
  padding: 20px;
}

.dashboard-cleaner-card__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.dashboard-cleaner-stats {
  margin-top: 10px;
}

.dashboard-cleaner-progress__fill {
  background: var(--dashboard-threshold-ok);
}

.drag-ghost {
  opacity: 0.42;
}

.drag-chosen {
  box-shadow: var(--shadow-lg), var(--shadow-glow-accent);
}

@media (max-width: 1279px) {
  .dashboard-hero-grid,
  .dashboard-kpi-grid,
  .dashboard-telemetry-grid,
  .dashboard-footer-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 960px) {
  .dashboard-hero-grid,
  .dashboard-kpi-grid,
  .dashboard-telemetry-grid,
  .dashboard-drawer-grid,
  .dashboard-footer-strip,
  .dashboard-identity-grid,
  .dashboard-actions-grid,
  .dashboard-status-pills {
    grid-template-columns: 1fr;
  }

  .dashboard-header-actions,
  .dashboard-refresh-block {
    width: 100%;
    justify-content: flex-start;
  }

  .dashboard-health-drawer__categories,
  .dashboard-health-issue-card,
  .dashboard-kpi-drawer__hero,
  .dashboard-kpi-drawer__thresholds {
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .dashboard-page {
    gap: 14px;
    padding-inline: 0;
  }

  .dashboard-panel,
  .dashboard-cleaner-card {
    padding: 16px;
  }
}
</style>