<template>
  <RekaTooltipRoot
    :open="open"
    :delay-duration="resolvedDelay"
    :disabled="tooltipDisabled"
    :disable-hoverable-content="true"
    :ignore-non-keyboard-focus="true"
    @update:open="handleOpenChange"
  >
    <RekaTooltipTrigger
      :as-child="asChild"
      @pointerdown.capture="handlePointerDown"
      @pointerup.capture="handlePointerUp"
      @pointercancel.capture="dismiss('pointercancel')"
      @pointerleave.capture="handlePointerLeave"
      @click.capture="dismiss('click')"
      @keydown.esc.capture.stop.prevent="dismiss('escape-key')"
    >
      <slot />
    </RekaTooltipTrigger>

    <RekaTooltipPortal to="#sc-tooltip-root">
      <RekaTooltipContent
        v-if="hasRenderableContent"
        class="sc-tooltip"
        :class="[`sc-tooltip--${resolvedVariant}`, contentClass]"
        :side="resolvedSide"
        :align="resolvedAlign"
        :side-offset="resolvedOffset"
        :collision-padding="12"
        :avoid-collisions="true"
        sticky="partial"
        :hide-when-detached="true"
        update-position-strategy="always"
      >
        <slot name="content" :dismiss="dismiss">
          <TooltipContentBody>
            <div class="sc-tooltip__header" :class="{ 'sc-tooltip__header--stacked': showRichFooter }">
              <TooltipLabel v-if="label">{{ label }}</TooltipLabel>
              <div v-if="showInlineMeta" class="sc-tooltip__inline-meta">
                <TooltipBadge v-if="badge">{{ badge }}</TooltipBadge>
                <TooltipBadge v-if="status">{{ status }}</TooltipBadge>
                <TooltipShortcut v-if="shortcut && !showRichFooter" :shortcut="shortcut" />
              </div>
            </div>
            <TooltipDescription v-if="description">{{ description }}</TooltipDescription>
            <div v-if="showRichFooter" class="sc-tooltip__footer">
              <div class="sc-tooltip__footer-badges">
                <TooltipBadge v-if="badge">{{ badge }}</TooltipBadge>
                <TooltipBadge v-if="status">{{ status }}</TooltipBadge>
              </div>
              <TooltipShortcut v-if="shortcut" :shortcut="shortcut" />
            </div>
          </TooltipContentBody>
        </slot>
        <RekaTooltipArrow v-if="withArrow" class="sc-tooltip__arrow" :width="6" :height="6" />
      </RekaTooltipContent>
    </RekaTooltipPortal>
  </RekaTooltipRoot>
</template>

<script>
import {
  TooltipArrow as RekaTooltipArrow,
  TooltipContent as RekaTooltipContent,
  TooltipPortal as RekaTooltipPortal,
  TooltipRoot as RekaTooltipRoot,
  TooltipTrigger as RekaTooltipTrigger
} from 'reka-ui'
import TooltipBadge from './tooltip-badge.vue'
import TooltipContentBody from './tooltip-content.vue'
import TooltipDescription from './tooltip-description.vue'
import TooltipLabel from './tooltip-label.vue'
import TooltipShortcut from './tooltip-shortcut.vue'

const TOOLTIP_OPEN_EVENT = 'sentinel:tooltip-open'

let nextTooltipId = 0

function coarsePointer() {
  return typeof window !== 'undefined' && window.matchMedia('(pointer: coarse)').matches
}

