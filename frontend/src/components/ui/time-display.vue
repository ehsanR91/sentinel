<template>
  <div class="d-flex flex-column" :title="formatted.title">
    <span>{{ formatted.primary }}</span>
    <span v-if="formatted.secondary" class="font-mono text-muted text-xs">{{ formatted.secondary }}</span>
  </div>
</template>

<script>
import { formatTimestamp } from '@/utils/formatters'

export default {
  name: 'TimeDisplay',
  props: {
    value: { type: [String, Number, Date], required: true },
    mode: { type: String, default: '' }
  },
  computed: {
    formatted () {
      return formatTimestamp(this.value, { mode: this.mode || undefined })
    }
  }
}
</script>