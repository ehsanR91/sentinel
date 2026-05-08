<template>
  <div class="pin-field" role="group" :aria-label="label">
    <input
      v-for="(digit, index) in digits"
      :key="index"
      :ref="el => assignRef(el, index)"
      :value="digit"
      class="pin-field__cell sc-focus-ring"
      inputmode="numeric"
      pattern="[0-9]*"
      maxlength="1"
      :aria-label="`PIN digit ${index + 1} of ${length}`"
      @input="onInput(index, $event)"
      @keydown.backspace="onBackspace(index, $event)"
      @paste.prevent="onPaste"
    >
  </div>
</template>

<script>
export default {
  name: 'PinField',
  props: {
    modelValue: { type: String, default: '' },
    length: { type: Number, default: 6 },
    label: { type: String, default: 'Security PIN' }
  },
  emits: ['update:modelValue'],
  data() {
    return {
      refs: []
    }
  },
  computed: {
    digits() {
      return Array.from({ length: this.length }, (_, index) => this.modelValue[index] || '')
    }
  },
  methods: {
    assignRef(el, index) {
      if (el) this.refs[index] = el
    },
    updateAt(index, value) {
      const next = this.digits.slice()
      next[index] = value
      this.$emit('update:modelValue', next.join(''))
    },
    onInput(index, event) {
      const value = String(event.target.value || '').replace(/\D/g, '').slice(-1)
      this.updateAt(index, value)
      if (value && index < this.length - 1) {
        this.refs[index + 1]?.focus()
      }
    },
    onBackspace(index, event) {
      if (this.digits[index]) {
        this.updateAt(index, '')
        return
      }
      if (index > 0) {
        event.preventDefault()
        this.refs[index - 1]?.focus()
        this.updateAt(index - 1, '')
      }
    },
    onPaste(event) {
      const value = (event.clipboardData?.getData('text') || '').replace(/\D/g, '').slice(0, this.length)
      this.$emit('update:modelValue', value)
      this.$nextTick(() => {
        this.refs[Math.min(value.length, this.length - 1)]?.focus()
      })
    }
  }
}
</script>

<style scoped>
.pin-field {
  display: flex;
  gap: var(--space-8);
}

.pin-field__cell {
  width: 48px;
  min-height: 52px;
  text-align: center;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-1);
  color: var(--text-primary);
  font-size: var(--font-size-20);
  font-family: var(--font-mono);
}

@media (max-width: 768px) {
  .pin-field__cell {
    width: 44px;
  }
}
</style>