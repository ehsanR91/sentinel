<template>
  <section class="health-anchor sc-focus-ring">
    <div class="health-anchor__header">
      <div>
        <div class="health-anchor__eyebrow">Health Anchor</div>
        <h2 class="health-anchor__title">System Health</h2>
      </div>
      <AppButton variant="secondary" size="sm" icon="mdi mdi-arrow-top-right" label="Open details" @click="$emit('open')" />
    </div>

    <div v-if="loading" class="health-anchor__loading" aria-hidden="true">
      <span class="health-anchor__loading-ring"></span>
      <div class="health-anchor__loading-lines">
        <span></span>
        <span></span>
        <span></span>
      </div>
    </div>

    <div v-else class="health-anchor__body">
      <button
        type="button"
        class="health-anchor__meter"
        :class="`health-anchor__meter--${statusTone}`"
        :aria-valuenow="score"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-valuetext="`${score} out of 100, ${statusWord}`"
        role="meter"
        @click="$emit('open')"
      >
        <svg viewBox="0 0 180 180" class="health-anchor__svg" aria-hidden="true">
          <circle cx="90" cy="90" r="64" class="health-anchor__track"></circle>
          <circle
            cx="90"
            cy="90"
            r="64"
            class="health-anchor__progress"
            :class="`health-anchor__progress--${statusTone}`"
            :style="outerProgressStyle"
          ></circle>
          <path
            v-for="segment in categorySegments"
            :key="segment.key"
            :d="segment.path"
            class="health-anchor__segment"
            :style="{ stroke: segment.color }"
          ></path>
        </svg>
        <div class="health-anchor__center">
          <div class="health-anchor__score">{{ score }}</div>
          <div class="health-anchor__suffix">/ 100</div>
          <div class="health-anchor__status">{{ statusWord }}</div>
        </div>
      </button>

      <div class="health-anchor__content">
        <p class="health-anchor__summary">{{ healthData.summary || 'Collecting posture data from the agent.' }}</p>
        <div class="health-anchor__meta-row">
          <span class="health-anchor__posture-label">24h posture · {{ stale ? 'Stale' : `Updated ${relativeTimestamp}` }}</span>
        </div>
        <div class="health-anchor__history">
          <div class="health-anchor__history-label">24h posture</div>
          <svg viewBox="0 0 220 42" preserveAspectRatio="none" aria-hidden="true">
            <defs>
              <linearGradient :id="historyGradientId" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" :stop-color="ringColor" stop-opacity="0.26" />
                <stop offset="100%" :stop-color="ringColor" stop-opacity="0" />
              </linearGradient>
            </defs>
            <path v-if="historyArea" :d="historyArea" :fill="`url(#${historyGradientId})`"></path>
            <path v-if="historyLine" :d="historyLine" :style="{ stroke: ringColor }" class="health-anchor__history-line"></path>
            <line v-if="!historyLine" x1="0" y1="21" x2="220" y2="21" class="health-anchor__history-empty"></line>
          </svg>
        </div>

        <div class="health-anchor__issues">
          <div class="health-anchor__issues-head">
            <span>Open issues</span>
            <span>{{ prioritizedIssues.length }}</span>
          </div>
          <div v-if="!prioritizedIssues.length" class="health-anchor__empty">No active remediation items.</div>
          <div v-else class="health-anchor__issue-list">
            <div v-for="issue in prioritizedIssues" :key="issue.name" class="health-anchor__issue-row" :class="{ 'is-critical': issue.status === 'critical' }">
              <div class="health-anchor__issue-accent-bar" aria-hidden="true" />
              <div class="health-anchor__issue-badge" :class="`health-anchor__issue-badge--${issue.status}`">{{ severityLabel(issue.status) }}</div>
              <div class="health-anchor__issue-content">
                <div class="health-anchor__issue-title" :title="issue.name">{{ issue.name }}</div>
                <div class="health-anchor__issue-message" :title="issue.message">{{ issue.message }}</div>
              </div>
              <div class="health-anchor__issue-actions">
                <AppButton variant="secondary" size="sm" label="Fix" class="health-anchor__btn-fix" @click.stop="$emit('inspect-issue', issue)" />
                <button type="button" class="health-anchor__btn-arrow" aria-label="Inspect issue" @click.stop="$emit('inspect-issue', issue)">
                  <i class="mdi mdi-arrow-right" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script>
