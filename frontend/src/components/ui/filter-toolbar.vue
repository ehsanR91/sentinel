<template>
  <div class="sc-toolbar sc-surface">
    <!-- Collapsible header row -->
    <button
      type="button"
      class="sc-toolbar__toggle sc-focus-ring"
      :aria-expanded="expanded ? 'true' : 'false'"
      @click="expanded = !expanded"
    >
      <span class="sc-toolbar__toggle-label">
        <i class="mdi mdi-filter-outline"></i>
        Filters / Settings
        <span v-if="activeChips.length" class="sc-toolbar__toggle-badge">{{ activeChips.length }}</span>
      </span>
      <i class="mdi" :class="expanded ? 'mdi-chevron-up' : 'mdi-chevron-down'"></i>
    </button>

    <!-- Collapsible body -->
    <Transition name="toolbar-collapse">
      <div v-show="expanded" class="card-body">
        <div class="sc-toolbar__bar">
          <div class="sc-toolbar__left">
            <div class="sc-toolbar__search">
              <i class="mdi mdi-magnify"></i>
              <input
                :value="searchQuery"
                class="form-control sc-focus-ring"
                :placeholder="searchPlaceholder"
                type="search"
                @input="$emit('update:searchQuery', $event.target.value)"
              />
            </div>
            <slot name="controls" />
          </div>
          <div class="sc-toolbar__right">
            <slot name="meta">
              <span class="sc-inline-note">{{ resultLabel }}</span>
            </slot>
          </div>
        </div>
        <div v-if="activeChips.length" class="sc-filter-chips mt-3">
          <button
            v-for="chip in activeChips"
            :key="chip.key"
            type="button"
            class="sc-chip sc-chip--interactive sc-focus-ring"
            @click="$emit('remove-chip', chip.key)"
          >
            <span>{{ chip.label }}</span>
            <i class="mdi mdi-close"></i>
          </button>
          <button type="button" class="sc-chip sc-chip--interactive sc-focus-ring" @click="$emit('clear-all')">
            Clear all
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script>
export default {
  name: 'FilterToolbar',
  props: {
    searchQuery: { type: String, default: '' },
    searchPlaceholder: { type: String, default: 'Search…' },
    activeChips: { type: Array, default: () => [] },
    resultLabel: { type: String, default: '' },
    // Allow parent to start expanded (e.g. when filters are active)
    defaultExpanded: { type: Boolean, default: false }
  },
  emits: ['update:searchQuery', 'remove-chip', 'clear-all'],
  data () {
    return {
      expanded: this.defaultExpanded || this.activeChips.length > 0
    }
  },
  watch: {
    // Auto-expand when a chip is added from outside (e.g. stat-card click)
    activeChips (chips) {
      if (chips.length > 0) this.expanded = true
    }
  }
}
</script>

<style scoped>
.sc-toolbar__toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.6rem 1.25rem;
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--sc-text-secondary, #8aa4c8);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.03em;
  transition: color 0.15s ease;
}
.sc-toolbar__toggle:hover {
  color: var(--sc-text, #c9d8f0);
}
.sc-toolbar__toggle-label {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}
.sc-toolbar__toggle-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(74, 158, 255, 0.18);
  color: #4a9eff;
  font-size: 0.65rem;
  font-weight: 700;
}

/* Collapse transition */
.toolbar-collapse-enter-active,
.toolbar-collapse-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toolbar-collapse-enter-from,
.toolbar-collapse-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
