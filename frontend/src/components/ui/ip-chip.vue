<template>
  <Tooltip :label="ip" :description="tooltipDescription" :variant="tooltipDescription ? 'rich' : 'default'" as-child>
    <button type="button" class="sc-chip sc-chip--interactive sc-focus-ring font-mono" @click="$emit('click')">
      <i class="mdi mdi-lan-connect"></i>
      <span>{{ ip }}</span>
    </button>
  </Tooltip>
</template>

<script>
import Tooltip from './tooltip.vue'

export default {
  name: 'IpChip',
  components: { Tooltip },
  props: {
    ip: { type: String, required: true },
    tooltip: { type: String, default: '' }
  },
  emits: ['click'],
  computed: {
    tooltipDescription () {
      const trimmed = String(this.tooltip || '').trim()
      if (!trimmed || trimmed === this.ip) return ''
      return trimmed.startsWith(`${this.ip}\n`) ? trimmed.slice(this.ip.length + 1) : trimmed
    }
  }
}
</script>