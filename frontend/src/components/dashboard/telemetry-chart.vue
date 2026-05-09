<template>
  <article class="telemetry-card sc-focus-ring">
    <div class="telemetry-card__header">
      <div class="telemetry-card__title-wrap">
        <div class="telemetry-card__icon" aria-hidden="true">
          <i :class="icon"></i>
        </div>
        <div>
          <div class="telemetry-card__eyebrow">Live Telemetry</div>
          <h3 class="telemetry-card__title">{{ title }}</h3>
          <p v-if="description" class="telemetry-card__description">{{ description }}</p>
        </div>
      </div>
      <div class="telemetry-card__controls">
        <div class="telemetry-card__range-group" role="tablist" :aria-label="`${title} range`">
          <button
            v-for="option in normalizedRanges"
            :key="option.key"
            type="button"
            class="telemetry-card__range"
            :class="{ 'is-active': option.key === selectedRange, 'is-disabled': !option.enabled }"
            :disabled="!option.enabled"
            @click="selectedRange = option.key"
          >
            {{ option.label }}
          </button>
        </div>
        <button type="button" class="telemetry-card__toggle" @click="lockScale = !lockScale">
          {{ lockScale ? 'Lock 0–100' : 'Auto scale' }}
        </button>
        <button type="button" class="telemetry-card__toggle telemetry-card__collapse" @click="isCollapsed = !isCollapsed"><i class="mdi" :class="isCollapsed ? 'mdi-chevron-down' : 'mdi-chevron-up'"></i></button>
        <div class="telemetry-card__live" :class="{ 'is-paused': paused || !live }">
          <span class="telemetry-card__live-dot"></span>
          {{ paused ? 'PAUSED' : live ? 'LIVE' : 'IDLE' }}
        </div>
      </div>
    </div>

    <div class="telemetry-card__body" v-show="!isCollapsed" @mouseenter="pauseLive" @mouseleave="resumeLive">
      <apexchart
        type="area"
        :height="height"
        :options="chartOptions"
        :series="normalizedChartSeries"
      />
    </div>
  </article>
</template>

<script>
function numericValues(series = []) {
  return series.flatMap(item => (item.data || [])
    .map(point => Number(typeof point === 'object' && point !== null ? point.y : point))
    .filter(value => Number.isFinite(value)))
}

function pointTimestamp(point) {
  const value = typeof point === 'object' && point !== null ? point.x : null
  const timestamp = Number(value)
  return Number.isFinite(timestamp) ? timestamp : null
}

function rangeDurationMs(range) {
  return {
    '1m': 60_000,
    '5m': 300_000,
    '15m': 900_000,
    '1h': 3_600_000
  }[range] || 60_000
}

function relativeLabel(timestamp) {
  const secondsAgo = Math.max(0, Math.round((Date.now() - Number(timestamp || Date.now())) / 1000))
  if (secondsAgo >= 3600) return `${Math.round(secondsAgo / 3600)}h ago`
  if (secondsAgo >= 60) return `${Math.round(secondsAgo / 60)}m ago`
  return `${secondsAgo}s ago`
}

function lastEnabledRange(ranges = []) {
  return ranges.find(option => option.enabled)?.key || (ranges[0]?.key ?? '1m')
}

