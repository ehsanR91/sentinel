<template>
  <button
    :id="inputId"
    type="button"
    class="toggle-switch sc-focus-ring"
    :class="{ 'is-on': modelValue, 'is-disabled': disabled }"
    role="switch"
    :aria-checked="String(modelValue)"
    :disabled="disabled"
    @click="$emit('update:modelValue', !modelValue)"
  >
    <span class="toggle-switch__track">
      <span class="toggle-switch__thumb"></span>
    </span>
    <span v-if="label" class="toggle-switch__label">{{ label }}</span>
  </button>
</template>

<script>
export default {
  name: 'ToggleSwitch',
  props: {
    modelValue: { type: Boolean, default: false },
    inputId: { type: String, required: true },
    label: { type: String, default: '' },
    disabled: { type: Boolean, default: false }
  },
  emits: ['update:modelValue']
}
</script>

<style scoped>
.toggle-switch {
  display: inline-flex;
  align-items: center;
  gap: var(--space-10);
  border: 0;
  background: transparent;
  color: var(--text-primary);
}

.toggle-switch__track {
  position: relative;
  width: 46px;
  height: 28px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--surface-1) 72%, var(--border-subtle));
  transition: background 0.2s ease;
}

.toggle-switch__thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.22);
  transition: transform 0.2s ease;
}

.toggle-switch.is-on .toggle-switch__track {
  background: var(--accent);
}

.toggle-switch.is-on .toggle-switch__thumb {
  transform: translateX(18px);
}

.toggle-switch.is-disabled {
  opacity: 0.55;
}

.toggle-switch__label {
  font-size: var(--font-size-13);
}
</style>