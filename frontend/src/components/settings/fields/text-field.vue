<template>
  <div class="field-root">
    <input
      :id="inputId"
      :value="modelValue"
      :type="type"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      class="field-control sc-focus-ring"
      :class="{ 'is-error': error, 'is-readonly': readonly }"
      @input="$emit('update:modelValue', $event.target.value)"
      @blur="$emit('blur')"
    >
  </div>
</template>

<script>
export default {
  name: 'TextField',
  props: {
    modelValue: { type: [String, Number], default: '' },
    inputId: { type: String, required: true },
    type: { type: String, default: 'text' },
    placeholder: { type: String, default: '' },
    error: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    readonly: { type: Boolean, default: false }
  },
  emits: ['update:modelValue', 'blur']
}
</script>

<style scoped>
.field-control {
  width: 100%;
  min-height: 44px;
  padding: 0.75rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-1);
  color: var(--text-primary);
}

.field-control.is-error {
  border-color: var(--danger);
}

.field-control.is-readonly {
  background: color-mix(in srgb, var(--surface-1) 78%, transparent);
}
</style>