export default {
  name: 'Tooltip',
  components: {
    RekaTooltipArrow,
    RekaTooltipContent,
    RekaTooltipPortal,
    RekaTooltipRoot,
    RekaTooltipTrigger,
    TooltipBadge,
    TooltipContentBody,
    TooltipDescription,
    TooltipLabel,
    TooltipShortcut
  },
  props: {
    label: { type: String, default: '' },
    description: { type: String, default: '' },
    shortcut: { type: [String, Array], default: '' },
    placement: { type: String, default: 'top' },
    delay: { type: Number, default: null },
    variant: { type: String, default: 'default' },
    asChild: { type: Boolean, default: true },
    disabled: { type: Boolean, default: false },
    withArrow: { type: Boolean, default: false },
    badge: { type: String, default: '' },
    status: { type: String, default: '' },
    offset: { type: Number, default: 7 },
    align: { type: String, default: 'center' },
    contentClass: { type: [String, Array, Object], default: '' }
  },
  data () {
    return {
      open: false,
      longPressTimer: null,
      longPressActive: false,
      longPressHideTimer: null,
      localId: `sc-tooltip-${nextTooltipId++}`,
      windowBlurHandler: null,
      peerOpenHandler: null,
      focusTrapHandler: null
    }
  },
  computed: {
    hasRenderableContent () {
      return Boolean(this.label || this.description || this.badge || this.status || (Array.isArray(this.shortcut) ? this.shortcut.length : this.shortcut))
    },
    tooltipDisabled () {
      return this.disabled || !this.hasRenderableContent
    },
    resolvedVariant () {
      if (['default', 'rich', 'inline'].includes(this.variant)) return this.variant
      return this.description ? 'rich' : 'default'
    },
    resolvedDelay () {
      return typeof this.delay === 'number' ? this.delay : undefined
    },
    resolvedSide () {
      return ['top', 'right', 'bottom', 'left'].includes(this.placement) ? this.placement : 'top'
    },
    resolvedAlign () {
      return ['start', 'center', 'end'].includes(this.align) ? this.align : 'center'
    },
    resolvedOffset () {
      return Number.isFinite(this.offset) ? this.offset : 7
    },
    showInlineMeta () {
      return Boolean((this.badge || this.status || this.shortcut) && !this.showRichFooter)
    },
    showRichFooter () {
      return Boolean(this.description && (this.badge || this.status || this.shortcut))
    }
  },
  mounted () {
    this.windowBlurHandler = () => this.dismiss('window-blur')
    this.peerOpenHandler = event => {
      if (event.detail?.id !== this.localId) {
        this.dismiss('peer-open')
      }
    }
    this.focusTrapHandler = event => {
      if (!this.open) return
      if (event.target?.closest('.swal2-container, .modal, .offcanvas, .command-palette, .sc-popover-panel')) {
        this.dismiss('overlay-focus')
      }
    }
    window.addEventListener('blur', this.windowBlurHandler)
    window.addEventListener(TOOLTIP_OPEN_EVENT, this.peerOpenHandler)
    window.addEventListener('sentinel:popover-open', this.peerOpenHandler)
    window.addEventListener('sentinel:command-palette-open', this.peerOpenHandler)
    document.addEventListener('focusin', this.focusTrapHandler, true)
  },
  beforeUnmount () {
    this.clearTouchTimers()
    window.removeEventListener('blur', this.windowBlurHandler)
    window.removeEventListener(TOOLTIP_OPEN_EVENT, this.peerOpenHandler)
    window.removeEventListener('sentinel:popover-open', this.peerOpenHandler)
    window.removeEventListener('sentinel:command-palette-open', this.peerOpenHandler)
    document.removeEventListener('focusin', this.focusTrapHandler, true)
  },
  methods: {
    emitOpenEvent () {
      window.dispatchEvent(new CustomEvent(TOOLTIP_OPEN_EVENT, { detail: { id: this.localId } }))
    },
    clearTouchTimers () {
      if (this.longPressTimer) {
        window.clearTimeout(this.longPressTimer)
        this.longPressTimer = null
      }
      if (this.longPressHideTimer) {
        window.clearTimeout(this.longPressHideTimer)
        this.longPressHideTimer = null
      }
    },
    handleOpenChange (value) {
      if (this.tooltipDisabled) {
        this.open = false
        return
      }
      this.open = value
      if (value) {
        this.emitOpenEvent()
      }
    },
    dismiss () {
      this.clearTouchTimers()
      this.longPressActive = false
      this.open = false
    },
    handlePointerDown (event) {
      this.dismiss('pointerdown')
      if (!this.hasRenderableContent) return
      if (!coarsePointer() && event.pointerType !== 'touch' && event.pointerType !== 'pen') return
      this.longPressTimer = window.setTimeout(() => {
        this.longPressActive = true
        this.open = true
        this.emitOpenEvent()
      }, 500)
    },
    handlePointerUp (event) {
      if (!coarsePointer() && event.pointerType !== 'touch' && event.pointerType !== 'pen') {
        return
      }
      if (this.longPressTimer) {
        window.clearTimeout(this.longPressTimer)
        this.longPressTimer = null
      }
      if (!this.longPressActive) return
      this.longPressHideTimer = window.setTimeout(() => {
        this.dismiss('long-press-timeout')
      }, 1500)
    },
    handlePointerLeave (event) {
      if (event.pointerType === 'mouse' || event.pointerType === '') {
        this.clearTouchTimers()
        return
      }
      if (!this.longPressActive) {
        this.clearTouchTimers()
      }
    }
  }
}
</script>

