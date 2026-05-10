<template>
  <component
    :is="interactive ? 'button' : 'article'"
    class="kpi-card sc-focus-ring"
    :class="[
      `kpi-card--${toneClass}`,
      {
        'kpi-card--interactive': interactive,
        'kpi-card--stale': stale,
        'kpi-card--loading': loading
      }
    ]"
    :type="interactive ? 'button' : undefined"
    :aria-label="ariaLabel"
    @click="interactive && $emit('click')"
  >
    <div class="kpi-card__header">
      <div class="kpi-card__heading">
        <span class="kpi-card__icon" aria-hidden="true">
          <i :class="icon"></i>
        </span>
        <div>
          <div class="kpi-card__label">{{ label }}</div>
          <div class="kpi-card__meta-row">
            <span v-if="rangeLabel" class="kpi-card__range">{{ rangeLabel }}</span>
            <span v-if="live" class="kpi-card__live" :class="{ 'kpi-card__live--paused': stale }">
              <span class="kpi-card__live-dot"></span>
              {{ stale ? 'STALE' : 'LIVE' }}
            </span>
          </div>
        </div>
      </div>
      <div v-if="statusLabel" class="kpi-card__status">{{ statusLabel }}</div>
    </div>

    <div v-if="loading" class="kpi-card__skeleton" aria-hidden="true">
      <span class="kpi-card__skeleton-line kpi-card__skeleton-line--lg"></span>
      <span class="kpi-card__skeleton-line"></span>
      <span class="kpi-card__skeleton-chart"></span>
      <span class="kpi-card__skeleton-line kpi-card__skeleton-line--sm"></span>
    </div>

    <template v-else>
      <div class="kpi-card__value-row">
        <div class="kpi-card__value">{{ value }}</div>
        <div v-if="deltaLabel" class="kpi-card__delta" :class="`kpi-card__delta--${deltaToneClass}`">
          <i :class="deltaIcon"></i>
          <span>{{ deltaLabel }}</span>
        </div>
      </div>

      <!-- ── Gauge (arc pressure meter) ───────────────────────────── -->
      <div v-if="variant === 'gauge'" class="kpi-card__gauge" aria-hidden="true">
        <svg viewBox="0 0 120 66" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M10,60 A50,50 0 0,1 110,60" class="kpi-card__gauge-track" />
          <path v-if="hasThreshold" :d="gaugeArcOk"    class="kpi-card__gauge-ok"    />
          <path v-if="hasThreshold" :d="gaugeArcWarn"  class="kpi-card__gauge-warn"  />
          <path v-if="hasThreshold" :d="gaugeArcError" class="kpi-card__gauge-error" />
          <line v-if="hasThreshold"
            x1="60" y1="60"
            :x2="gaugeNeedleX2" :y2="gaugeNeedleY2"
            class="kpi-card__gauge-needle"
          />
          <circle cx="60" cy="60" r="3.5" class="kpi-card__gauge-pivot" />
          <text x="11"  y="65" class="kpi-card__gauge-label">0</text>
          <text v-if="hasThreshold" :x="gaugeWarnLabelX" :y="gaugeWarnLabelY" class="kpi-card__gauge-label" text-anchor="middle">{{ threshold.warn }}</text>
          <text x="109" y="65" class="kpi-card__gauge-label" text-anchor="end">{{ hasThreshold ? threshold.max : '' }}</text>
        </svg>
      </div>

      <!-- ── Stat grid ─────────────────────────────────────────────── -->
      <div v-else-if="variant === 'stat-grid'" class="kpi-card__stat-grid">
        <div
          v-for="stat in statItems"
          :key="stat.label"
          class="kpi-card__stat-cell"
          :class="`kpi-card__stat-cell--${stat.tone || 'default'}`"
        >
          <span class="kpi-card__stat-cell-value">{{ stat.value }}</span>
          <span class="kpi-card__stat-cell-label">{{ stat.label }}</span>
        </div>
      </div>

      <!-- ── Uptime banner ──────────────────────────────────────────── -->
      <div v-else-if="variant === 'uptime'" class="kpi-card__uptime-banner">
        <span class="kpi-card__uptime-dot"></span>
        <span class="kpi-card__uptime-text">{{ uptimeSubtitle }}</span>
      </div>

      <!-- ── Default sparkline ──────────────────────────────────────── -->
      <div v-else class="kpi-card__sparkline" :class="{ 'kpi-card__sparkline--flat': !sparkline.length }" aria-hidden="true">
        <svg ref="sparklineSvg" viewBox="0 0 220 48" preserveAspectRatio="none"
          @mousemove="onSparklineMove"
          @mouseleave="onSparklineLeave"
        >
          <defs>
            <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" :stop-color="sparkColor" stop-opacity="0.3" />
              <stop offset="100%" :stop-color="sparkColor" stop-opacity="0" />
            </linearGradient>
          </defs>
          <path v-if="primaryArea" :d="primaryArea" :fill="`url(#${gradientId})`"></path>
          <path v-if="secondaryLine" :d="secondaryLine" class="kpi-card__sparkline-secondary"></path>
          <path v-if="primaryLine" :d="primaryLine" class="kpi-card__sparkline-line" :style="{ stroke: sparkColor }"></path>
          <line v-if="!primaryLine" x1="0" y1="24" x2="220" y2="24" class="kpi-card__sparkline-empty"></line>
          <line v-if="hoverPoint"
            :x1="hoverPoint.px" y1="4"
            :x2="hoverPoint.px" y2="44"
            class="kpi-card__sparkline-crosshair"
          />
          <circle v-if="hoverPoint"
            :cx="hoverPoint.px" :cy="hoverPoint.py" r="2.5"
            class="kpi-card__sparkline-dot" :style="{ fill: sparkColor }"
          />
        </svg>
        <div v-if="hoverPoint" class="kpi-card__sparkline-tooltip" :style="hoverTooltipStyle">
          <span class="kpi-card__sparkline-tooltip-value">{{ hoverValueLabel }}</span>
          <span v-if="hoverTimeLabel" class="kpi-card__sparkline-tooltip-time">{{ hoverTimeLabel }}</span>
        </div>
      </div>

      <div class="kpi-card__context">
        <div v-for="(line, index) in contextLines" :key="`${label}-ctx-${index}`" class="kpi-card__context-line">{{ line }}</div>
      </div>

      <div v-if="hasThreshold" class="kpi-card__threshold" aria-hidden="true">
        <div class="kpi-card__threshold-track">
          <span class="kpi-card__threshold-zone kpi-card__threshold-zone--ok" :style="{ width: `${okWidth}%` }"></span>
          <span class="kpi-card__threshold-zone kpi-card__threshold-zone--warn" :style="{ width: `${warnWidth}%` }"></span>
          <span class="kpi-card__threshold-zone kpi-card__threshold-zone--error" :style="{ width: `${errorWidth}%` }"></span>
          <span class="kpi-card__threshold-marker" :style="{ left: `${markerPosition}%` }"></span>
        </div>
      </div>
    </template>
  </component>
