<template>
  <textarea
    :id="inputId"
    :value="modelValue"
    :placeholder="placeholder"
    :rows="rows"
    :disabled="disabled"
    :readonly="readonly"
    class="textarea-control sc-focus-ring"
    :class="{ 'is-error': error }"
    @input="$emit('update:modelValue', $event.target.value)"
    @blur="$emit('blur')"
  />
</template>

<script>
export default {
  name: 'TextAreaField',
  props: {
    modelValue: { type: String, default: '' },
    inputId: { type: String, required: true },
    placeholder: { type: String, default: '' },
    rows: { type: Number, default: 4 },
    error: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    readonly: { type: Boolean, default: false }
  },
  emits: ['update:modelValue', 'blur']
}
</script>

<style scoped>
.textarea-control {
  width: 100%;
  padding: 0.75rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-1);
  color: var(--text-primary);
  resize: vertical;
}

.textarea-control.is-error {
  border-color: var(--danger);
}
</style>