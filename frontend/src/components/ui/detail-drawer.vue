<template>
  <teleport to="body">
    <div v-if="modelValue">
      <div class="sc-drawer-backdrop" @click="$emit('update:modelValue', false)"></div>
      <aside class="sc-drawer" aria-modal="true" role="dialog">
        <div class="sc-drawer__header d-flex align-items-start justify-content-between gap-3">
          <div>
            <div class="text-sm text-secondary mb-1" v-if="subtitle">{{ subtitle }}</div>
            <h5 class="mb-0">{{ title }}</h5>
          </div>
          <div class="d-flex gap-2">
            <slot name="nav" />
            <button type="button" class="sc-button sc-button--ghost sc-button--sm sc-button--icon-only" aria-label="Close details" @click="$emit('update:modelValue', false)">
              <i class="mdi mdi-close"></i>
            </button>
          </div>
        </div>
        <div class="sc-drawer__body">
          <slot />
        </div>
        <div v-if="$slots.footer" class="sc-drawer__footer">
          <slot name="footer" />
        </div>
      </aside>
    </div>
  </teleport>
</template>

<script>
export default {
  name: 'DetailDrawer',
  props: {
    modelValue: { type: Boolean, default: false },
    title: { type: String, required: true },
    subtitle: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  mounted () {
    window.addEventListener('keydown', this.onKeyDown)
  },
  beforeUnmount () {
    window.removeEventListener('keydown', this.onKeyDown)
  },
  methods: {
    onKeyDown (event) {
      if (!this.modelValue) return
      if (event.key === 'Escape') {
        this.$emit('update:modelValue', false)
      }
      if (event.key === 'ArrowLeft') {
        this.$emit('navigate', -1)
      }
      if (event.key === 'ArrowRight') {
        this.$emit('navigate', 1)
      }
    }
  },
  emits: ['update:modelValue', 'navigate']
}
</script>