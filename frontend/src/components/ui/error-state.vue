<template>
  <div class="sc-inline-error" role="alert">
    <div class="d-flex gap-3 align-items-start">
      <i class="mdi mdi-alert-circle-outline" aria-hidden="true"></i>
      <div>
        <div class="fw-600 mb-1">{{ title }}</div>
        <div>{{ description }}</div>
      </div>
    </div>
    <div v-if="$slots.actions"><slot name="actions" /></div>
  </div>
</template>

<script>
export default {
  name: 'ErrorState',
  props: {
    title: { type: String, required: true },
    description: { type: String, required: true }
  }
}
</script>