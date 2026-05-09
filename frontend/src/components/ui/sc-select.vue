<template>
  <div
    ref="root"
    class="sc-select"
    :class="[`sc-select--${size}`, { 'is-open': open, 'is-disabled': disabled }]"
    @keydown="onKeydown"
  >
    <button
      type="button"
      class="sc-select__trigger sc-focus-ring"
      :disabled="disabled"
      :aria-haspopup="'listbox'"
      :aria-expanded="open"
      :aria-labelledby="inputId || undefined"
      @click.stop="toggle"
    >
      <span class="sc-select__value">{{ selectedLabel }}</span>
      <i class="mdi mdi-chevron-down sc-select__chevron" aria-hidden="true"></i>
    </button>

    <Teleport to="body">
      <ul
        v-if="open"
        ref="menu"
        class="sc-select__menu"
        role="listbox"
        :style="menuStyle"
        @mousedown.prevent
      >
        <li
          v-for="(option, index) in normalizedOptions"
          :key="option.value"
          role="option"
          class="sc-select__option"
          :class="{
            'is-selected': option.value === modelValue,
            'is-focused': index === focusedIndex
          }"
          :aria-selected="option.value === modelValue"
          @click="select(option)"
          @mouseenter="focusedIndex = index"
        >
          <i v-if="option.icon" :class="option.icon" class="sc-select__option-icon" aria-hidden="true"></i>
          <span>{{ option.label }}</span>
          <i v-if="option.value === modelValue" class="mdi mdi-check sc-select__check" aria-hidden="true"></i>
        </li>
      </ul>
    </Teleport>
  </div>
</template>

<script>
export default {
  name: 'ScSelect',
  props: {
    modelValue: { type: [String, Number, Boolean], default: '' },
    options: { type: Array, default: () => [] },
    placeholder: { type: String, default: 'Select…' },
    disabled: { type: Boolean, default: false },
    size: { type: String, default: 'md' },
    inputId: { type: String, default: '' }
  },
  emits: ['update:modelValue', 'change'],
  data() {
    return {
      open: false,
      focusedIndex: -1,
      menuStyle: {}
    }
  },
  computed: {
    normalizedOptions() {
      return this.options.map(opt =>
        typeof opt === 'object' ? opt : { value: opt, label: String(opt) }
      )
    },
    selectedLabel() {
      const found = this.normalizedOptions.find(o => o.value === this.modelValue)
      return found ? found.label : this.placeholder
    }
  },
  mounted() {
    this._onOutside = (e) => {
      if (!this.$refs.root?.contains(e.target)) this.close()
    }
    window.addEventListener('pointerdown', this._onOutside)
    window.addEventListener('scroll', this._onScroll, { passive: true, capture: true })
    window.addEventListener('resize', this._onScroll, { passive: true })
  },
  beforeUnmount() {
    window.removeEventListener('pointerdown', this._onOutside)
    window.removeEventListener('scroll', this._onScroll, { capture: true })
    window.removeEventListener('resize', this._onScroll)
  },
  methods: {
    toggle() {
      if (this.disabled) return
      this.open ? this.close() : this.openMenu()
    },
    openMenu() {
      this.open = true
      this.focusedIndex = this.normalizedOptions.findIndex(o => o.value === this.modelValue)
      this.$nextTick(() => this._positionMenu())
    },
    close() {
      this.open = false
    },
    select(option) {
      this.$emit('update:modelValue', option.value)
      this.$emit('change', option.value)
      this.close()
    },
    _onScroll() {
      if (this.open) this._positionMenu()
    },
    _positionMenu() {
      const trigger = this.$refs.root
      const menu = this.$refs.menu
      if (!trigger || !menu) return
      const rect = trigger.getBoundingClientRect()
      const menuH = menu.offsetHeight || 280
      const vp = window.innerHeight
      const spaceBelow = vp - rect.bottom - 6
      const spaceAbove = rect.top - 6
      const openUp = spaceBelow < menuH && spaceAbove > spaceBelow
      this.menuStyle = {
        position: 'fixed',
        zIndex: 9800,
        left: `${rect.left}px`,
        width: `${Math.max(rect.width, 160)}px`,
        ...(openUp
          ? { bottom: `${vp - rect.top + 4}px`, top: 'auto' }
          : { top: `${rect.bottom + 4}px`, bottom: 'auto' })
      }
    },
    onKeydown(e) {
      if (!this.open) {
        if (['Enter', ' ', 'ArrowDown', 'ArrowUp'].includes(e.key)) {
          e.preventDefault()
          this.openMenu()
        }
        return
      }
      if (e.key === 'Escape') { this.close(); return }
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        this.focusedIndex = Math.min(this.focusedIndex + 1, this.normalizedOptions.length - 1)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        this.focusedIndex = Math.max(this.focusedIndex - 1, 0)
      } else if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        if (this.focusedIndex >= 0) this.select(this.normalizedOptions[this.focusedIndex])
      }
    }
  }
}
</script>