<style>
@keyframes sc-tooltip-in {
  from {
    opacity: 0;
    transform: scale(0.96);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes sc-tooltip-out {
  from {
    opacity: 1;
    transform: scale(1);
  }
  to {
    opacity: 0;
    transform: scale(0.97);
  }
}

#sc-tooltip-root {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: var(--z-tooltip);
}

.sc-tooltip {
  pointer-events: none;
  position: relative;
  display: grid;
  gap: 2px;
  max-width: min(280px, calc(var(--reka-tooltip-content-available-width, 280px) - 16px));
  padding: 5px 8px;
  border: 1px solid var(--tooltip-border);
  border-radius: var(--tooltip-radius-compact);
  background: var(--tooltip-surface);
  color: var(--tooltip-text);
  box-shadow: var(--tooltip-shadow);
  backdrop-filter: var(--tooltip-blur);
  -webkit-backdrop-filter: var(--tooltip-blur);
  font-family: var(--tooltip-font-family);
  transform-origin: var(--reka-tooltip-content-transform-origin);
  will-change: transform, opacity;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.sc-tooltip::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  box-shadow: inset 0 1px 0 var(--tooltip-inner-highlight);
  pointer-events: none;
}

.sc-tooltip[data-state='delayed-open'],
.sc-tooltip[data-state='instant-open'] {
  animation: sc-tooltip-in 120ms ease-out;
}

.sc-tooltip[data-state='closed'] {
  animation: sc-tooltip-out 80ms ease-in;
}

.sc-tooltip--rich,
.sc-tooltip--inline:has(.sc-tooltip__description),
.sc-tooltip:has(.sc-tooltip__description) {
  border-radius: var(--tooltip-radius-rich);
  padding: 8px 10px;
}

.sc-tooltip--inline {
  padding: 5px 6px 5px 9px;
}

.sc-tooltip__stack,
.sc-tooltip__footer,
.sc-tooltip__footer-badges,
.sc-tooltip__inline-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.sc-tooltip__stack {
  flex-direction: column;
  align-items: stretch;
  gap: 2px;
}

.sc-tooltip__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.sc-tooltip__header--stacked {
  align-items: flex-start;
}

.sc-tooltip__label {
  font-size: 12px;
  line-height: 1.35;
  font-weight: 500;
  letter-spacing: -0.005em;
  color: var(--tooltip-text);
}

.sc-tooltip__description {
  margin-top: 2px;
  font-size: 11px;
  line-height: 1.4;
  font-weight: 400;
  color: var(--tooltip-description);
  white-space: pre-line;
}

.sc-tooltip__footer {
  justify-content: space-between;
  margin-top: 4px;
}

.sc-tooltip__inline-meta {
  margin-left: auto;
}

.sc-tooltip__divider {
  height: 1px;
  width: 100%;
  margin: 4px 0;
  background: color-mix(in srgb, var(--tooltip-border) 88%, transparent);
}

.sc-tooltip__badge {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--tooltip-keycap-bg) 88%, transparent);
  border: 1px solid color-mix(in srgb, var(--tooltip-keycap-border) 90%, transparent);
  font-size: 10px;
  line-height: 1;
  font-weight: 600;
  letter-spacing: 0.01em;
  color: var(--tooltip-text);
}

.sc-tooltip__shortcut {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: auto;
}

.sc-tooltip__keycap {
  min-width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 5px;
  border-radius: 4px;
  background: var(--tooltip-keycap-bg);
  border: 1px solid var(--tooltip-keycap-border);
  color: var(--tooltip-keycap-text);
  font-family: var(--tooltip-font-family);
  font-size: 11px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.sc-tooltip__arrow {
  fill: var(--tooltip-surface);
}

@media (prefers-reduced-motion: reduce) {
  .sc-tooltip[data-state='delayed-open'],
  .sc-tooltip[data-state='instant-open'],
  .sc-tooltip[data-state='closed'] {
    animation: none;
    transition: opacity 250ms ease;
  }
}

@media (forced-colors: active) {
  .sc-tooltip {
    forced-color-adjust: auto;
    border-color: CanvasText;
    background: Canvas;
    color: CanvasText;
    box-shadow: none;
  }

  .sc-tooltip__keycap,
  .sc-tooltip__badge {
    border-color: CanvasText;
    background: Canvas;
    color: CanvasText;
  }
}
</style>