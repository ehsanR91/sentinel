<template>
  <section class="settings-section sc-surface" :class="{ 'settings-section--sticky': sticky }" :aria-labelledby="headingId">
    <header class="settings-section__header">
      <div>
        <div v-if="eyebrow" class="settings-section__eyebrow">{{ eyebrow }}</div>
        <h2 :id="headingId" class="settings-section__title">{{ title }}</h2>
        <p v-if="description" class="settings-section__description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="settings-section__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="settings-section__body">
      <slot />
    </div>
  </section>
</template>

<script>
export default {
  name: 'SettingsSection',
  props: {
    title: { type: String, required: true },
    description: { type: String, default: '' },
    eyebrow: { type: String, default: '' },
    headingId: { type: String, required: true },
    sticky: { type: Boolean, default: false }
  }
}
</script>

<style scoped>
.settings-section {
  padding: var(--space-20);
  border-radius: var(--radius-lg);
}

.settings-section--sticky {
  scroll-margin-top: calc(84px + var(--space-20));
}

.settings-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-12);
  margin-bottom: var(--space-20);
}

.settings-section__eyebrow {
  font-size: var(--font-size-11);
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  margin-bottom: var(--space-6);
}

.settings-section__title {
  margin: 0;
  font-size: var(--font-size-20);
  font-weight: 600;
  color: var(--text-primary);
}

.settings-section__description {
  margin: var(--space-8) 0 0;
  max-width: 68ch;
  font-size: var(--font-size-13);
  color: var(--text-secondary);
}

.settings-section__actions {
  display: flex;
  align-items: center;
  gap: var(--space-8);
}

.settings-section__body {
  display: grid;
  gap: var(--space-16);
}

@media (max-width: 768px) {
  .settings-section {
    padding: var(--space-16);
  }

  .settings-section__header {
    flex-direction: column;
  }
}
</style>