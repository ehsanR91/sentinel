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

      <div class="kpi-card__sparkline" :class="{ 'kpi-card__sparkline--flat': !sparkline.length }" aria-hidden="true">
        <svg viewBox="0 0 220 48" preserveAspectRatio="none">
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
        </svg>
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

function normalizePoints(values = [], width = 220, height = 48) {
  const clean = values
    .map(value => Number(value))
    .filter(value => Number.isFinite(value))

  if (!clean.length) return []

  const min = Math.min(...clean)
  const max = Math.max(...clean)
  const span = max - min || 1

  return clean.map((value, index) => {
    const x = clean.length === 1 ? width / 2 : (index / (clean.length - 1)) * width
    const y = height - (((value - min) / span) * (height - 8) + 4)
    return [Number(x.toFixed(2)), Number(y.toFixed(2))]
  })
}

function linePath(points = []) {
  if (!points.length) return ''
  return points.map(([x, y], index) => `${index === 0 ? 'M' : 'L'}${x},${y}`).join(' ')
}

function areaPath(points = [], width = 220, height = 48) {
  if (!points.length) return ''
  const line = linePath(points)
  const lastPoint = points[points.length - 1]
  const firstPoint = points[0]
  return `${line} L${lastPoint[0]},${height} L${firstPoint[0]},${height} Z`
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
  overflow: hidden;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.02), transparent);
}

.kpi-card__sparkline svg {
  width: 100%;
  height: 100%;
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

.kpi-card__context {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 32px;
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
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.22);
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