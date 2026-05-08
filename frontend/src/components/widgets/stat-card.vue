<template>
  <component
    :is="clickable ? 'button' : 'div'"
    class="metric-card sc-surface sc-focus-ring"
    :class="{ 'metric-card--interactive': clickable }"
    :aria-label="`${label}: ${value}${sub ? ` (${sub})` : ''}`"
    :type="clickable ? 'button' : undefined"
    role="region"
    @click="clickable && $emit('click')"
  >
    <div class="metric-icon" :style="`background:${sanitizedIconBg};color:${sanitizedIconColor}`" aria-hidden="true">
      <i :class="sanitizedIcon"></i>
    </div>
    <div class="metric-body">
      <div class="metric-label">{{ label }}</div>
      <div class="metric-value" :class="valueClass">{{ value }}</div>
      <div v-if="sub" class="metric-sub">{{ sub }}</div>
      <div v-if="progress !== null" class="mt-2" role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100">
        <div class="progress">
          <div
            class="progress-bar"
            :class="progressClass"
            :style="`width:${progress}%`"
          ></div>
        </div>
      </div>
    </div>
  </component>
</template>

<script>
const ALLOWED_ICON_RE = /^mdi|ri|ti|fa|fas|far|fal|fab|glyphicon glyphicon-/
const COLOR_RE = /^(#([0-9a-fA-F]{3}){1,2}|rgba?\(\s*\d+\s*,\s*\d+\s*,\s*\d+(\s*,\s*[\d.]+)?\s*\)|hsla?\(\s*\d+\s*,\s*\d+%\s*,\s*\d+%(\s*,\s*[\d.]+)?\s*\)|[a-z-]+)$/i

export default {
  name: 'StatCard',
  props: {
    label: { type: String, required: true },
    value: { type: [String, Number], required: true },
    sub:   { type: String, default: null },
    icon:  { type: String, default: 'mdi mdi-information-outline' },
    iconColor: { type: String, default: '#4a9eff' },
    iconBg:    { type: String, default: 'rgba(74,158,255,0.12)' },
    progress:  { type: Number, default: null },
    clickable: { type: Boolean, default: false },
    tone: { type: String, default: 'default' }
  },
  emits: ['click'],
  computed: {
    sanitizedIcon() {
      return ALLOWED_ICON_RE.test(this.icon.trim()) ? this.icon.trim() : 'mdi mdi-information-outline'
    },
    sanitizedIconColor() {
      return COLOR_RE.test(this.iconColor.trim()) ? this.iconColor.trim() : '#4a9eff'
    },
    sanitizedIconBg() {
      return COLOR_RE.test(this.iconBg.trim()) ? this.iconBg.trim() : 'rgba(74,158,255,0.12)'
    },
    progressClass() {
      if (this.progress > 85) return 'bg-danger'
      if (this.progress > 65) return 'bg-warning'
      return 'bg-success'
    },
    valueClass() {
      return this.tone !== 'default' ? `metric-value--${this.tone}` : ''
    }
  }
}
</script>

<style scoped>
.metric-card {
  width: 100%;
  display: flex;
  align-items: center;
  gap: var(--space-16);
  padding: var(--space-16);
  border: 1px solid var(--border-default);
  background: var(--surface-1);
  text-align: left;
}

.metric-card--interactive {
  cursor: pointer;
}

.metric-card--interactive:hover {
  background: var(--surface-2);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.metric-icon {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-md);
  font-size: 20px;
  flex-shrink: 0;
}

.metric-body {
  min-width: 0;
}

.metric-label {
  color: var(--text-tertiary);
  font-size: var(--font-size-11);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.metric-value {
  color: var(--text-primary);
  font-size: 24px;
  line-height: 1.2;
  font-weight: 600;
}

.metric-value--warn {
  color: var(--state-warn-fg);
}

.metric-value--error,
.metric-value--critical {
  color: var(--state-error-fg);
}

.metric-sub {
  color: var(--text-secondary);
  font-size: var(--font-size-12);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 768px) {
  .metric-card {
    padding: var(--space-12);
    gap: var(--space-12);
  }
}
</style>
