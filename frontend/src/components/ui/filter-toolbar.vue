<template>
  <div class="sc-toolbar sc-surface">
    <div class="card-body">
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
  </div>
</template>

<script>
export default {
  name: 'FilterToolbar',
  props: {
    searchQuery: { type: String, default: '' },
    searchPlaceholder: { type: String, default: 'Search…' },
    activeChips: { type: Array, default: () => [] },
    resultLabel: { type: String, default: '' }
  },
  emits: ['update:searchQuery', 'remove-chip', 'clear-all']
}
</script>