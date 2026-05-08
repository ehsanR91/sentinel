<template>
  <transition name="save-bar-slide">
    <div v-if="visible" class="save-bar sc-surface" aria-live="polite">
      <div class="save-bar__copy">
        <strong>{{ stateLabel }}</strong>
        <button v-if="changes.length" type="button" class="save-bar__changes" @click="$emit('review')">
          {{ changes.length }} unsaved {{ changes.length === 1 ? 'change' : 'changes' }}
        </button>
      </div>
      <div class="save-bar__actions">
        <AppButton variant="ghost" size="sm" label="Discard" :disabled="saving" @click="$emit('discard')" />
        <AppButton variant="primary" size="sm" :label="saveLabel" :loading="saving" :disabled="disabled" @click="$emit('save')" />
      </div>
    </div>
  </transition>
</template>

<script>
import AppButton from '@/components/ui/app-button.vue'

export default {
  name: 'SaveBar',
  components: { AppButton },
  props: {
    visible: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
    saving: { type: Boolean, default: false },
    changes: { type: Array, default: () => [] },
    stateLabel: { type: String, default: 'Unsaved changes' },
    saveLabel: { type: String, default: 'Save' }
  },
  emits: ['save', 'discard', 'review']
}
</script>

<style scoped>
.save-bar {
  position: sticky;
  bottom: var(--space-16);
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-12);
  padding: var(--space-16) var(--space-20);
  border-radius: var(--radius-lg);
  border: 1px solid color-mix(in srgb, var(--accent) 28%, var(--border-subtle));
  box-shadow: 0 18px 40px color-mix(in srgb, #000 24%, transparent);
}

.save-bar__copy {
  display: grid;
  gap: var(--space-4);
}

.save-bar__copy strong {
  color: var(--text-primary);
  font-size: var(--font-size-15);
}

.save-bar__changes {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--accent);
  font-size: var(--font-size-13);
  text-align: left;
}

.save-bar__actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-8);
}

.save-bar-slide-enter-active,
.save-bar-slide-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.save-bar-slide-enter-from,
.save-bar-slide-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

@media (max-width: 768px) {
  .save-bar {
    left: 0;
    right: 0;
    bottom: 0;
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
    padding-bottom: calc(var(--space-16) + env(safe-area-inset-bottom));
    flex-direction: column;
    align-items: stretch;
  }

  .save-bar__actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>