<template>
  <div class="mini-timeseries-chart">
    <svg :viewBox="`0 0 ${width} ${height}`" class="mini-timeseries-chart__svg" preserveAspectRatio="none" aria-hidden="true">
      <g v-for="tick in yTicks" :key="`grid-${tick.value}`">
        <line class="mini-timeseries-chart__grid" :x1="padding.left" :x2="width - padding.right" :y1="tick.y" :y2="tick.y" />
        <text class="mini-timeseries-chart__label" :x="padding.left - 8" :y="tick.y + 4" text-anchor="end">{{ formatValue(tick.value) }}</text>
      </g>

      <g v-for="threshold in thresholdLines" :key="`threshold-${threshold.label}-${threshold.value}`">
        <line
          class="mini-timeseries-chart__threshold"
          :x1="padding.left"
          :x2="width - padding.right"
          :y1="threshold.y"
          :y2="threshold.y"
          :stroke="threshold.color"
        />
        <text
          class="mini-timeseries-chart__threshold-label"
          :x="width - padding.right"
          :y="threshold.y - 6"
          text-anchor="end"
          :fill="threshold.color"
        >{{ threshold.label }}</text>
      </g>

      <g v-for="entry in drawableSeries" :key="entry.name">
        <polygon
          v-if="entry.areaPoints"
          class="mini-timeseries-chart__area"
          :points="entry.areaPoints"
          :fill="entry.fillColor"
        />
        <polyline
          v-if="entry.linePoints"
          class="mini-timeseries-chart__line"
          :points="entry.linePoints"
          :stroke="entry.color"
        />
        <circle
          v-if="entry.dot"
          :cx="entry.dot.x"
          :cy="entry.dot.y"
          r="3.25"
          :fill="entry.color"
        />
      </g>

      <g v-for="tick in xTicks" :key="`x-${tick.label}`">
        <text class="mini-timeseries-chart__label" :x="tick.x" :y="height - 6" text-anchor="middle">{{ tick.label }}</text>
      </g>
    </svg>

    <div v-if="showLegend && drawableSeries.length" class="mini-timeseries-chart__legend">
      <span v-for="entry in drawableSeries" :key="`legend-${entry.name}`" class="mini-timeseries-chart__legend-item">
        <i :style="{ background: entry.color }"></i>
        {{ entry.name }}
      </span>
    </div>

    <div v-if="!drawableSeries.length" class="mini-timeseries-chart__empty">No chart data yet.</div>
  </div>
</template>

<script>
function parsePoint(point, index) {
  if (typeof point === 'number') {
    return Number.isFinite(point) ? { x: index, y: point } : null
  }
  if (!point || typeof point !== 'object') return null
  const x = Number(point.x)
  const y = Number(point.y)
  if (!Number.isFinite(y)) return null
  return { x: Number.isFinite(x) ? x : index, y }
}