</template>

<script>
function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max)
}

// Module-level cached time formatter for hover tooltip
const _sparkTimeFmt = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })

// Accepts plain numbers OR {x: timestamp, y: value} objects.
// Nulls and non-finite values are properly excluded.
function normalizePoints(series = [], width = 220, height = 48) {
  const items = series.map(v => {
    if (v !== null && typeof v === 'object') return { v: v.y, ts: v.x }
    return { v, ts: null }
  }).filter(({ v }) => v !== null && v !== undefined && Number.isFinite(Number(v)))

  if (!items.length) return []

  const values = items.map(i => Number(i.v))
  const min = Math.min(...values)
  const max = Math.max(...values)
  const span = max - min || 1

  return items.map(({ v, ts }, index) => {
    const numV = Number(v)
    const x = items.length === 1 ? width / 2 : (index / (items.length - 1)) * width
    const y = height - (((numV - min) / span) * (height - 8) + 4)
    return { px: Number(x.toFixed(2)), py: Number(y.toFixed(2)), v: numV, ts }
  })
}

function linePath(points = []) {
  if (!points.length) return ''
  return points.map(({ px, py }, index) => `${index === 0 ? 'M' : 'L'}${px},${py}`).join(' ')
}

function areaPath(points = [], height = 48) {
  if (!points.length) return ''
  const line = linePath(points)
  const lastPoint = points[points.length - 1]
  const firstPoint = points[0]
  return `${line} L${lastPoint.px},${height} L${firstPoint.px},${height} Z`
}

