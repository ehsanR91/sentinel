<template>
  <section class="danger-zone" :aria-labelledby="headingId">
    <header class="danger-zone__header">
      <div>
        <div class="danger-zone__eyebrow">Danger Zone</div>
        <h3 :id="headingId" class="danger-zone__title">{{ title }}</h3>
        <p v-if="description" class="danger-zone__description">{{ description }}</p>
      </div>
    </header>
    <div class="danger-zone__list">
      <article v-for="item in items" :key="item.id" class="danger-zone__item">
        <div>
          <h4>{{ item.label }}</h4>
          <p>{{ item.description }}</p>
        </div>
        <div class="danger-zone__item-actions">
          <StatusBadge v-if="item.badge" :label="item.badge.label" :state="item.badge.state" />
          <AppButton :variant="item.variant || 'danger'" size="sm" :label="item.actionLabel" @click="$emit('action', item)" />
        </div>
      </article>
    </div>
  </section>
</template>

<script>
import AppButton from '@/components/ui/app-button.vue'
import StatusBadge from '@/components/ui/status-badge.vue'

export default {
  name: 'DangerZone',
  components: { AppButton, StatusBadge },
  props: {
    title: { type: String, default: 'High-impact actions' },
    description: { type: String, default: '' },
    headingId: { type: String, required: true },
    items: { type: Array, default: () => [] }
  },
  emits: ['action']
}
</script>

<style scoped>
.danger-zone {
  border: 1px solid color-mix(in srgb, var(--danger) 38%, var(--border-subtle));
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--danger) 6%, var(--surface-1));
}

.danger-zone__header {
  padding: var(--space-20) var(--space-20) var(--space-12);
}

.danger-zone__eyebrow {
  font-size: var(--font-size-11);
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: color-mix(in srgb, var(--danger) 76%, var(--text-primary));
  margin-bottom: var(--space-6);
}

.danger-zone__title {
  margin: 0;
  color: var(--text-primary);
  font-size: var(--font-size-18);
}

.danger-zone__description {
  margin: var(--space-8) 0 0;
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.danger-zone__list {
  border-top: 1px solid color-mix(in srgb, var(--danger) 22%, var(--border-subtle));
}

.danger-zone__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-12);
  padding: var(--space-16) var(--space-20);
}

.danger-zone__item + .danger-zone__item {
  border-top: 1px solid color-mix(in srgb, var(--danger) 16%, var(--border-subtle));
}

.danger-zone__item h4 {
  margin: 0;
  font-size: var(--font-size-15);
  color: var(--text-primary);
}

.danger-zone__item p {
  margin: var(--space-6) 0 0;
  font-size: var(--font-size-13);
  color: var(--text-secondary);
}

.danger-zone__item-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-8);
}

@media (max-width: 768px) {
  .danger-zone__item {
    flex-direction: column;
    align-items: stretch;
  }

  .danger-zone__item-actions {
    justify-content: space-between;
  }
}
</style>