<style scoped>
.sc-select {
  position: relative;
  display: inline-flex;
  min-width: 0;
  width: 100%;
}

.sc-select__trigger {
  width: 100%;
  min-height: 34px;
  padding: 0 2.25rem 0 0.75rem;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--surface-2, #141c2f);
  border: 1px solid var(--border-default, #1e2d4a);
  border-radius: var(--radius-md, 6px);
  color: var(--text-primary, #c9d8f0);
  font-size: var(--font-size-13, 0.8125rem);
  font-weight: 400;
  line-height: 1.4;
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  position: relative;
}

.sc-select__trigger:hover:not(:disabled) {
  border-color: var(--border-strong, #243656);
  background: var(--surface-3, #1a2540);
}

.is-open .sc-select__trigger {
  border-color: var(--accent, #4a9eff);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent, #4a9eff) 18%, transparent);
  background: var(--surface-2, #141c2f);
}

.sc-select__trigger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.sc-select__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sc-select__chevron {
  position: absolute;
  right: 0.6rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary, #5a7499);
  font-size: 16px;
  transition: transform 0.18s ease;
  pointer-events: none;
  flex-shrink: 0;
}

.is-open .sc-select__chevron {
  transform: translateY(-50%) rotate(180deg);
}

/* Sizes */
.sc-select--sm .sc-select__trigger {
  min-height: 28px;
  font-size: var(--font-size-12, 0.75rem);
  padding: 0 2rem 0 0.6rem;
}

.sc-select--lg .sc-select__trigger {
  min-height: 40px;
  padding: 0 2.5rem 0 0.9rem;
}
</style>

<style>
/* Global — menu is teleported to body */
.sc-select__menu {
  list-style: none;
  margin: 0;
  padding: 5px;
  background: var(--sc-bg-card, #0f1629);
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 10px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.42), 0 2px 8px rgba(0, 0, 0, 0.22);
  overflow-y: auto;
  max-height: 260px;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  animation: sc-select-in 0.13s ease;
}

@keyframes sc-select-in {
  from { opacity: 0; transform: translateY(-4px) scale(0.98); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}

.sc-select__option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 6px;
  font-size: 0.8125rem;
  color: var(--sc-text-secondary, #8aa4c8);
  cursor: pointer;
  transition: background 0.1s ease, color 0.1s ease;
  white-space: nowrap;
  user-select: none;
}

.sc-select__option:hover,
.sc-select__option.is-focused {
  background: rgba(74, 158, 255, 0.1);
  color: var(--sc-text, #c9d8f0);
}

.sc-select__option.is-selected {
  color: var(--sc-blue, #4a9eff);
  font-weight: 500;
}

.sc-select__option-icon {
  font-size: 15px;
  color: var(--sc-text-muted, #5a7499);
  flex-shrink: 0;
}

.sc-select__check {
  margin-left: auto;
  font-size: 14px;
  color: var(--sc-blue, #4a9eff);
  flex-shrink: 0;
}

[data-theme="light"] .sc-select__menu {
  background: #ffffff;
  border-color: rgba(0, 0, 0, 0.1);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.14), 0 2px 8px rgba(0, 0, 0, 0.08);
}

[data-theme="light"] .sc-select__option {
  color: #4a5568;
}

[data-theme="light"] .sc-select__option:hover,
[data-theme="light"] .sc-select__option.is-focused {
  background: rgba(37, 99, 235, 0.08);
  color: #1a202c;
}

[data-theme="light"] .sc-select__option.is-selected {
  color: #2563eb;
}

[data-theme="light"] .sc-select__trigger {
  background: #f8fafc;
  border-color: #d1d9e6;
  color: #1a202c;
}

[data-theme="light"] .sc-select__trigger:hover:not(:disabled) {
  border-color: #a0aec0;
  background: #f1f5f9;
}
</style>
