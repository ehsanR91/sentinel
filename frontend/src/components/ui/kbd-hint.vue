<template>
  <span class="sc-kbd-hint" aria-hidden="true">{{ hint }}</span>
</template>

<script>
export default {
  name: 'KbdHint',
  props: {
    hint: { type: String, required: true }
  }
}
</script>

<style scoped>
.sc-kbd-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 22px;
  padding: 0 0.45rem;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--border-subtle) 84%, transparent);
  background: color-mix(in srgb, var(--surface-2) 86%, transparent);
  color: var(--text-tertiary);
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1;
}
</style>