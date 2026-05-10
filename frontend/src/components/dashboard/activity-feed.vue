<template>
  <section class="activity-feed sc-focus-ring">
    <div class="activity-feed__header">
      <div>
        <div class="activity-feed__eyebrow">Activity Feed</div>
        <h3 class="activity-feed__title">Live Operations Timeline</h3>
      </div>
      <div class="activity-feed__tabs" role="tablist" aria-label="Activity feed views">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="activity-feed__tab"
          :class="{ 'is-active': tab.key === activeTab }"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
          <span>{{ tab.count }}</span>
        </button>
        <button type="button" class="activity-feed__tab" style="padding: 0 8px;" @click="isCollapsed = !isCollapsed"><i class="mdi" :class="isCollapsed ? 'mdi-chevron-down' : 'mdi-chevron-up'"></i></button>
      </div>
    </div>

    <div class="activity-feed__body" v-show="!isCollapsed">
      <div v-if="!activeItems.length" class="activity-feed__empty">
        <i class="mdi mdi-timeline-clock-outline"></i>
        <span>No recent activity. New alerts, logins, audit events, and maintenance runs will appear here.</span>
      </div>
      <button
        v-for="item in activeItems"
        :key="item.id"
        type="button"
        class="activity-feed__row"
        :class="`activity-feed__row--${item.severity || 'info'}`"
        @click="$emit('open-item', item)"
      >
        <span class="activity-feed__icon" :class="`activity-feed__icon--${item.severity || 'info'}`">
          <i :class="item.icon || 'mdi mdi-bell-outline'"></i>
        </span>
        <span class="activity-feed__content">
          <span class="activity-feed__summary">{{ item.summary }}</span>
          <span class="activity-feed__meta">{{ item.meta }}</span>
        </span>
        <span class="activity-feed__time">{{ formatRelative(item.ts) }}</span>
      </button>
    </div>
  </section>
</template>

<script>
export default {
  name: 'DashboardActivityFeed',
  props: {
    itemsByTab: { type: Object, default: () => ({}) }
  },
  emits: ['open-item'],
  data() {
    return {
      isCollapsed: false,
      activeTab: 'all'
    }
  },
  computed: {
    tabs() {
      const entries = Object.entries(this.itemsByTab)
      return entries.map(([key, items]) => ({
        key,
        label: key === 'all' ? 'All Events' : key.charAt(0).toUpperCase() + key.slice(1),
        count: Array.isArray(items) ? items.length : 0
      }))
    },
    activeItems() {
      const items = this.itemsByTab[this.activeTab] || []
      return Array.isArray(items) ? items.slice(0, 18) : []
    }
  },
  methods: {
    formatRelative(ts) {
      if (!ts) return 'Unknown'
      const seconds = Math.max(0, Math.floor((Date.now() - ts) / 1000))
      if (seconds < 60) return `${seconds}s ago`
      if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
      if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
      return `${Math.floor(seconds / 86400)}d ago`
    }
  }
}
</script>

<style scoped>
.activity-feed {
  border-radius: 22px;
  border: 1px solid var(--dashboard-panel-border);
  background: var(--dashboard-panel-bg);
  box-shadow: var(--shadow-md);
}

.activity-feed__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 18px;
  border-bottom: 1px solid var(--dashboard-panel-border);
}

.activity-feed__eyebrow {
  color: var(--text-tertiary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 700;
}

.activity-feed__title {
  margin: 4px 0 0;
  font-size: 18px;
  color: var(--text-primary);
}

.activity-feed__tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.activity-feed__tab {
  border-radius: 999px;
  border: 1px solid var(--dashboard-panel-border);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
  padding: 8px 12px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.activity-feed__tab span {
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  display: inline-grid;
  place-items: center;
  font-size: 10px;
}

.activity-feed__tab.is-active {
  background: var(--accent-muted);
  color: var(--text-primary);
}

.activity-feed__body {
  display: flex;
  flex-direction: column;
}

.activity-feed__row {
  width: 100%;
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 18px;
  border: 0;
  border-bottom: 1px solid rgba(138, 164, 200, 0.08);
  background: transparent;
  text-align: left;
  font-variant-numeric: tabular-nums;
}

.activity-feed__row:hover {
  background: var(--dashboard-activity-highlight);
}

.activity-feed__icon {
  width: 34px;
  height: 34px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  font-size: 18px;
}

.activity-feed__icon--critical,
.activity-feed__icon--error {
  background: var(--state-error-bg);
  color: var(--state-error-fg);
}

.activity-feed__icon--warning,
.activity-feed__icon--warn {
  background: var(--state-warn-bg);
  color: var(--state-warn-fg);
}

.activity-feed__icon--info,
.activity-feed__icon--healthy {
  background: var(--state-info-bg);
  color: var(--state-info-fg);
}

.activity-feed__content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.activity-feed__summary {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.activity-feed__meta,
.activity-feed__time,
.activity-feed__empty {
  color: var(--text-secondary);
  font-size: 12px;
}

.activity-feed__time {
  white-space: nowrap;
}

.activity-feed__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 40px 18px;
}

@media (max-width: 960px) {
  .activity-feed__header {
    flex-direction: column;
  }

  .activity-feed__row {
    grid-template-columns: auto 1fr;
  }

  .activity-feed__time {
    grid-column: 2;
  }
}
</style>