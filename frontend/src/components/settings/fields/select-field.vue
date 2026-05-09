<template>
  <ScSelect
    :input-id="inputId"
    :model-value="modelValue"
    :options="options"
    :disabled="disabled"
    :class="{ 'is-error': error }"
    @update:model-value="$emit('update:modelValue', $event)"
    @blur="$emit('blur')"
  />
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