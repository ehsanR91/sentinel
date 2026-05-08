<template>
  <select
    :id="inputId"
    :value="modelValue"
    class="select-control sc-focus-ring"
    :class="{ 'is-error': error }"
    :disabled="disabled"
    @change="$emit('update:modelValue', $event.target.value)"
    @blur="$emit('blur')"
  >
    <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
  </select>
</template>

<script>
export default {
  name: 'SelectField',
  props: {
    modelValue: { type: [String, Number], default: '' },
    inputId: { type: String, required: true },
    options: { type: Array, default: () => [] },
    error: { type: String, default: '' },
    disabled: { type: Boolean, default: false }
  },
  emits: ['update:modelValue', 'blur']
}
</script>

<style scoped>
.select-control {
  width: 100%;
  min-height: 44px;
  padding: 0.75rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-1);
  color: var(--text-primary);
}

.select-control.is-error {
  border-color: var(--danger);
}
</style>