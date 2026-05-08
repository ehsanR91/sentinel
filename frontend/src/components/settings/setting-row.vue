<template>
  <article
    class="setting-row"
    :class="{ 'setting-row--highlighted': highlighted, 'setting-row--disabled': disabled }"
    :id="rowId"
  >
    <header class="setting-row__header">
      <div class="setting-row__title-wrap">
        <label v-if="labelFor" class="setting-row__label" :for="labelFor">{{ label }}</label>
        <div v-else class="setting-row__label">{{ label }}</div>
        <div class="setting-row__meta-inline">
          <slot name="status">
            <StatusBadge v-if="statusLabel" :label="statusLabel" :state="statusState" />
          </slot>
          <button
            v-if="helpText || helpHref"
            type="button"
            class="setting-row__help sc-focus-ring"
            :aria-label="`Learn more about ${label}`"
            @click="$emit('help')"
          >
            <i class="mdi mdi-help-circle-outline" aria-hidden="true"></i>
          </button>
          <div v-if="$slots.menu" class="setting-row__menu">
            <slot name="menu" />
          </div>
        </div>
      </div>
      <p v-if="description" class="setting-row__description">{{ description }}</p>
      <a v-if="helpHref" class="setting-row__learn" :href="helpHref" target="_blank" rel="noreferrer">learn more -></a>
      <div v-else-if="helpText" class="setting-row__help-text">{{ helpText }}</div>
    </header>

    <div class="setting-row__control">
      <slot />
    </div>

    <footer v-if="footnote || error" class="setting-row__footer">
      <div v-if="error" class="setting-row__error" role="alert">
        <i class="mdi mdi-alert-circle-outline" aria-hidden="true"></i>
        <span>{{ error }}</span>
      </div>
      <div v-if="footnote" class="setting-row__footnote">{{ footnote }}</div>
    </footer>
  </article>
</template>

<script>
import StatusBadge from '@/components/ui/status-badge.vue'

export default {
  name: 'SettingRow',
  components: { StatusBadge },
  props: {
    rowId: { type: String, required: true },
    label: { type: String, required: true },
    description: { type: String, default: '' },
    helpText: { type: String, default: '' },
    helpHref: { type: String, default: '' },
    footnote: { type: String, default: '' },
    statusLabel: { type: String, default: '' },
    statusState: { type: String, default: 'muted' },
    error: { type: String, default: '' },
    labelFor: { type: String, default: '' },
    highlighted: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false }
  },
  emits: ['help']
}
</script>

<style scoped>
.setting-row {
  padding: var(--space-16);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--surface-2);
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.setting-row--highlighted {
  border-color: color-mix(in srgb, var(--warning) 58%, var(--border-subtle));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--warning) 22%, transparent);
}

.setting-row--disabled {
  opacity: 0.74;
}

.setting-row__header {
  display: grid;
  gap: var(--space-8);
}

.setting-row__title-wrap {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-12);
}

.setting-row__label {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.setting-row__meta-inline {
  display: inline-flex;
  align-items: center;
  gap: var(--space-8);
}

.setting-row__description,
.setting-row__help-text,
.setting-row__footnote,
.setting-row__learn {
  margin: 0;
  font-size: var(--font-size-13);
  color: var(--text-secondary);
}

.setting-row__learn {
  text-decoration: none;
  color: var(--accent);
}

.setting-row__help {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--text-tertiary);
}

.setting-row__control {
  margin-top: var(--space-16);
}

.setting-row__footer {
  display: grid;
  gap: var(--space-8);
  margin-top: var(--space-12);
}

.setting-row__error {
  display: inline-flex;
  align-items: center;
  gap: var(--space-6);
  font-size: var(--font-size-13);
  color: var(--danger);
}

@media (max-width: 768px) {
  .setting-row__title-wrap {
    flex-direction: column;
  }
}
</style>