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
        <div class="telemetry-card__live" :class="{ 'is-paused': paused || !live }">
          <span class="telemetry-card__live-dot"></span>
          {{ paused ? 'PAUSED' : live ? 'LIVE' : 'IDLE' }}
        </div>
      </div>
    </div>

    <div class="telemetry-card__body" @mouseenter="pauseLive" @mouseleave="resumeLive">
      <apexchart
        type="area"
        :height="height"
        :options="chartOptions"
        :series="chartSeries"
      />
    </div>
  </article>
</template>

<script>
function numericValues(series = []) {
  return series.flatMap(item => (item.data || []).map(value => Number(value)).filter(value => Number.isFinite(value)))
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
      paused: false,
      frozenSeries: [],
      selectedRange: lastEnabledRange(this.rangeOptions),
      lockScale: false
    }
  },
  computed: {
    normalizedRanges() {
      return this.rangeOptions.map(option => ({ enabled: option.enabled !== false, ...option }))
    },
    chartSeries() {
      return this.paused ? this.frozenSeries : this.series
    },
    yBounds() {
      if (this.lockScale || this.percentScale) {
        return { min: 0, max: 100 }
      }
      const values = numericValues(this.chartSeries)
      if (!values.length) return { min: 0, max: 10 }
      const min = Math.min(...values)
      const max = Math.max(...values)
      const span = Math.max(max - min, max * 0.1, 1)
      const lower = Math.max(0, min - span * 0.25)
      const upper = max + span * 0.2
      return { min: Number(lower.toFixed(2)), max: Number(upper.toFixed(2)) }
    },
    xCategories() {
      const sampleCount = Math.max(...this.chartSeries.map(item => item.data?.length || 0), 0)
      const ranges = {
        '1m': 60,
        '5m': 300,
        '15m': 900,
        '1h': 3600
      }
      const duration = ranges[this.selectedRange] || 60
      return Array.from({ length: sampleCount }, (_, index) => {
        const secondsAgo = duration - Math.round((index / Math.max(sampleCount - 1, 1)) * duration)
        if (secondsAgo >= 3600) return `${Math.round(secondsAgo / 3600)}h`
        if (secondsAgo >= 60) return `${Math.round(secondsAgo / 60)}m`
        return `${secondsAgo}s`
      })
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
          animations: { enabled: !this.paused },
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
          categories: this.xCategories,
          axisBorder: { show: false },
          axisTicks: { show: false },
          labels: {
            show: true,
            style: { colors: labelColor, fontSize: '11px' },
            formatter: (value, timestamp, opts) => {
              const index = opts?.dataPointIndex ?? 0
              const cadence = Math.max(1, Math.floor(this.xCategories.length / 6))
              return index % cadence === 0 ? value : ''
            }
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
          theme: 'dark',
          shared: true,
          x: {
            formatter: (value, ctx) => this.xCategories[ctx.dataPointIndex] || value
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
        theme: { mode: 'dark' }
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