export default {
  name: 'DashboardKpiCard',
  props: {
    label: { type: String, required: true },
    icon: { type: String, required: true },
    value: { type: [String, Number], required: true },
    deltaLabel: { type: String, default: '' },
    deltaDirection: { type: String, default: 'neutral' },
    deltaTone: { type: String, default: 'neutral' },
    sparkline: { type: Array, default: () => [] },
    sparklineSecondary: { type: Array, default: () => [] },
    contextLines: { type: Array, default: () => [] },
    threshold: { type: Object, default: null },
    variant: { type: String, default: 'spark' },
    statItems: { type: Array, default: () => [] },
    uptimeSubtitle: { type: String, default: 'System stable' },
    interactive: { type: Boolean, default: true },
    live: { type: Boolean, default: false },
    stale: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    rangeLabel: { type: String, default: '' },
    statusLabel: { type: String, default: '' },
    tone: { type: String, default: 'default' },
    sparkColor: { type: String, default: 'var(--dashboard-spark-line)' }
  },
  emits: ['click'],
  data() {
    return {
      hoverIdx: null
    }
  },
  computed: {
    ariaLabel() {
      return `${this.label}: ${this.value}${this.deltaLabel ? `. Change ${this.deltaLabel}.` : ''}`
    },
    gradientId() {
      return `kpi-gradient-${this._.uid}`
    },
    toneClass() {
      if (this.stale) return 'stale'
      return ['ok', 'warn', 'error', 'critical'].includes(this.tone) ? this.tone : 'default'
    },
    deltaToneClass() {
      return ['good', 'bad', 'neutral'].includes(this.deltaTone) ? this.deltaTone : 'neutral'
    },
    deltaIcon() {
      return {
        up: 'mdi mdi-arrow-top-right',
        down: 'mdi mdi-arrow-bottom-right',
        neutral: 'mdi mdi-minus'
      }[this.deltaDirection] || 'mdi mdi-minus'
    },
    primaryPoints() {
      return normalizePoints(this.sparkline)
    },
    secondaryPoints() {
      return normalizePoints(this.sparklineSecondary)
    },
    primaryLine() {
      return linePath(this.primaryPoints)
    },
    primaryArea() {
      return areaPath(this.primaryPoints)
    },
    secondaryLine() {
      return linePath(this.secondaryPoints)
    },
    hoverPoint() {
      if (this.hoverIdx === null || !this.primaryPoints.length) return null
      return this.primaryPoints[this.hoverIdx]
    },
    hoverTooltipStyle() {
      if (!this.hoverPoint) return {}
      const pct = (this.hoverPoint.px / 220) * 100
      return {
        left: `${pct}%`,
        transform: pct > 60 ? 'translateX(-100%)' : 'translateX(0)'
      }
    },
    hoverValueLabel() {
      if (!this.hoverPoint) return ''
      const v = this.hoverPoint.v
      return Number.isFinite(v) ? (v % 1 < 0.05 ? String(Math.round(v)) : v.toFixed(1)) : ''
    },
    hoverTimeLabel() {
      if (!this.hoverPoint || !this.hoverPoint.ts) return ''
      return _sparkTimeFmt.format(new Date(this.hoverPoint.ts))
    },
    hasThreshold() {
      return !!(this.threshold && Number.isFinite(Number(this.threshold.max)))
    },
    maxThreshold() {
      return Math.max(Number(this.threshold?.max || 100), 1)
    },
    okWidth() {
      return clamp((Number(this.threshold?.warn || this.maxThreshold) / this.maxThreshold) * 100, 0, 100)
    },
    warnWidth() {
      const crit = Number(this.threshold?.crit || this.maxThreshold)
      return clamp(((crit - Number(this.threshold?.warn || 0)) / this.maxThreshold) * 100, 0, 100)
    },
    errorWidth() {
      return clamp(100 - this.okWidth - this.warnWidth, 0, 100)
    },
    markerPosition() {
      return clamp((Number(this.threshold?.value || 0) / this.maxThreshold) * 100, 0, 100)
    },
    // SVG arc gauge helpers — semi-circle from left (180°) to right (0°)
    // Centre (60,60), radius 50, arc from angle 180→0 (degrees in SVG space)
    gaugeNeedleX2() {
      const pct = clamp(Number(this.threshold?.value || 0) / this.maxThreshold, 0, 1)
      const angle = Math.PI - pct * Math.PI // 180° → 0°
      return Number((60 + 42 * Math.cos(angle)).toFixed(2))
    },
    gaugeNeedleY2() {
      const pct = clamp(Number(this.threshold?.value || 0) / this.maxThreshold, 0, 1)
      const angle = Math.PI - pct * Math.PI
      return Number((60 - 42 * Math.sin(angle)).toFixed(2))
    },
    gaugeWarnLabelX() {
      const pct = clamp(Number(this.threshold?.warn || 0) / this.maxThreshold, 0, 1)
      const angle = Math.PI - pct * Math.PI
      return Number((60 + 54 * Math.cos(angle)).toFixed(2))
    },
    gaugeWarnLabelY() {
      const pct = clamp(Number(this.threshold?.warn || 0) / this.maxThreshold, 0, 1)
      const angle = Math.PI - pct * Math.PI
      return Number((60 - 54 * Math.sin(angle) + 4).toFixed(2))
    },
    // Arc path for a zone: fromPct → toPct along the semi-circle
    gaugeArcOk() {
      return this._gaugeZoneArc(0, Number(this.threshold?.warn || 0) / this.maxThreshold)
    },
    gaugeArcWarn() {
      const warnPct = Number(this.threshold?.warn || 0) / this.maxThreshold
      const critPct = Number(this.threshold?.crit || this.maxThreshold) / this.maxThreshold
      return this._gaugeZoneArc(warnPct, critPct)
    },
    gaugeArcError() {
      const critPct = Number(this.threshold?.crit || this.maxThreshold) / this.maxThreshold
      return this._gaugeZoneArc(critPct, 1)
    }
  },
  methods: {
    _gaugeZoneArc(fromPct, toPct) {
      const r = 50; const cx = 60; const cy = 60
      const a1 = Math.PI - clamp(fromPct, 0, 1) * Math.PI
      const a2 = Math.PI - clamp(toPct, 0, 1) * Math.PI
      const x1 = cx + r * Math.cos(a1); const y1 = cy - r * Math.sin(a1)
      const x2 = cx + r * Math.cos(a2); const y2 = cy - r * Math.sin(a2)
      const large = toPct - fromPct > 0.5 ? 1 : 0
      return `M${x1.toFixed(2)},${y1.toFixed(2)} A${r},${r} 0 ${large},1 ${x2.toFixed(2)},${y2.toFixed(2)}`
    },
    onSparklineMove(event) {
      if (!this.primaryPoints.length) return
      const svg = this.$refs.sparklineSvg
      const rect = svg.getBoundingClientRect()
      const svgX = ((event.clientX - rect.left) / rect.width) * 220
      let closest = 0
      let minDist = Infinity
      this.primaryPoints.forEach(({ px }, i) => {
        const d = Math.abs(px - svgX)
        if (d < minDist) { minDist = d; closest = i }
      })
      this.hoverIdx = closest
    },
    onSparklineLeave() {
      this.hoverIdx = null
    }
  }
}
</script>

