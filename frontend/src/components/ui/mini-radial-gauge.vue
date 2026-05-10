<template>
  <div class="mini-radial-gauge" :style="wrapperStyle">
    <svg class="mini-radial-gauge__svg" viewBox="0 0 120 120" aria-hidden="true">
      <defs>
        <linearGradient :id="gradientId" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" :stop-color="color" stop-opacity="0.95" />
          <stop offset="100%" :stop-color="color" stop-opacity="0.68" />
        </linearGradient>
      </defs>
      <circle class="mini-radial-gauge__track" cx="60" cy="60" :r="radius" />
      <circle
        class="mini-radial-gauge__value"
        cx="60"
        cy="60"
        :r="radius"
        :stroke="`url(#${gradientId})`"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="dashOffset"
      />
    </svg>
    <div class="mini-radial-gauge__center">
      <strong>{{ valueLabel }}</strong>
      <small v-if="subtitle">{{ subtitle }}</small>
    </div>
  </div>
</template>

<script>
let gaugeId = 0

export default {
  name: 'MiniRadialGauge',
  props: {
    value: { type: Number, default: 0 },
    color: { type: String, default: '#4a9eff' },
    size: { type: Number, default: 160 },
    thickness: { type: Number, default: 10 },
    subtitle: { type: String, default: '' }
  },
  data() {
    gaugeId += 1
    return {
      gradientId: `mini-radial-gauge-${gaugeId}`
    }
  },
  computed: {
    safeValue() {
      const numeric = Number(this.value || 0)
      if (!Number.isFinite(numeric)) return 0
      return Math.min(100, Math.max(0, numeric))
    },
    radius() {
      return 60 - Math.max(4, this.thickness)
    },
    circumference() {
      return 2 * Math.PI * this.radius
    },
    dashOffset() {
      return this.circumference * (1 - this.safeValue / 100)
    },
    valueLabel() {
      return `${Math.round(this.safeValue)}%`
    },
    wrapperStyle() {
      return {
        width: `${this.size}px`,
        height: `${this.size}px`,
        '--gauge-track': 'rgba(30, 45, 74, 0.95)',
        '--gauge-thickness': `${this.thickness}px`
      }
    }
  }
}
</script>

<style scoped>
.mini-radial-gauge {
  position: relative;
  display: inline-grid;
  place-items: center;
  border-radius: 50%;
}

.mini-radial-gauge__svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
  filter: drop-shadow(0 8px 16px rgba(8, 17, 31, 0.24));
}

.mini-radial-gauge__track,
.mini-radial-gauge__value {
  fill: none;
  stroke-width: var(--gauge-thickness);
}

.mini-radial-gauge__track {
  stroke: var(--gauge-track);
}

.mini-radial-gauge__value {
  stroke-linecap: round;
  transition: stroke-dashoffset 180ms ease;
}

.mini-radial-gauge__center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.mini-radial-gauge__center strong {
  color: #e2ecff;
  font-size: 22px;
  font-weight: 700;
  line-height: 1;
}

.mini-radial-gauge__center small {
  margin-top: 6px;
  color: var(--sc-text-muted);
  font-size: 11px;
}
</style>