function rgba(color, alpha) {
  if (typeof color !== 'string') return `rgba(107,168,255,${alpha})`
  if (color.startsWith('#') && (color.length === 7 || color.length === 4)) {
    let value = color.slice(1)
    if (value.length === 3) {
      value = value.split('').map(char => char + char).join('')
    }
    const r = Number.parseInt(value.slice(0, 2), 16)
    const g = Number.parseInt(value.slice(2, 4), 16)
    const b = Number.parseInt(value.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  return color
}

function formatAxisTime(value, isTimeScale) {
  if (!isTimeScale) return ''
  return _timeFormatter.format(new Date(value))
}

// Module-level cached formatter — constructing Intl.DateTimeFormat is
// expensive (locale negotiation). Reuse a single instance for all charts.
const _timeFormatter = new Intl.DateTimeFormat([], {
  hour: '2-digit',
  minute: '2-digit'
})

export default {
  name: 'MiniTimeseriesChart',
  props: {
    series: { type: Array, default: () => [] },
    height: { type: Number, default: 260 },
    width: { type: Number, default: 1000 },
    formatter: { type: Function, default: null },
    thresholds: { type: Array, default: () => [] },
    percentScale: { type: Boolean, default: false },
    showLegend: { type: Boolean, default: true },
    minValue: { type: Number, default: null },
    maxValue: { type: Number, default: null }
  },
  computed: {
    padding() {
      return { top: 14, right: 14, bottom: 28, left: 48 }
    },
    plotWidth() {
      return Math.max(10, this.width - this.padding.left - this.padding.right)
    },
    plotHeight() {
      return Math.max(10, this.height - this.padding.top - this.padding.bottom)
    },
    normalizedSeries() {
      return this.series
        .map((entry, seriesIndex) => {
          const points = (entry?.data || [])
            .map((point, index) => parsePoint(point, index))
            .filter(Boolean)
          return {
            name: entry?.name || `Series ${seriesIndex + 1}`,
            color: entry?.color || '#6ba8ff',
            points
          }
        })
        .filter(entry => entry.points.length)
    },
    isTimeScale() {
      const firstPoint = this.normalizedSeries[0]?.points?.[0]
      return Number(firstPoint?.x) > 10_000
    },
    xDomain() {
      const xs = this.normalizedSeries.flatMap(entry => entry.points.map(point => point.x))
      if (!xs.length) return { min: 0, max: 1 }
      return {
        min: Math.min(...xs),
        max: Math.max(...xs)
      }
    },
    yDomain() {
      if (this.percentScale) return { min: 0, max: 100 }
      if (Number.isFinite(this.minValue) && Number.isFinite(this.maxValue)) {
        return { min: this.minValue, max: this.maxValue }
      }
      const ys = this.normalizedSeries.flatMap(entry => entry.points.map(point => point.y))
      if (!ys.length) return { min: 0, max: 10 }
      const rawMin = Number.isFinite(this.minValue) ? this.minValue : Math.min(...ys)
      const rawMax = Number.isFinite(this.maxValue) ? this.maxValue : Math.max(...ys)
      const span = Math.max(1, rawMax - rawMin)
      return {
        min: Number.isFinite(this.minValue) ? this.minValue : Math.max(0, rawMin - span * 0.12),
        max: Number.isFinite(this.maxValue) ? this.maxValue : rawMax + span * 0.12
      }
    },
    drawableSeries() {
      const xSpan = Math.max(1, this.xDomain.max - this.xDomain.min)
      const ySpan = Math.max(1, this.yDomain.max - this.yDomain.min)
      const yBottom = this.padding.top + this.plotHeight
      return this.normalizedSeries.map(entry => {
        const svgPoints = entry.points.map(point => ({
          x: this.padding.left + ((point.x - this.xDomain.min) / xSpan) * this.plotWidth,
          y: this.padding.top + (1 - (point.y - this.yDomain.min) / ySpan) * this.plotHeight
        }))
        const linePoints = svgPoints.map(point => `${point.x},${point.y}`).join(' ')
        const firstPoint = svgPoints[0]
        const lastPoint = svgPoints[svgPoints.length - 1]
        return {
          ...entry,
          linePoints,
          areaPoints: svgPoints.length > 1
            ? `${firstPoint.x},${yBottom} ${linePoints} ${lastPoint.x},${yBottom}`
            : null,
          fillColor: rgba(entry.color, 0.12),
          dot: svgPoints.length === 1 ? svgPoints[0] : null
        }
      })
    },
    thresholdLines() {
      const ySpan = Math.max(1, this.yDomain.max - this.yDomain.min)
      return this.thresholds
        .filter(threshold => Number.isFinite(Number(threshold?.value)))
        .map(threshold => ({
          ...threshold,
          y: this.padding.top + (1 - (Number(threshold.value) - this.yDomain.min) / ySpan) * this.plotHeight
        }))
    },
    yTicks() {
      const values = []
      const steps = 4
      for (let index = 0; index <= steps; index += 1) {
        const ratio = index / steps
        const value = this.yDomain.max - (this.yDomain.max - this.yDomain.min) * ratio
        values.push({
          value,
          y: this.padding.top + this.plotHeight * ratio
        })
      }
      return values
    },
    xTicks() {
      if (!this.drawableSeries.length || !this.isTimeScale) return []
      const values = [0, 0.5, 1].map(ratio => this.xDomain.min + (this.xDomain.max - this.xDomain.min) * ratio)
      return values.map(value => ({
        x: this.padding.left + ((value - this.xDomain.min) / Math.max(1, this.xDomain.max - this.xDomain.min)) * this.plotWidth,
        label: formatAxisTime(value, this.isTimeScale)
      }))
    }
  },
  methods: {
    formatValue(value) {
      if (this.formatter) return this.formatter(value)
      if (Math.abs(value) >= 100) return Math.round(value)
      return Number(value).toFixed(1)
    }
  }
}
</script>

<style scoped>
.mini-timeseries-chart {
  position: relative;
  width: 100%;
}

.mini-timeseries-chart__svg {
  width: 100%;
  height: auto;
  display: block;
}

.mini-timeseries-chart__grid {
  stroke: rgba(30, 45, 74, 0.85);
  stroke-dasharray: 4 6;
}

.mini-timeseries-chart__label,
.mini-timeseries-chart__threshold-label,
.mini-timeseries-chart__empty {
  fill: #5a7499;
  color: #5a7499;
  font-size: 11px;
}

.mini-timeseries-chart__threshold {
  stroke-width: 1.2;
  stroke-dasharray: 5 5;
}

.mini-timeseries-chart__line {
  fill: none;
  stroke-width: 2.2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.mini-timeseries-chart__area {
  transition: opacity 160ms ease;
}

.mini-timeseries-chart__legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 10px;
  padding-left: 48px;
}

.mini-timeseries-chart__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #8aa4c8;
  font-size: 12px;
  font-weight: 600;
}

.mini-timeseries-chart__legend-item i {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  display: inline-block;
}

.mini-timeseries-chart__empty {
  padding: 24px 0 8px 48px;
}
</style>