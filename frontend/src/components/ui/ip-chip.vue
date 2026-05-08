<template>
  <button type="button" class="sc-chip sc-chip--interactive sc-focus-ring font-mono" :title="tooltip" @click="$emit('click')">
    <i class="mdi mdi-lan-connect"></i>
    <span>{{ ip }}</span>
  </button>
</template>

<script>
export default {
  name: 'IpChip',
  props: {
    ip: { type: String, required: true },
    tooltip: { type: String, default: '' }
  },
  emits: ['click']
}
</script>