export default {
  name: 'DashboardTelemetryChart',
  props: {
    title: { type: String, required: true },
    description: { type: String, default: '' },
    icon: { type: String, required: true },
    series: { type: Array, default: () => [] },
    live: { type: Boolean, default: false },
    height: { type: Number, default: 260 },
    stacked: { type: Boolean, default: false },
    formatter: { type: Function, default: null },
    thresholds: { type: Array, default: () => [] },
    rangeOptions: {
      type: Array,
      default: () => ([
        { key: '1m', label: '1m', enabled: true },
        { key: '5m', label: '5m', enabled: false },
        { key: '15m', label: '15m', enabled: false },
        { key: '1h', label: '1h', enabled: false }
      ])
    },
    percentScale: { type: Boolean, default: false }
  },
  data() {
    return {
      isCollapsed: false,
      paused: false,
      frozenSeries: [],
      selectedRange: lastEnabledRange(this.rangeOptions),
      lockScale: false,
      currentTheme: 'dark'
    }
  },
  computed: {
    normalizedRanges() {
      return this.rangeOptions.map(option => ({ enabled: option.enabled !== false, ...option }))
    },
    chartSeries() {
      return this.paused ? this.frozenSeries : this.series
    },
    normalizedChartSeries() {
      return this.chartSeries.map(item => ({
        ...item,
        data: (item.data || []).map((point, index) => {
          if (typeof point === 'object' && point !== null) return point
          return {
            x: Date.now() - ((item.data || []).length - 1 - index) * 1000,
            y: point
          }
        })
      }))
    },
    yBounds() {
      if (this.lockScale || this.percentScale) {
        return { min: 0, max: 100 }
      }
      const values = numericValues(this.normalizedChartSeries)
      if (!values.length) return { min: 0, max: 10 }
      const sorted = [...values].sort((left, right) => left - right)
      const min = sorted[Math.floor(sorted.length * 0.05)] ?? sorted[0]
      const max = sorted[Math.ceil(sorted.length * 0.95) - 1] ?? sorted[sorted.length - 1]
      const span = Math.max(max - min, max * 0.12, 1)
      const lower = Math.max(0, min - span * 0.25)
      const upper = max + span * 0.2
      return { min: Number(lower.toFixed(2)), max: Number(upper.toFixed(2)) }
    },
    xBounds() {
      const timestamps = this.normalizedChartSeries
        .flatMap(item => (item.data || []).map(pointTimestamp))
        .filter(value => value !== null)
      const max = timestamps.length ? Math.max(...timestamps) : Date.now()
      return { min: max - rangeDurationMs(this.selectedRange), max }
    },
    chartOptions() {
      const labelColor = 'var(--text-tertiary)'
      return {
        chart: {
          id: `dashboard-${this._.uid}`,
          group: 'dashboard-telemetry',
          type: 'area',
          stacked: this.stacked,
          toolbar: { show: false },
          zoom: { enabled: false },
          animations: { enabled: false },
          background: 'transparent',
          foreColor: labelColor,
          parentHeightOffset: 0
        },
        stroke: {
          curve: 'smooth',
          width: this.chartSeries.map(() => 2.4)
        },
        fill: {
          type: 'gradient',
          gradient: {
            shadeIntensity: 0.2,
            opacityFrom: 0.28,
            opacityTo: 0.04,
            stops: [0, 100]
          }
        },
        grid: {
          borderColor: 'var(--dashboard-panel-border)',
          strokeDashArray: 5,
          padding: { left: 4, right: 8, top: 4, bottom: 0 }
        },
        legend: {
          position: 'top',
          horizontalAlign: 'left',
          labels: { colors: 'var(--text-secondary)' }
        },
        dataLabels: { enabled: false },
        markers: { size: 0, hover: { size: 5 } },
        xaxis: {
          type: 'datetime',
          min: this.xBounds.min,
          max: this.xBounds.max,
          axisBorder: { show: false },
          axisTicks: { show: false },
          labels: {
            show: true,
            datetimeUTC: false,
            style: { colors: labelColor, fontSize: '11px' },
            formatter: value => relativeLabel(value)
          },
          crosshairs: {
            show: true,
            stroke: { color: 'var(--accent)', dashArray: 4 }
          }
        },
        yaxis: {
          min: this.yBounds.min,
          max: this.yBounds.max,
          labels: {
            style: { colors: labelColor, fontSize: '11px' },
            formatter: value => this.formatValue(value)
          }
        },
        tooltip: {
          theme: this.currentTheme,
          shared: true,
          x: {
            formatter: value => relativeLabel(value)
          },
          y: {
            formatter: value => this.formatValue(value)
          }
        },
        annotations: {
          yaxis: this.thresholds.map(threshold => ({
            y: threshold.value,
            borderColor: threshold.color,
            strokeDashArray: 4,
            label: {
              text: threshold.label,
              borderColor: threshold.color,
              style: {
                color: '#08111f',
                background: threshold.color,
                fontSize: '10px'
              }
            }
          }))
        },
        theme: { mode: this.currentTheme }
      }
    }
  },
  watch: {
    series: {
      deep: true,
      immediate: true,
      handler(value) {
        if (!this.paused) {
          this.frozenSeries = value.map(item => ({ ...item, data: [...(item.data || [])] }))
        }
      }
    }
  },
  mounted() {
    this.currentTheme = document.documentElement.getAttribute('data-theme') || 'dark'
    this._onStorageChange = (e) => {
      if (e.key === 'sc_theme') this.currentTheme = e.newValue === 'light' ? 'light' : 'dark'
    }
    this._onThemeChange = (e) => {
      this.currentTheme = e.detail || document.documentElement.getAttribute('data-theme') || 'dark'
    }
    window.addEventListener('storage', this._onStorageChange)
    window.addEventListener('sc:theme-change', this._onThemeChange)
  },
  beforeUnmount() {
    window.removeEventListener('storage', this._onStorageChange)
    window.removeEventListener('sc:theme-change', this._onThemeChange)
  },
  methods: {
    formatValue(value) {
      if (this.formatter) return this.formatter(value)
      if (Math.abs(value) >= 100) return Math.round(value)
      return Number(value).toFixed(1)
    },
    pauseLive() {
      this.paused = true
      this.frozenSeries = this.series.map(item => ({ ...item, data: [...(item.data || [])] }))
    },
    resumeLive() {
      this.paused = false
    }
  }
}
</script>