<style scoped>
.kpi-card {
  width: 100%;
  min-height: 148px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-radius: 18px;
  border: 1px solid var(--dashboard-panel-border);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.02), transparent 38%),
    var(--dashboard-panel-bg);
  box-shadow: var(--shadow-md);
  text-align: left;
  position: relative;
  overflow: hidden;
  font-variant-numeric: tabular-nums;
}

.kpi-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--dashboard-grid-pattern) 1px, transparent 1px),
    linear-gradient(90deg, var(--dashboard-grid-pattern) 1px, transparent 1px);
  background-size: 28px 28px;
  opacity: 0.35;
  pointer-events: none;
}

.kpi-card--interactive {
  cursor: pointer;
}

.kpi-card--interactive:hover {
  transform: translateY(-2px);
}

.kpi-card--ok {
  box-shadow: var(--shadow-md), var(--dashboard-glow-ok);
}

.kpi-card--warn {
  box-shadow: var(--shadow-md), var(--dashboard-glow-warn);
}

.kpi-card--error,
.kpi-card--critical {
  box-shadow: var(--shadow-md), var(--dashboard-glow-error);
}

.kpi-card--stale {
  opacity: 0.85;
}

.kpi-card__header,
.kpi-card__value-row,
.kpi-card__context,
.kpi-card__threshold,
.kpi-card__skeleton {
  position: relative;
  z-index: 1;
}

