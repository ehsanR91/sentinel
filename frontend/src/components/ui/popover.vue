<template>
  <div ref="root" class="sc-popover-root" :class="rootClass">
    <slot
      name="trigger"
      :open="modelValue"
      :toggle="toggle"
      :close="close"
      :trigger-ref="setTriggerRef"
      :trigger-attrs="triggerAttrs"
    />

    <Teleport to="body">
      <transition :name="prefersReducedMotion ? '' : transitionName">
        <div v-if="modelValue" class="sc-popover-layer" :style="layerStyle">
          <div
            v-if="isSheet"
            class="sc-popover-backdrop"
            @click="close('backdrop')"
          ></div>
          <section
            :id="panelId"
            ref="panel"
            class="sc-popover-panel sc-surface"
            :class="panelClasses"
            :style="panelStyle"
            role="dialog"
            aria-modal="true"
            :aria-labelledby="headingId"
            @keydown="onPanelKeydown"
          >
            <div v-if="isSheet" class="sc-popover-sheet-handle" aria-hidden="true"></div>
            <header v-if="title || $slots.header" class="sc-popover-header">
              <slot name="header">
                <div>
                  <div v-if="subtitle" class="sc-popover-subtitle">{{ subtitle }}</div>
                  <h3 :id="headingId" class="sc-popover-title">{{ title }}</h3>
                </div>
                <button
                  type="button"
                  class="sc-popover-close sc-focus-ring"
                  aria-label="Close"
                  @click="close('button')"
                >
                  <i class="mdi mdi-close" aria-hidden="true"></i>
                </button>
              </slot>
            </header>
            <div class="sc-popover-body" :class="bodyClass">
              <slot :close="close" :is-sheet="isSheet" />
            </div>
            <footer v-if="$slots.footer" class="sc-popover-footer">
              <slot name="footer" :close="close" />
            </footer>
          </section>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script>
const OPEN_EVENT = 'sentinel:popover-open'
const CLOSE_EVENT = 'sentinel:popover-close'
const FOCUSABLE = [
  'a[href]',
  'area[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])'
].join(', ')

let nextId = 0

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

