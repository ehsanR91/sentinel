<template>
  <div class="sc-empty-state">
    <div class="sc-empty-state__icon"><i :class="icon"></i></div>
    <div class="sc-empty-state__title">{{ title }}</div>
    <div class="sc-empty-state__text">{{ description }}</div>
    <div v-if="$slots.actions"><slot name="actions" /></div>
  </div>
</template>

<script>
export default {
  name: 'EmptyState',
  props: {
    icon: { type: String, default: 'mdi mdi-database-off-outline' },
    title: { type: String, required: true },
    description: { type: String, required: true }
  }
}
</script>