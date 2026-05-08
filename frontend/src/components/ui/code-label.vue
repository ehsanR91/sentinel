<template>
  <Tooltip :label="meta.label" :description="tooltipDescription" variant="rich" as-child>
    <span class="d-inline-flex align-items-center gap-2">
      <i :class="meta.icon" aria-hidden="true"></i>
      <span>{{ meta.label }}</span>
    </span>
  </Tooltip>
</template>

<script>
import { getReasonMeta } from '@/utils/formatters'
import Tooltip from './tooltip.vue'

export default {
  name: 'CodeLabel',
  components: { Tooltip },
  props: {
    code: { type: String, default: '' }
  },
  computed: {
    meta () {
      return getReasonMeta(this.code)
    },
    tooltipDescription () {
      return `${this.meta.description}\nCode: ${this.meta.code || 'success'}`
    }
  }
}
</script>