export default {
  name: 'Popover',
  props: {
    modelValue: { type: Boolean, default: false },
    title: { type: String, default: '' },
    subtitle: { type: String, default: '' },
    placement: { type: String, default: 'bottom-end' },
    width: { type: [Number, String], default: 360 },
    minWidth: { type: [Number, String], default: 280 },
    maxWidth: { type: [Number, String], default: 420 },
    offset: { type: Number, default: 10 },
    sheetBreakpoint: { type: Number, default: 640 },
    rootClass: { type: [String, Array, Object], default: '' },
    panelClass: { type: [String, Array, Object], default: '' },
    bodyClass: { type: [String, Array, Object], default: '' },
    closeOnOutside: { type: Boolean, default: true },
    closeOnEscape: { type: Boolean, default: true },
    closeOnSelect: { type: Boolean, default: false },
    restoreFocus: { type: Boolean, default: true }
  },
  emits: ['update:modelValue', 'after-open', 'after-close'],
  data() {
    return {
      localId: `sc-popover-${nextId++}`,
      headingId: `sc-popover-heading-${nextId++}`,
      panelId: `sc-popover-panel-${nextId++}`,
      triggerEl: null,
      position: { top: 0, left: 0 },
      prefersReducedMotion: false,
      viewportWidth: typeof window === 'undefined' ? 1440 : window.innerWidth,
      viewportScale: typeof window === 'undefined' ? 1 : (window.visualViewport?.scale || 1),
      restoreTarget: null
    }
  },
  computed: {
    triggerAttrs() {
      return {
        'aria-expanded': String(this.modelValue),
        'aria-haspopup': 'dialog',
        'aria-controls': this.panelId
      }
    },
    resolvedWidth() {
      return typeof this.width === 'number' ? `${this.width}px` : this.width
    },
    resolvedMinWidth() {
      return typeof this.minWidth === 'number' ? `${this.minWidth}px` : this.minWidth
    },
    resolvedMaxWidth() {
      return typeof this.maxWidth === 'number' ? `${this.maxWidth}px` : this.maxWidth
    },
    isSheet() {
      return this.viewportWidth <= this.sheetBreakpoint || this.viewportScale >= 1.75
    },
    panelClasses() {
      return [
        this.panelClass,
        { 'sc-popover-panel--sheet': this.isSheet }
      ]
    },
    layerStyle() {
      return {
        zIndex: 1400
      }
    },
    panelStyle() {
      if (this.isSheet) {
        return {
          width: 'min(100vw, 100%)',
          maxWidth: '100vw'
        }
      }
      return {
        top: `${this.position.top}px`,
        left: `${this.position.left}px`,
        width: this.resolvedWidth,
        minWidth: this.resolvedMinWidth,
        maxWidth: this.resolvedMaxWidth
      }
    },
    transitionName() {
      return this.isSheet ? 'sc-sheet' : 'sc-popover'
    }
  },
  watch: {
    modelValue(value) {
      if (value) {
        this.onOpen()
      } else {
        this.onClose()
      }
    }
  },
  mounted() {
    this.prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.addEventListener(OPEN_EVENT, this.onPeerOpen)
    window.addEventListener(CLOSE_EVENT, this.onPeerClose)
    window.addEventListener('resize', this.reposition, { passive: true })
    window.visualViewport?.addEventListener('resize', this.onViewportResize, { passive: true })
    if (this.modelValue) {
      this.onOpen()
    }
  },
  beforeUnmount() {
    window.removeEventListener(OPEN_EVENT, this.onPeerOpen)
    window.removeEventListener(CLOSE_EVENT, this.onPeerClose)
    window.removeEventListener('resize', this.reposition)
    window.visualViewport?.removeEventListener('resize', this.onViewportResize)
    document.removeEventListener('pointerdown', this.onDocumentPointerDown, true)
    document.removeEventListener('focusin', this.onDocumentFocusIn, true)
  },
  methods: {
    setTriggerRef(element) {
      this.triggerEl = element
    },
    emitOpen(value) {
      this.$emit('update:modelValue', value)
    },
    toggle() {
      this.modelValue ? this.close('toggle') : this.open()
    },
    open() {
      this.emitOpen(true)
    },
    close(reason = 'programmatic') {
      if (!this.modelValue) return
      this.emitOpen(false)
      window.dispatchEvent(new CustomEvent(CLOSE_EVENT, { detail: { id: this.localId, reason } }))
    },
    closeAfterAction() {
      if (this.closeOnSelect) {
        this.close('select')
      }
    },
    onOpen() {
      this.restoreTarget = document.activeElement instanceof HTMLElement ? document.activeElement : this.triggerEl
      this.reposition()
      window.dispatchEvent(new CustomEvent(OPEN_EVENT, { detail: { id: this.localId } }))
      document.addEventListener('pointerdown', this.onDocumentPointerDown, true)
      document.addEventListener('focusin', this.onDocumentFocusIn, true)
      this.$nextTick(() => {
        const focusable = this.getFocusableElements()
        focusable[0]?.focus()
        this.$emit('after-open')
      })
    },
    onClose() {
      document.removeEventListener('pointerdown', this.onDocumentPointerDown, true)
      document.removeEventListener('focusin', this.onDocumentFocusIn, true)
      if (this.restoreFocus && this.restoreTarget?.focus) {
        this.restoreTarget.focus()
      }
      this.$emit('after-close')
    },
    onPeerOpen(event) {
      if (event.detail?.id !== this.localId && this.modelValue) {
        this.emitOpen(false)
      }
    },
    onPeerClose() {},
    onViewportResize() {
      this.viewportWidth = window.visualViewport?.width || window.innerWidth
      this.viewportScale = window.visualViewport?.scale || 1
      this.reposition()
    },
    reposition() {
      this.viewportWidth = window.visualViewport?.width || window.innerWidth
      this.viewportScale = window.visualViewport?.scale || 1
      if (this.isSheet || !this.triggerEl) return
      const rect = this.triggerEl.getBoundingClientRect()
      const viewportWidth = window.innerWidth
      const panelWidth = Number.parseInt(this.resolvedWidth, 10) || 360
      let left = rect.right - panelWidth
      if (this.placement.includes('start')) {
        left = rect.left
      } else if (this.placement.includes('center')) {
        left = rect.left + (rect.width / 2) - (panelWidth / 2)
      }
      left = clamp(left, 12, viewportWidth - panelWidth - 12)
      const top = rect.bottom + this.offset
      this.position = { top, left }
    },
    onDocumentPointerDown(event) {
      const target = event.target
      if (!this.closeOnOutside) return
      if (this.$refs.panel?.contains(target) || this.triggerEl?.contains(target)) {
        return
      }
      this.close('outside')
    },
    onDocumentFocusIn(event) {
      if (!this.modelValue || this.isSheet) return
      const target = event.target
      if (this.$refs.panel?.contains(target) || this.triggerEl?.contains(target)) {
        return
      }
      const focusable = this.getFocusableElements()
      focusable[0]?.focus()
    },
    onPanelKeydown(event) {
      if (this.closeOnEscape && event.key === 'Escape') {
        event.preventDefault()
        this.close('escape')
        return
      }
      if (event.key !== 'Tab') return
      const focusable = this.getFocusableElements()
      if (!focusable.length) return
      const currentIndex = focusable.indexOf(document.activeElement)
      const nextIndex = event.shiftKey
        ? (currentIndex <= 0 ? focusable.length - 1 : currentIndex - 1)
        : (currentIndex === focusable.length - 1 ? 0 : currentIndex + 1)
      event.preventDefault()
      focusable[nextIndex]?.focus()
    },
    getFocusableElements() {
      return Array.from(this.$refs.panel?.querySelectorAll(FOCUSABLE) || [])
        .filter(element => !element.hasAttribute('disabled'))
    }
  }
}
</script>

