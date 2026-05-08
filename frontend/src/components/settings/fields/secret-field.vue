<template>
  <div class="secret-field">
    <TextField
      :input-id="inputId"
      :type="revealed ? 'text' : 'password'"
      :model-value="modelValue"
      :placeholder="placeholder"
      :error="error"
      :disabled="disabled"
      @update:modelValue="$emit('update:modelValue', $event)"
      @blur="$emit('blur')"
    />
    <div class="secret-field__actions">
      <button type="button" class="secret-field__action sc-focus-ring" @click="revealed = !revealed">
        <i class="mdi" :class="revealed ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"></i>
      </button>
      <button type="button" class="secret-field__action sc-focus-ring" @click="copyValue">
        <i class="mdi mdi-content-copy"></i>
      </button>
      <slot name="actions" />
    </div>
  </div>
</template>

<script>
import TextField from './text-field.vue'

export default {
  name: 'SecretField',
  components: { TextField },
  props: {
    modelValue: { type: String, default: '' },
    inputId: { type: String, required: true },
    placeholder: { type: String, default: '' },
    error: { type: String, default: '' },
    disabled: { type: Boolean, default: false }
  },
  emits: ['update:modelValue', 'blur'],
  data() {
    return {
      revealed: false
    }
  },
  methods: {
    copyValue() {
      navigator.clipboard.writeText(this.modelValue || '').catch(() => {})
    }
  }
}
</script>

<style scoped>
.secret-field {
  display: flex;
  align-items: stretch;
  gap: var(--space-8);
}

.secret-field__actions {
  display: inline-flex;
  gap: var(--space-8);
}

.secret-field__action {
  width: 44px;
  min-height: 44px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-1);
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .secret-field {
    flex-direction: column;
  }

  .secret-field__actions {
    justify-content: flex-end;
  }
}
</style>