.kpi-card__header,
.kpi-card__value-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.kpi-card__heading {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.kpi-card__icon {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: rgba(107, 168, 255, 0.12);
  color: var(--accent);
  font-size: 16px;
}

.kpi-card__label,
.kpi-card__range,
.kpi-card__live,
.kpi-card__status {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.kpi-card__label {
  color: var(--text-secondary);
  font-weight: 700;
}

.kpi-card__meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
}

.kpi-card__range,
.kpi-card__status {
  color: var(--text-tertiary);
}

.kpi-card__live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dashboard-live-dot);
}

.kpi-card__live--paused {
  color: var(--dashboard-stale-dot);
}

.kpi-card__live-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 6px color-mix(in srgb, currentColor 16%, transparent);
  animation: kpi-pulse 1.6s ease-in-out infinite;
}

.kpi-card__value {
  color: var(--text-primary);
  font-size: clamp(24px, 3vw, 31px);
  line-height: 1;
  font-weight: 700;
}

.kpi-card__delta {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  margin-top: 6px;
}

.kpi-card__delta--good {
  color: var(--state-ok-fg);
}

.kpi-card__delta--bad {
  color: var(--state-error-fg);
}

.kpi-card__delta--neutral {
  color: var(--text-tertiary);
}

.kpi-card__sparkline {
  position: relative;
  z-index: 1;
  height: 48px;
  border-radius: 10px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.02), transparent);
}

.kpi-card__sparkline svg {
  width: 100%;
  height: 100%;
  display: block;
  cursor: crosshair;
}