import AppButton from '@/components/ui/app-button.vue'

function polarToCartesian(cx, cy, radius, angleInDegrees) {
  const angleInRadians = ((angleInDegrees - 90) * Math.PI) / 180
  return {
    x: cx + radius * Math.cos(angleInRadians),
    y: cy + radius * Math.sin(angleInRadians)
  }
}

function describeArc(cx, cy, radius, startAngle, endAngle) {
  const start = polarToCartesian(cx, cy, radius, endAngle)
  const end = polarToCartesian(cx, cy, radius, startAngle)
  const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1'
  return ['M', start.x, start.y, 'A', radius, radius, 0, largeArcFlag, 0, end.x, end.y].join(' ')
}

function sparklinePoints(values = [], width = 220, height = 42) {
  const clean = values.map(value => Number(value)).filter(value => Number.isFinite(value))
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

function areaPath(points = [], width = 220, height = 42) {
  if (!points.length) return ''
  const last = points[points.length - 1]
  const first = points[0]
  return `${linePath(points)} L${last[0]},${height} L${first[0]},${height} Z`
}

import { getHealthTone, getHealthColor, getHealthStatusWord } from '@/utils/health'

export default {
  name: 'DashboardHealthRing',
  components: { AppButton },
  props: {
    healthData: { type: Object, required: true },
    history: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
    stale: { type: Boolean, default: false }
  },
  emits: ['open', 'inspect-issue'],
  computed: {
    score() {
      return Math.max(0, Math.min(100, Number(this.healthData?.score || 0)))
    },
    statusTone() {
      return getHealthTone(this.score)
    },
    statusWord() {
      return getHealthStatusWord(this.score)
    },
    statusLabel() {
      return String(this.healthData?.overall_status || 'unknown').toUpperCase()
    },
    ringColor() {
      return getHealthColor(this.score)
    },
    outerProgressStyle() {
      const radius = 64
      const circumference = 2 * Math.PI * radius
      const progress = (this.score / 100) * circumference
      return {
        strokeDasharray: `${progress} ${circumference}`
      }
    },
    categories() {
      const groups = {
        security: { label: 'Security', color: 'var(--dashboard-ring-error)', total: 0, count: 0 },
        integrity: { label: 'Integrity', color: 'var(--dashboard-ring-warn)', total: 0, count: 0 },
        availability: { label: 'Availability', color: 'var(--dashboard-ring-ok)', total: 0, count: 0 },
        performance: { label: 'Performance', color: 'var(--accent)', total: 0, count: 0 }
      }

      ;(this.healthData?.checks || []).forEach(check => {
        const name = String(check.name || '').toLowerCase()
        const bucket =
          /sudo|permission|apparmor|audit|network/.test(name) ? 'security' :
          /database|binary|file/.test(name) ? 'integrity' :
          /service|dependency/.test(name) ? 'availability' : 'performance'
        const value = check.status === 'healthy' ? 100 : check.status === 'warning' ? 60 : check.status === 'critical' ? 20 : 40
        groups[bucket].total += value
        groups[bucket].count += 1
      })

      return Object.entries(groups).map(([key, group]) => ({
        key,
        label: group.label,
        color: group.color,
        score: group.count ? Math.round(group.total / group.count) : this.score
      }))
    },
    categorySegments() {
      const gap = 7
      const totalSweep = 360 - gap * this.categories.length
      const segmentSweep = totalSweep / Math.max(this.categories.length, 1)
      return this.categories.map((category, index) => {
        const startAngle = -90 + index * (segmentSweep + gap)
        const endAngle = startAngle + (segmentSweep * category.score) / 100
        return {
          ...category,
          path: describeArc(90, 90, 48, startAngle, endAngle)
        }
      })
    },
    prioritizedIssues() {
      const order = { critical: 0, warning: 1, unknown: 2, healthy: 3 }
      return [...(this.healthData?.checks || [])]
        .filter(check => check.status === 'critical' || check.status === 'warning')
        .sort((left, right) => order[left.status] - order[right.status])
        .slice(0, 4)
    },
    relativeTimestamp() {
      const ts = this.healthData?.timestamp ? new Date(this.healthData.timestamp).getTime() : 0
      if (!ts) return 'unknown'
      const seconds = Math.max(0, Math.floor((Date.now() - ts) / 1000))
      if (seconds < 60) return `${seconds}s ago`
      if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
      return `${Math.floor(seconds / 3600)}h ago`
    },
    historyPoints() {
      return sparklinePoints(this.history)
    },
    historyLine() {
      return linePath(this.historyPoints)
    },
    historyArea() {
      return areaPath(this.historyPoints)
    },
    historyGradientId() {
      return `health-history-${this._.uid}`
    }
  },
  methods: {
    severityLabel(status) {
      return status === 'critical' ? 'Critical' : status === 'warning' ? 'Warn' : 'Info'
    }
  }
}
</script>

<style scoped>
.health-anchor {
  border-radius: 26px;
  border: 1px solid var(--dashboard-panel-border-strong);
  background: var(--dashboard-hero-bg);
  box-shadow: var(--shadow-lg);
  padding: 18px;
  position: relative;
  overflow: hidden;
  font-variant-numeric: tabular-nums;
}

.health-anchor::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 10% 0%, rgba(255, 255, 255, 0.06), transparent 40%);
  pointer-events: none;
}