<style scoped>
.sc-popover-layer {
  position: fixed;
  inset: 0;
  pointer-events: none;
}

.sc-popover-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(5, 9, 18, 0.42);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  pointer-events: auto;
}

.sc-popover-panel {
  position: absolute;
  pointer-events: auto;
  overflow: hidden;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  box-shadow: 0 24px 60px color-mix(in srgb, #000 28%, transparent);
}

.sc-popover-panel--sheet {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  top: auto;
  border-radius: 22px 22px 0 0;
  padding-bottom: calc(var(--space-16) + env(safe-area-inset-bottom));
}

.sc-popover-sheet-handle {
  width: 52px;
  height: 5px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--text-tertiary) 30%, transparent);
  margin: 10px auto 2px;
}

.sc-popover-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-10);
  padding: var(--space-16) var(--space-16) var(--space-10);
}

.sc-popover-title {
  margin: 0;
  font-size: var(--font-size-16);
  font-weight: 600;
  color: var(--text-primary);
}

.sc-popover-subtitle {
  margin-bottom: var(--space-4);
  font-size: var(--font-size-11);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}

.sc-popover-close {
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
}

.sc-popover-body {
  padding: 0 var(--space-16) var(--space-16);
}

.sc-popover-footer {
  padding: 0 var(--space-16) var(--space-16);
}

.sc-popover-enter-active,
.sc-popover-leave-active,
.sc-sheet-enter-active,
.sc-sheet-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.sc-popover-enter-from,
.sc-popover-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.985);
}

.sc-sheet-enter-from,
.sc-sheet-leave-to {
  opacity: 0;
  transform: translateY(16px);
}
</style>