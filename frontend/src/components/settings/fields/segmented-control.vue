<template>
  <div class="segmented-control" role="radiogroup" :aria-label="label">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="segmented-control__item sc-focus-ring"
      :class="{ active: modelValue === option.value }"
      role="radio"
      :aria-checked="String(modelValue === option.value)"
      @click="$emit('update:modelValue', option.value)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<script>
export default {
  name: 'SegmentedControl',
  props: {
    modelValue: { type: [String, Number], default: '' },
    options: { type: Array, default: () => [] },
    label: { type: String, default: '' }
  },
  emits: ['update:modelValue']
}
</script>

<style scoped>
.segmented-control {
  display: inline-flex;
  padding: 4px;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  border: 1px solid var(--border-subtle);
}

.segmented-control__item {
  min-width: 88px;
  padding: 0.55rem 0.8rem;
  border: 0;
  border-radius: calc(var(--radius-md) - 4px);
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.segmented-control__item.active {
  background: var(--surface-2);
  color: var(--text-primary);
}
</style>