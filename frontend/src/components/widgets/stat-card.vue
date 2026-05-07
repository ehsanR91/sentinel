<template>
  <div class="metric-card" :aria-label="`${label}: ${value}${sub ? ` (${sub})` : ''}`" role="region">
    <div class="metric-icon" :style="`background:${sanitizedIconBg}`" aria-hidden="true">
      <i :class="sanitizedIcon" :style="`color:${sanitizedIconColor}`"></i>
    </div>
    <div class="metric-body">
      <div class="metric-label">{{ label }}</div>
      <div class="metric-value">{{ value }}</div>
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
  </div>
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
    progress:  { type: Number, default: null }
  },
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
    }
  }
}
</script>
