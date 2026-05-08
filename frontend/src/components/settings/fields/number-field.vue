<template>
  <TextField
    :input-id="inputId"
    type="number"
    :model-value="modelValue"
    :placeholder="placeholder"
    :error="error"
    :disabled="disabled"
    :readonly="readonly"
    @update:modelValue="$emit('update:modelValue', Number($event))"
    @blur="$emit('blur')"
  />
</template>

<script>
import TextField from './text-field.vue'

export default {
  name: 'NumberField',
  components: { TextField },
  props: {
    modelValue: { type: [String, Number], default: '' },
    inputId: { type: String, required: true },
    placeholder: { type: String, default: '' },
    error: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    readonly: { type: Boolean, default: false }
  },
  emits: ['update:modelValue', 'blur']
}
</script>