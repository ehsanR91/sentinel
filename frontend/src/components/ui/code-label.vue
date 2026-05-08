<template>
  <span :title="title" class="d-inline-flex align-items-center gap-2">
    <i :class="meta.icon" aria-hidden="true"></i>
    <span>{{ meta.label }}</span>
  </span>
</template>

<script>
import { getReasonMeta } from '@/utils/formatters'

export default {
  name: 'CodeLabel',
  props: {
    code: { type: String, default: '' }
  },
  computed: {
    meta () {
      return getReasonMeta(this.code)
    },
    title () {
      return `${this.meta.description}\nCode: ${this.meta.code || 'success'}`
    }
  }
}
</script>