<style scoped>
.telemetry-card {
  border-radius: 22px;
  border: 1px solid var(--dashboard-panel-border);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.02), transparent 42%),
    var(--dashboard-panel-bg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.telemetry-card__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 18px 12px;
  border-bottom: 1px solid var(--dashboard-panel-border);
}

.telemetry-card__title-wrap {
  display: flex;
  gap: 12px;
  min-width: 0;
}

.telemetry-card__icon {
  width: 34px;
  height: 34px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: rgba(107, 168, 255, 0.14);
  color: var(--accent);
  font-size: 18px;
  flex-shrink: 0;
}

.telemetry-card__eyebrow {
  color: var(--text-tertiary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 700;
}

.telemetry-card__title {
  margin: 2px 0 0;
  font-size: 18px;
  color: var(--text-primary);
}

.telemetry-card__description {
  margin: 4px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.telemetry-card__controls {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
  align-items: flex-start;
}

.telemetry-card__range-group,
.telemetry-card__toggle,
.telemetry-card__live {
  border-radius: 999px;
  border: 1px solid var(--dashboard-panel-border);
  background: rgba(255, 255, 255, 0.02);
}

.telemetry-card__range-group {
  display: inline-flex;
  padding: 3px;
}

.telemetry-card__range,
.telemetry-card__toggle {
  color: var(--text-secondary);
  background: transparent;
  border: 0;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.telemetry-card__range {
  padding: 6px 9px;
  border-radius: 999px;
}

.telemetry-card__range.is-active {
  background: var(--accent-muted);
  color: var(--text-primary);
}

.telemetry-card__range.is-disabled {
  opacity: 0.45;
}

.telemetry-card__toggle {
  padding: 9px 12px;
}

.telemetry-card__live {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 12px;
  color: var(--dashboard-live-dot);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.telemetry-card__live.is-paused {
  color: var(--dashboard-stale-dot);
}

.telemetry-card__live-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 6px color-mix(in srgb, currentColor 18%, transparent);
  animation: telemetry-pulse 1.8s ease-in-out infinite;
}

.telemetry-card__body {
  padding: 10px 10px 6px;
}

@keyframes telemetry-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.55; transform: scale(0.88); }
}

@media (prefers-reduced-motion: reduce) {
  .telemetry-card__live-dot {
    animation: none;
  }
}

@media (max-width: 960px) {
  .telemetry-card__header {
    flex-direction: column;
  }
}
</style>