.kpi-card__sparkline-line {
  fill: none;
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.kpi-card__sparkline-secondary {
  fill: none;
  stroke: var(--dashboard-spark-line-alt);
  stroke-width: 1.75;
  stroke-linecap: round;
  stroke-linejoin: round;
  opacity: 0.75;
}

.kpi-card__sparkline-empty {
  stroke: var(--border-default);
  stroke-width: 1;
  stroke-dasharray: 4 5;
}

.kpi-card__sparkline-crosshair {
  stroke: var(--border-strong);
  stroke-width: 1;
  stroke-dasharray: 3 3;
  pointer-events: none;
}

.kpi-card__sparkline-dot {
  pointer-events: none;
}

.kpi-card__sparkline-tooltip {
  position: absolute;
  top: 3px;
  background: var(--surface-3);
  border: 1px solid var(--border-default);
  border-radius: 7px;
  padding: 3px 7px;
  font-size: 10px;
  line-height: 1.5;
  white-space: nowrap;
  pointer-events: none;
  z-index: 20;
  color: var(--text-primary);
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 0;
}

.kpi-card__sparkline-tooltip-value {
  font-weight: 700;
  font-family: var(--font-family-monospace, monospace);
  font-size: 11px;
}

.kpi-card__sparkline-tooltip-time {
  color: var(--text-tertiary);
  font-size: 9px;
}

.kpi-card__context {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 32px;
}

/* ── Gauge ─────────────────────────────────────────────────────────── */
.kpi-card__gauge {
  position: relative;
  z-index: 1;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.kpi-card__gauge svg {
  width: 100%;
  max-width: 180px;
  height: auto;
}

.kpi-card__gauge-track {
  stroke: var(--border-subtle);
  stroke-width: 6;
  stroke-linecap: round;
  fill: none;
}

.kpi-card__gauge-ok {
  stroke: var(--state-ok);
  stroke-width: 6;
  stroke-linecap: butt;
  fill: none;
  opacity: 0.75;
}

.kpi-card__gauge-warn {
  stroke: var(--state-warn, #f5a623);
  stroke-width: 6;
  stroke-linecap: butt;
  fill: none;
  opacity: 0.75;
}

.kpi-card__gauge-error {
  stroke: var(--state-error);
  stroke-width: 6;
  stroke-linecap: butt;
  fill: none;
  opacity: 0.75;
}

.kpi-card__gauge-needle {
  stroke: var(--text-primary);
  stroke-width: 1.5;
  stroke-linecap: round;
}

.kpi-card__gauge-pivot {
  fill: var(--text-primary);
}

.kpi-card__gauge-label {
  font-size: 7px;
  fill: var(--text-tertiary);
}

/* ── Stat grid ──────────────────────────────────────────────────────── */
.kpi-card__stat-grid {
  position: relative;
  z-index: 1;
  display: flex;
  gap: 8px;
  height: 56px;
  align-items: stretch;
}

.kpi-card__stat-cell {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  border-radius: 12px;
  border: 1px solid var(--border-subtle);
  background: var(--surface-1);
  padding: 6px 4px;
}

.kpi-card__stat-cell--ok    { border-color: color-mix(in srgb, var(--state-ok) 30%, transparent); background: color-mix(in srgb, var(--state-ok) 8%, var(--surface-1)); }
.kpi-card__stat-cell--warn  { border-color: color-mix(in srgb, var(--state-warn, #f5a623) 30%, transparent); background: color-mix(in srgb, var(--state-warn, #f5a623) 8%, var(--surface-1)); }
.kpi-card__stat-cell--error { border-color: color-mix(in srgb, var(--state-error) 30%, transparent); background: color-mix(in srgb, var(--state-error) 8%, var(--surface-1)); }

.kpi-card__stat-cell-value {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.kpi-card__stat-cell-label {
  font-size: 9px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-tertiary);
  font-weight: 700;
}

/* ── Uptime banner ──────────────────────────────────────────────────── */
.kpi-card__uptime-banner {
  position: relative;
  z-index: 1;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: color-mix(in srgb, var(--state-ok) 8%, var(--surface-1));
  border: 1px solid color-mix(in srgb, var(--state-ok) 22%, transparent);
  border-radius: 12px;
}

.kpi-card__uptime-dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: var(--state-ok);
  box-shadow: 0 0 0 5px color-mix(in srgb, var(--state-ok) 18%, transparent);
  animation: kpi-pulse 1.8s ease-in-out infinite;
  flex: none;
}

.kpi-card__uptime-text {
  font-size: 12px;
  font-weight: 600;
  color: var(--state-ok);
  letter-spacing: 0.04em;
}

.kpi-card__context-line {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: var(--font-family-monospace);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kpi-card__threshold-track {
  height: 6px;
  display: flex;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.04);
  position: relative;
}

.kpi-card__threshold-zone--ok {
  background: var(--dashboard-threshold-ok);
}

.kpi-card__threshold-zone--warn {
  background: var(--dashboard-threshold-warn);
}

.kpi-card__threshold-zone--error {
  background: var(--dashboard-threshold-error);
}

.kpi-card__threshold-marker {
  position: absolute;
  top: -3px;
  width: 12px;
  height: 12px;
  border-radius: 999px;
  background: var(--text-primary);
  border: 2px solid var(--surface-1);
  transform: translateX(-50%);
  box-shadow: var(--shadow-sm);
}

.kpi-card__skeleton {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kpi-card__skeleton-line,
.kpi-card__skeleton-chart {
  border-radius: 8px;
  background: linear-gradient(90deg, rgba(138, 164, 200, 0.14) 25%, rgba(138, 164, 200, 0.28) 50%, rgba(138, 164, 200, 0.14) 75%);
  background-size: 200% 100%;
  animation: kpi-shimmer 1.4s linear infinite;
}

.kpi-card__skeleton-line {
  height: 12px;
  width: 72%;
}

.kpi-card__skeleton-line--lg {
  height: 30px;
  width: 42%;
}

.kpi-card__skeleton-line--sm {
  width: 54%;
}

.kpi-card__skeleton-chart {
  height: 44px;
}

@keyframes kpi-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.55; transform: scale(0.86); }
}

@keyframes kpi-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

@media (prefers-reduced-motion: reduce) {
  .kpi-card,
  .kpi-card__live-dot,
  .kpi-card__skeleton-line,
  .kpi-card__skeleton-chart {
    animation: none !important;
    transition: none !important;
  }
}

@media (max-width: 768px) {
  .kpi-card {
    min-height: 132px;
    padding: 14px;
  }
}
</style>