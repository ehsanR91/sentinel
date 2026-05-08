<template>
  <div class="kv-list">
    <div v-if="!normalizedItems.length" class="kv-list__empty">{{ emptyText }}</div>
    <div v-for="(item, index) in normalizedItems" :key="index" class="kv-list__row">
      <input
        :value="item.value"
        class="kv-list__input sc-focus-ring"
        placeholder="CIDR or IP"
        @input="updateItem(index, 'value', $event.target.value)"
        @paste="onPaste(index, $event)"
      >
      <input
        :value="item.label"
        class="kv-list__input sc-focus-ring"
        placeholder="Optional label"
        @input="updateItem(index, 'label', $event.target.value)"
      >
      <button type="button" class="kv-list__delete sc-focus-ring" @click="removeItem(index)">
        <i class="mdi mdi-trash-can-outline"></i>
      </button>
    </div>
    <button type="button" class="kv-list__add sc-focus-ring" @click="addItem">
      <i class="mdi mdi-plus"></i>
      <span>Add entry</span>
    </button>
  </div>
</template>

<script>
export default {
  name: 'KeyValueList',
  props: {
    modelValue: { type: Array, default: () => [] },
    emptyText: { type: String, default: 'No entries yet.' }
  },
  emits: ['update:modelValue'],
  computed: {
    normalizedItems() {
      return this.modelValue.map(item => ({ value: item.value || '', label: item.label || '' }))
    }
  },
  methods: {
    emitItems(items) {
      this.$emit('update:modelValue', items)
    },
    updateItem(index, key, value) {
      const next = this.normalizedItems.slice()
      next[index] = { ...next[index], [key]: value }
      this.emitItems(next)
    },
    addItem() {
      this.emitItems([...this.normalizedItems, { value: '', label: '' }])
    },
    removeItem(index) {
      const next = this.normalizedItems.slice()
      next.splice(index, 1)
      this.emitItems(next)
    },
    onPaste(index, event) {
      const pasted = event.clipboardData?.getData('text') || ''
      const entries = pasted.split(/\r?\n/).map(line => line.trim()).filter(Boolean)
      if (entries.length <= 1) return
      event.preventDefault()
      const next = this.normalizedItems.slice()
      next.splice(index, 1, ...entries.map(value => ({ value, label: '' })))
      this.emitItems(next)
    }
  }
}
</script>

<style scoped>
.kv-list {
  display: grid;
  gap: var(--space-10);
}

.kv-list__empty {
  padding: var(--space-16);
  border: 1px dashed var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.kv-list__row {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1.5fr) auto;
  gap: var(--space-8);
}

.kv-list__input,
.kv-list__add,
.kv-list__delete {
  min-height: 44px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: var(--surface-1);
  color: var(--text-primary);
}

.kv-list__input {
  padding: 0.75rem 0.9rem;
}

.kv-list__add,
.kv-list__delete {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-6);
}

.kv-list__add {
  justify-self: start;
  padding: 0 0.85rem;
}

.kv-list__delete {
  width: 44px;
}

@media (max-width: 768px) {
  .kv-list__row {
    grid-template-columns: 1fr;
  }
}
</style>