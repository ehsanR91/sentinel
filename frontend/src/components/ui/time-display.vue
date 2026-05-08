<template>
  <Tooltip :label="formatted.primary" :description="tooltipDescription" :variant="tooltipDescription ? 'rich' : 'default'" as-child>
    <div class="d-flex flex-column">
      <span>{{ formatted.primary }}</span>
      <span v-if="formatted.secondary" class="font-mono text-muted text-xs">{{ formatted.secondary }}</span>
    </div>
  </Tooltip>
</template>

<script>
import { formatTimestamp } from '@/utils/formatters'
import Tooltip from './tooltip.vue'

export default {
  name: 'TimeDisplay',
  components: { Tooltip },
  props: {
    value: { type: [String, Number, Date], required: true },
    mode: { type: String, default: '' }
  },
  computed: {
    formatted () {
      return formatTimestamp(this.value, { mode: this.mode || undefined })
    },
    tooltipDescription () {
      const parts = [this.formatted.title, this.formatted.secondary]
        .map(value => String(value || '').trim())
        .filter(Boolean)
      return parts.filter(value => value !== this.formatted.primary).join('\n')
    }
  }
}
</script>