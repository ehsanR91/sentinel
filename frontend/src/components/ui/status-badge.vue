<template>
  <Tooltip :label="label" :description="tooltipDescription" :status="normalizedState" :variant="tooltipDescription ? 'rich' : 'default'" as-child>
    <span class="sc-badge" :class="`sc-badge--${normalizedState}`">
      <i v-if="resolvedIcon" :class="resolvedIcon" aria-hidden="true"></i>
      <span>{{ label }}</span>
    </span>
  </Tooltip>
</template>

<script>
import Tooltip from './tooltip.vue'

const ICONS = {
  ok: 'mdi mdi-check-circle-outline',
  info: 'mdi mdi-information-outline',
  warn: 'mdi mdi-alert-outline',
  error: 'mdi mdi-alert-circle-outline',
  critical: 'mdi mdi-alert-octagon-outline',
  muted: 'mdi mdi-minus-circle-outline',
  pending: 'mdi mdi-timer-sand'
}

export default {
  name: 'StatusBadge',
  components: { Tooltip },
  props: {
    state: { type: String, default: 'muted' },
    label: { type: String, required: true },
    icon: { type: String, default: '' },
    title: { type: String, default: '' }
  },
  computed: {
    normalizedState () {
      return ['ok', 'info', 'warn', 'error', 'critical', 'muted', 'pending'].includes(this.state)
        ? this.state
        : 'muted'
    },
    resolvedIcon () {
      return this.icon || ICONS[this.normalizedState]
    },
    tooltipDescription () {
      const value = String(this.title || '').trim()
      if (!value || value === this.label) return ''
      return value
    }
  }
}
</script>