.health-anchor__header,
.health-anchor__body {
  position: relative;
  z-index: 1;
}

.health-anchor__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.health-anchor__eyebrow {
  color: var(--text-tertiary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 700;
}

.health-anchor__title {
  margin: 4px 0 0;
  font-size: 20px;
  color: var(--text-primary);
}

.health-anchor__body,
.health-anchor__loading {
  display: grid;
  grid-template-columns: minmax(164px, 184px) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.health-anchor__meter {
  border: 0;
  background: transparent;
  padding: 0;
  position: relative;
  width: 180px;
  height: 180px;
  margin: 0 auto;
}

.health-anchor__svg {
  width: 180px;
  height: 180px;
  transform: rotate(90deg);
}

.health-anchor__track,
.health-anchor__progress,
.health-anchor__segment {
  fill: none;
  stroke-linecap: round;
}

.health-anchor__track {
  stroke: var(--dashboard-ring-track);
  stroke-width: 16;
}

.health-anchor__progress {
  stroke: var(--dashboard-ring-ok);
  stroke-width: 16;
  transition: stroke-dasharray 0.5s ease, stroke 0.3s ease;
}

.health-anchor__progress--warn {
  stroke: var(--dashboard-ring-warn);
}

.health-anchor__progress--error {
  stroke: var(--dashboard-ring-error);
}

.health-anchor__segment {
  stroke-width: 8;
  opacity: 0.9;
}

.health-anchor__center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.health-anchor__score {
  font-size: 36px;
  line-height: 1;
  color: var(--text-primary);
  font-weight: 700;
}

.health-anchor__suffix,
.health-anchor__status {
  color: var(--text-secondary);
}

.health-anchor__suffix {
  font-size: 13px;
}

.health-anchor__status {
  margin-top: 6px;
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.health-anchor__content {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.health-anchor__summary {
  margin: 0;
  color: var(--text-primary);
  font-size: 15px;
  line-height: 1.5;
}

.health-anchor__meta-row,
.health-anchor__issues-head,
.health-anchor__issue-title-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.health-anchor__meta-row,
.health-anchor__issues-head,
.health-anchor__history-label,
.health-anchor__empty,
.health-anchor__issue-message,
.health-anchor__posture-label {
  color: var(--text-tertiary);
  font-size: 11px;
}

.health-anchor__history svg {
  width: 100%;
  height: 42px;
  margin-top: 6px;
}

.health-anchor__history-line {
  fill: none;
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.health-anchor__history-empty {
  stroke: var(--border-default);
  stroke-width: 1;
  stroke-dasharray: 4 5;
}

.health-anchor__issues {
  padding-top: 6px;
  border-top: 1px solid var(--dashboard-panel-border);
}

.health-anchor__issue-list {
  display: flex;
  flex-direction: column;
  margin-top: 6px;
}

.health-anchor__issue-row {
  display: grid;
  grid-template-columns: 3px 64px 1fr auto;
  grid-template-rows: auto auto;
  column-gap: 8px;
  padding: 8px 12px 8px 0;
  border-bottom: 1px solid var(--border-subtle);
  background: transparent;
  overflow: hidden;
  max-height: 56px;
}

.health-anchor__issue-accent-bar {
  grid-row: 1 / span 2;
  width: 3px;
  height: 100%;
  border-radius: 0;
  background: var(--state-warn);
}

.health-anchor__issue-row.is-critical .health-anchor__issue-accent-bar {
  background: var(--state-error);
}

.health-anchor__issue-row:first-child {
  border-top: 1px solid var(--border-subtle);
}

.health-anchor__issue-badge {
  grid-column: 2;
  grid-row: 1;
  align-self: start;
  justify-self: start;
  margin-top: 2px;
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.health-anchor__issue-badge--critical {
  background: var(--state-error-bg);
  color: var(--state-error-fg);
}

.health-anchor__issue-badge--warning {
  background: var(--state-warn-bg);
  color: var(--state-warn-fg);
}

.health-anchor__issue-content {
  grid-column: 3;
  grid-row: 1 / span 2;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}

.health-anchor__issue-title {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.health-anchor__issue-message {
  color: var(--text-secondary);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}

.health-anchor__issue-actions {
  grid-column: 4;
  grid-row: 1 / span 2;
  display: flex;
  gap: 4px;
  align-items: center;
  flex-shrink: 0;
}

.health-anchor__btn-fix {
  height: 22px;
  padding: 0 8px;
  font-size: 11px;
}

.health-anchor__btn-arrow {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm, 6px);
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.health-anchor__btn-arrow:hover {
  background: var(--surface-hover, var(--surface-3));
  color: var(--text-primary);
}

.health-anchor__loading-ring,
.health-anchor__loading-lines span {
  background: linear-gradient(90deg, rgba(138, 164, 200, 0.14) 25%, rgba(138, 164, 200, 0.28) 50%, rgba(138, 164, 200, 0.14) 75%);
  background-size: 200% 100%;
  animation: health-shimmer 1.4s linear infinite;
}

.health-anchor__loading-ring {
  width: 180px;
  height: 180px;
  border-radius: 999px;
}

.health-anchor__loading-lines {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.health-anchor__loading-lines span {
  height: 14px;
  border-radius: 8px;
}

.health-anchor__loading-lines span:first-child {
  width: 84%;
  height: 24px;
}

.health-anchor__loading-lines span:last-child {
  width: 60%;
}

@keyframes health-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

@media (prefers-reduced-motion: reduce) {
  .health-anchor__progress,
  .health-anchor__loading-ring,
  .health-anchor__loading-lines span {
    transition: none;
    animation: none;
  }
}

@media (max-width: 960px) {
  .health-anchor__body,
  .health-anchor__loading {
    grid-template-columns: 1fr;
  }
}
</style>