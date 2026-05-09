<template>
  <div class="sc-page-shell sc-focus-ring alerts-page" tabindex="0" @keydown="handleKeyboardNavigation">
    <PageHeader title="Alerts" icon="mdi mdi-bell-alert" :items="[{ text: 'Alerts', active: true, icon: 'mdi mdi-bell-alert' }]">
      <template #actions>
        <AppButton variant="secondary" size="md" icon="mdi mdi-refresh" :loading="loading" label="Refresh" @click="loadAlerts(true)" />
        <AppButton
          :variant="notificationsEnabled ? 'primary' : 'secondary'"
          size="md"
          :icon="notificationsEnabled ? 'mdi mdi-bell-ring' : 'mdi mdi-bell-off-outline'"
          :label="notificationsEnabled ? 'Notifications On' : 'Notifications Off'"
          @click="toggleNotifications"
        />
        <AppButton variant="primary" size="md" icon="mdi mdi-check-all" label="Mark All Read" @click="markAllRead" />
      </template>
    </PageHeader>

    <div v-if="queuedAlerts.length" class="sc-inline-error updates-banner">
      <div class="d-flex flex-wrap gap-2 align-items-center">
        <strong>{{ queuedAlerts.length }} new alerts queued</strong>
        <span>Your scroll position stays stable until you apply them.</span>
      </div>
      <div class="d-flex flex-wrap gap-2">
        <AppButton variant="secondary" size="sm" icon="mdi mdi-arrow-collapse-up" label="Show updates" @click="applyQueuedAlerts" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-close" label="Dismiss" @click="queuedAlerts = []" />
      </div>
    </div>

    <div class="row g-3 stat-row">
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Emergency" :value="countBySeverity('emergency')" sub="Immediate action" icon="mdi mdi-alert-octagon-outline" :tone="countBySeverity('emergency') ? 'critical' : 'default'" clickable @click="severityFilter = severityFilter === 'emergency' ? '' : 'emergency'" />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Critical" :value="countBySeverity('critical')" sub="Investigate now" icon="mdi mdi-alert-circle-outline" :tone="countBySeverity('critical') ? 'error' : 'default'" clickable @click="severityFilter = severityFilter === 'critical' ? '' : 'critical'" />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Warning" :value="countBySeverity('warning')" sub="Needs review" icon="mdi mdi-alert-outline" :tone="countBySeverity('warning') ? 'warn' : 'default'" clickable @click="severityFilter = severityFilter === 'warning' ? '' : 'warning'" />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Info" :value="countBySeverity('info')" sub="Observed signals" icon="mdi mdi-information-outline" clickable @click="severityFilter = severityFilter === 'info' ? '' : 'info'" />
      </div>
    </div>

    <FilterToolbar
      :search-query="searchQuery"
      search-placeholder="Search alert text, source, IP… Use source:sshd severity:warning ip:1.2.3.4"
      :active-chips="activeChips"
      :result-label="resultLabel"
      @update:search-query="updateSearch"
      @remove-chip="removeChip"
      @clear-all="clearFilters"
    >
      <template #controls>
        <ScSelect v-model="readFilter" :options="[{value:'all',label:'All states'},{value:'unread',label:'Unread'},{value:'read',label:'Read'}]" size="sm" />
        <ScSelect v-model="sourceFilter" :options="[{value:'',label:'All sources'},...sourceOptions.map(s=>({value:s,label:s}))]" size="sm" />
        <ScSelect v-model="datePreset" :options="[{value:'15m',label:'Last 15m'},{value:'1h',label:'Last 1h'},{value:'6h',label:'Last 6h'},{value:'24h',label:'Last 24h'},{value:'7d',label:'Last 7d'},{value:'custom',label:'Custom'},{value:'all',label:'All time'}]" size="sm" @change="applyDatePreset(datePreset)" />
        <template v-if="datePreset === 'custom'">
          <input v-model="customStart" type="datetime-local" class="form-control sc-focus-ring toolbar-input" />
          <input v-model="customEnd" type="datetime-local" class="form-control sc-focus-ring toolbar-input" />
        </template>
        <details class="toolbar-menu">
          <summary class="sc-button sc-button--secondary sc-button--md">
            <i class="mdi mdi-content-save-outline"></i>
            Saved filters
          </summary>
          <div class="toolbar-menu__body">
            <button type="button" class="dropdown-item" @click="saveCurrentFilter">Save current view</button>
            <div v-if="!savedFilters.length" class="toolbar-menu__empty">No saved views yet</div>
            <template v-else>
              <div v-for="filter in savedFilters" :key="filter.id" class="saved-filter-row">
                <button type="button" class="dropdown-item" @click="applySavedFilter(filter)">{{ filter.name }}</button>
                <button type="button" class="sc-button sc-button--ghost sc-button--sm sc-button--icon-only" aria-label="Delete saved filter" @click="deleteSavedFilter(filter.id)">
                  <i class="mdi mdi-close"></i>
                </button>
              </div>
            </template>
          </div>
        </details>
      </template>
      <template #meta>
        <span class="sc-inline-note">{{ resultLabel }}</span>
        <span class="sc-inline-note">Last updated {{ lastUpdatedLabel }}</span>
        <ScSelect v-model="density" :options="[{value:'comfortable',label:'Comfortable'},{value:'compact',label:'Compact'}]" size="sm" style="width:130px" @change="persistDensity" />
        <AppButton :variant="multiSelect ? 'primary' : 'secondary'" size="md" icon="mdi mdi-checkbox-multiple-marked-outline" :label="multiSelect ? 'Selecting' : 'Select'" @click="toggleMultiSelect" />
      </template>
    </FilterToolbar>

    <ErrorState v-if="errorMessage" title="Alert refresh failed" :description="errorMessage">
      <template #actions>
        <AppButton variant="secondary" size="sm" icon="mdi mdi-refresh" label="Retry" @click="loadAlerts(true)" />
      </template>
    </ErrorState>

    <div v-if="multiSelect && selectedGroupIds.length" class="sc-inline-error bulk-bar">
      <div class="d-flex flex-wrap gap-2 align-items-center">
        <strong>{{ selectedGroupIds.length }} alert groups selected</strong>
        <span>Bulk dismiss, snooze, or export the current selection.</span>
      </div>
      <div class="d-flex flex-wrap gap-2">
        <AppButton variant="secondary" size="sm" icon="mdi mdi-check" label="Dismiss selected" @click="markSelectedRead" />
        <AppButton variant="secondary" size="sm" icon="mdi mdi-timer-outline" label="Snooze 1h" @click="snoozeSelected(60 * 60 * 1000)" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-download" label="Export" @click="exportAlerts(true)" />
      </div>
    </div>

    <div class="sc-surface alerts-shell">
      <div ref="listContainer" class="alerts-list-wrap">
        <div v-if="loading" class="skeleton-wrap" aria-busy="true">
          <div v-for="n in 8" :key="n" class="skeleton-row"></div>
        </div>

        <EmptyState
          v-else-if="!alerts.length"
          icon="mdi mdi-bell-off-outline"
          title="No alerts yet"
          description="SentinelCore will surface detections here as auth, service, and security signals arrive."
        >
          <template #actions>
            <AppButton variant="secondary" size="md" icon="mdi mdi-refresh" label="Refresh" @click="loadAlerts(true)" />
          </template>
        </EmptyState>

        <EmptyState
          v-else-if="!visibleGroups.length"
          icon="mdi mdi-filter-off-outline"
          title="No alerts match these filters"
          description="Clear the active filters or wait for new alert activity."
        >
          <template #actions>
            <AppButton variant="secondary" size="md" icon="mdi mdi-filter-remove-outline" label="Clear filters" @click="clearFilters" />
          </template>
        </EmptyState>

        <template v-else>
          <div class="alerts-list">
            <article
              v-for="(group, index) in visibleGroups"
              :key="group.id"
              class="alert-row"
              :class="{
                unread: !group.read,
                highlighted: highlightedGroupIds.includes(group.id),
                expanded: expandedGroupId === group.id,
                compact: density === 'compact',
                active: focusedIndex === index
              }"
              @click="openGroup(group)"
              @mouseenter="scheduleSeen(group)"
              @mouseleave="cancelSeen(group.id)"
            >
              <div v-if="multiSelect" class="alert-checkbox" @click.stop>
                <input type="checkbox" :checked="selectedGroupIds.includes(group.id)" @change="toggleGroupSelected(group.id)" />
              </div>
              <div class="alert-row__content">
                <div class="alert-row__top">
                  <div class="alert-top-main">
                    <StatusBadge :state="severityState(group.base.severity)" :label="severityLabel(group.base.severity)" :icon="severityIcon(group.base.severity)" />
                    <span class="alert-type">{{ group.base.type || 'Alert' }}</span>
                    <span class="alert-source">{{ group.base.source || 'system' }}</span>
                    <span class="alert-time">{{ formatRelative(group.latestTs) }}</span>
                    <StatusBadge v-if="group.count > 1" state="info" :label="`× ${group.count} in 5m`" />
                  </div>
                  <div class="alert-top-actions" @click.stop>
                    <button
                      type="button"
                      class="sc-button sc-button--ghost sc-button--sm sc-button--icon-only"
                      :aria-label="expandedGroupId === group.id ? 'Collapse grouped events' : 'Expand grouped events'"
                      @click="toggleExpanded(group.id)"
                    >
                      <i :class="expandedGroupId === group.id ? 'mdi mdi-chevron-up' : 'mdi mdi-chevron-down'"></i>
                    </button>
                    <details class="alert-menu">
                      <summary class="sc-button sc-button--ghost sc-button--sm sc-button--icon-only">
                        <i class="mdi mdi-dots-horizontal"></i>
                      </summary>
                      <div class="toolbar-menu__body alert-menu__body">
                        <button type="button" class="dropdown-item" @click="markGroupRead(group)">Acknowledge</button>
                        <button type="button" class="dropdown-item" @click="markGroupRead(group)">Dismiss</button>
                        <button type="button" class="dropdown-item" @click="snoozeGroup(group, 5 * 60 * 1000)">Snooze 5m</button>
                        <button type="button" class="dropdown-item" @click="snoozeGroup(group, 60 * 60 * 1000)">Snooze 1h</button>
                        <button type="button" class="dropdown-item" @click="snoozeGroup(group, 24 * 60 * 60 * 1000)">Snooze 24h</button>
                        <button type="button" class="dropdown-item" @click="muteRule(group)">Mute this rule</button>
                        <button type="button" class="dropdown-item" :disabled="!group.base.ip" @click="blockSourceIp(group)">Block source IP</button>
                        <button type="button" class="dropdown-item" @click="exportGroup(group)">Export group</button>
                      </div>
                    </details>
                  </div>
                </div>

                <div class="alert-row__summary">
                  <span>{{ summarize(group.base) }}</span>
                </div>

                <div class="alert-row__chips" @click.stop>
                  <IpChip v-for="ip in alertIps(group.base)" :key="`${group.id}-${ip}`" :ip="ip" :tooltip="`Filter by ${ip}`" @click="filterByIp(ip)" />
                  <span v-for="meta in alertTags(group.base)" :key="`${group.id}-${meta}`" class="sc-chip">{{ meta }}</span>
                </div>

                <div v-if="expandedGroupId === group.id && group.items.length > 1" class="alert-expansion">
                  <div v-for="item in group.items" :key="item.id" class="alert-expansion__item">
                    <div>
                      <strong>{{ formatRelative(item.ts) }}</strong>
                      <span class="text-muted"> · {{ item.source }}</span>
                    </div>
                    <div class="text-sm text-secondary">{{ item.message }}</div>
                  </div>
                </div>
              </div>
            </article>
          </div>
          <div ref="loadMoreSentinel" class="load-more-sentinel"></div>
        </template>
      </div>
    </div>

    <DetailDrawer
      :model-value="showDrawer"
      :title="selectedGroup ? `${selectedGroup.base.type || 'Alert'} · ${selectedGroup.base.source || 'system'}` : 'Alert details'"
      :subtitle="selectedGroup ? severityLabel(selectedGroup.base.severity) : ''"
      @update:model-value="showDrawer = $event"
      @navigate="navigateDrawer"
    >
      <template #nav>
        <AppButton variant="ghost" size="sm" icon="mdi mdi-chevron-left" aria-label="Previous alert group" icon-only @click="navigateDrawer(-1)" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-chevron-right" aria-label="Next alert group" icon-only @click="navigateDrawer(1)" />
      </template>

      <div v-if="selectedGroup" class="drawer-grid">
        <section class="drawer-panel">
          <h6>Alert Summary</h6>
          <div class="drawer-stack">
            <div class="drawer-meta-row"><span>Severity</span><StatusBadge :state="severityState(selectedGroup.base.severity)" :label="severityLabel(selectedGroup.base.severity)" :icon="severityIcon(selectedGroup.base.severity)" /></div>
            <div class="drawer-meta-row"><span>Source</span><span>{{ selectedGroup.base.source || 'system' }}</span></div>
            <div class="drawer-meta-row"><span>Occurred</span><TimeDisplay :value="selectedGroup.latestTs" /></div>
            <div class="drawer-meta-row"><span>Volume</span><span>{{ selectedGroup.count }} events in the group</span></div>
            <div class="drawer-meta-row"><span>Read</span><StatusBadge :state="selectedGroup.read ? 'muted' : 'info'" :label="selectedGroup.read ? 'Seen' : 'New'" /></div>
          </div>
        </section>

        <section class="drawer-panel">
          <h6>Human Summary</h6>
          <p class="drawer-copy">{{ summarize(selectedGroup.base) }}</p>
          <div class="drawer-chip-row">
            <IpChip v-for="ip in alertIps(selectedGroup.base)" :key="`drawer-${ip}`" :ip="ip" :tooltip="`Filter by ${ip}`" @click="filterByIp(ip)" />
            <span v-for="meta in alertTags(selectedGroup.base)" :key="`drawer-meta-${meta}`" class="sc-chip">{{ meta }}</span>
          </div>
        </section>

        <section class="drawer-panel drawer-panel--wide">
          <h6>Grouped Events</h6>
          <ul class="drawer-list" v-if="selectedGroup.items.length">
            <li v-for="item in selectedGroup.items" :key="item.id">
              <strong>{{ formatRelative(item.ts) }}</strong>
              <span class="text-muted"> · {{ item.source }}</span>
              <div class="text-sm text-secondary">{{ item.message }}</div>
            </li>
          </ul>
        </section>

        <section class="drawer-panel drawer-panel--wide">
          <details>
            <summary>Raw JSON</summary>
            <pre>{{ JSON.stringify(selectedGroup.items, null, 2) }}</pre>
          </details>
        </section>
      </div>
      <template #footer>
        <div class="d-flex flex-wrap gap-2">
          <AppButton variant="secondary" size="sm" icon="mdi mdi-check" label="Dismiss" @click="selectedGroup && markGroupRead(selectedGroup)" />
          <AppButton variant="secondary" size="sm" icon="mdi mdi-timer-outline" label="Snooze 1h" @click="selectedGroup && snoozeGroup(selectedGroup, 60 * 60 * 1000)" />
          <AppButton variant="secondary" size="sm" icon="mdi mdi-bell-cancel-outline" label="Mute rule" @click="selectedGroup && muteRule(selectedGroup)" />
          <AppButton variant="secondary" size="sm" icon="mdi mdi-filter-outline" label="Filter source" @click="selectedGroup && filterBySource(selectedGroup.base.source)" />
          <AppButton variant="destructive" size="sm" icon="mdi mdi-close" label="Close" @click="showDrawer = false" />
        </div>
      </template>
    </DetailDrawer>

    <DetailDrawer :model-value="showShortcutHelp" title="Alert Shortcuts" subtitle="Keyboard navigation" @update:model-value="showShortcutHelp = $event">
      <ul class="drawer-list">
        <li><span class="sc-kbd">↑</span> / <span class="sc-kbd">↓</span> move through visible alert groups</li>
        <li><span class="sc-kbd">J</span> / <span class="sc-kbd">K</span> Vim-style next or previous alert group</li>
        <li><span class="sc-kbd">Enter</span> open the focused alert group</li>
        <li><span class="sc-kbd">Space</span> toggle selection in multi-select mode</li>
        <li><span class="sc-kbd">Esc</span> close the drawer or shortcut panel</li>
        <li><span class="sc-kbd">?</span> show this help panel</li>
      </ul>
    </DetailDrawer>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard from '@/components/widgets/stat-card.vue'
import AppButton from '@/components/ui/app-button.vue'
import StatusBadge from '@/components/ui/status-badge.vue'
import FilterToolbar from '@/components/ui/filter-toolbar.vue'
import DetailDrawer from '@/components/ui/detail-drawer.vue'
import EmptyState from '@/components/ui/empty-state.vue'
import ErrorState from '@/components/ui/error-state.vue'
import TimeDisplay from '@/components/ui/time-display.vue'
import IpChip from '@/components/ui/ip-chip.vue'
import api from '@/services/api'
import {
  extractIPs,
  formatAlertMeta,
  formatRelativeTime,
  getDensityPreference,
  groupAlerts,
  loadSavedFilters,
  matchesQuery,
  saveSavedFilters,
  setDensityPreference,
  summarizeAlert
} from '@/utils/formatters'

const NOTIFICATION_KEY = 'sc_alert_notifications_enabled'
const SNOOZE_KEY = 'sc_alert_snoozed_rules'
const MUTE_KEY = 'sc_alert_muted_rules'

function loadJsonPreference (key, fallback) {
  try {
    return JSON.parse(localStorage.getItem(key) || JSON.stringify(fallback))
  } catch {
    return fallback
  }
}

export default {
  name: 'AlertsPage',
  components: {
    PageHeader,
    StatCard,
    AppButton,
    StatusBadge,
    FilterToolbar,
    DetailDrawer,
    EmptyState,
    ErrorState,
    TimeDisplay,
    IpChip
  },
  data () {
    return {
      loading: false,
      errorMessage: '',
      alerts: [],
      queuedAlerts: [],
      searchQuery: '',
      debouncedSearch: '',
      searchTimer: null,
      severityFilter: '',
      readFilter: 'all',
      sourceFilter: '',
      datePreset: '24h',
      customStart: '',
      customEnd: '',
      savedFilters: loadSavedFilters('alerts'),
      density: getDensityPreference(),
      multiSelect: false,
      selectedGroupIds: [],
      visibleCount: 25,
      focusedIndex: 0,
      showDrawer: false,
      selectedGroupId: null,
      expandedGroupId: null,
      showShortcutHelp: false,
      notificationsEnabled: localStorage.getItem(NOTIFICATION_KEY) === 'true',
      snoozedRules: loadJsonPreference(SNOOZE_KEY, {}),
      mutedRules: loadJsonPreference(MUTE_KEY, []),
      highlightedGroupIds: [],
      seenTimers: {},
      pollingTimer: null,
      observer: null,
      lastLoadedAt: null
    }
  },
  computed: {
    activeRange () {
      if (this.datePreset === 'all') return { start: null, end: null }
      if (this.datePreset === 'custom') {
        return {
          start: this.customStart ? new Date(this.customStart) : null,
          end: this.customEnd ? new Date(this.customEnd) : null
        }
      }
      const hours = { '15m': 0.25, '1h': 1, '6h': 6, '24h': 24, '7d': 168 }[this.datePreset] || 24
      return { start: new Date(Date.now() - hours * 60 * 60 * 1000), end: null }
    },
    sourceOptions () {
      return [...new Set(this.alerts.map(alert => alert.source).filter(Boolean))].sort()
    },
    groupedAlerts () {
      const now = Date.now()
      return groupAlerts(this.alerts)
        .filter(group => !this.mutedRules.includes(this.ruleKey(group.base)))
        .filter(group => (this.snoozedRules[this.ruleKey(group.base)] || 0) <= now)
    },
    filteredGroups () {
      return this.groupedAlerts.filter(group => {
        const base = group.base
        if (this.severityFilter && base.severity !== this.severityFilter) return false
        if (this.readFilter === 'read' && !group.read) return false
        if (this.readFilter === 'unread' && group.read) return false
        if (this.sourceFilter && base.source !== this.sourceFilter) return false
        if (this.activeRange.start && (group.latestTs * 1000) < this.activeRange.start.getTime()) return false
        if (this.activeRange.end && (group.latestTs * 1000) > this.activeRange.end.getTime()) return false
        return matchesQuery(base, this.debouncedSearch, {
          fields: ['type', 'source', 'message', 'ip', 'username'],
          operators: {
            source: entry => entry.source,
            severity: entry => entry.severity,
            ip: entry => entry.ip,
            user: entry => entry.username
          }
        })
      })
    },
    visibleGroups () {
      return this.filteredGroups.slice(0, this.visibleCount)
    },
    selectedGroup () {
      return this.filteredGroups.find(group => group.id === this.selectedGroupId) || null
    },
    resultLabel () {
      return this.filteredGroups.length === this.groupedAlerts.length
        ? `${this.groupedAlerts.length} grouped alerts`
        : `${this.filteredGroups.length} of ${this.groupedAlerts.length} grouped alerts`
    },
    activeChips () {
      const chips = []
      if (this.severityFilter) chips.push({ key: 'severity', label: `Severity: ${this.severityFilter}` })
      if (this.readFilter !== 'all') chips.push({ key: 'state', label: `State: ${this.readFilter}` })
      if (this.sourceFilter) chips.push({ key: 'source', label: `Source: ${this.sourceFilter}` })
      if (this.datePreset !== 'all') chips.push({ key: 'range', label: `Range: ${this.datePreset}` })
      if (this.debouncedSearch) chips.push({ key: 'search', label: `Search: ${this.debouncedSearch}` })
      return chips
    },
    lastUpdatedLabel () {
      return this.lastLoadedAt ? formatRelativeTime(this.lastLoadedAt) : 'Never'
    }
  },
  watch: {
    searchQuery () {
      clearTimeout(this.searchTimer)
      this.searchTimer = setTimeout(() => {
        this.debouncedSearch = this.searchQuery
      }, 150)
    },
    filteredGroups () {
      if (this.focusedIndex >= this.visibleGroups.length) {
        this.focusedIndex = Math.max(0, this.visibleGroups.length - 1)
      }
      this.$nextTick(() => this.observeSentinel())
    }
  },
  mounted () {
    this.applyDatePreset(this.datePreset)
    this.loadAlerts(true)
    this.startPolling()
    this.observeSentinel()
  },
  beforeUnmount () {
    clearTimeout(this.searchTimer)
    clearInterval(this.pollingTimer)
    Object.values(this.seenTimers).forEach(timer => clearTimeout(timer))
    this.disconnectObserver()
  },
  methods: {
    async loadAlerts (manual = false) {
      this.loading = manual && !this.alerts.length
      this.errorMessage = ''
      try {
        const { data } = await api.getAlerts({ limit: 300 })
        const nextAlerts = Array.isArray(data) ? data : (data.alerts || [])
        if (!this.alerts.length || manual) {
          this.alerts = nextAlerts
          this.highlightNewGroups(nextAlerts)
        } else {
          const existingIds = new Set(this.alerts.map(alert => alert.id))
          const newItems = nextAlerts.filter(alert => !existingIds.has(alert.id))
          if (newItems.length) {
            this.handleIncomingAlerts(newItems, nextAlerts)
          } else {
            this.alerts = nextAlerts
          }
        }
        this.lastLoadedAt = Date.now()
      } catch (error) {
        this.errorMessage = error.response?.data?.detail || error.message || 'Unable to load alerts.'
      } finally {
        this.loading = false
      }
    },
    handleIncomingAlerts (newItems, nextAlerts) {
      if (this.notificationsEnabled) this.notifyIfNeeded(newItems)
      if (this.isNearTop()) {
        this.alerts = nextAlerts
        this.highlightNewGroups(newItems)
      } else {
        this.queuedAlerts = newItems
      }
    },
    applyQueuedAlerts () {
      this.alerts = [...this.queuedAlerts, ...this.alerts].sort((left, right) => (right.ts || 0) - (left.ts || 0))
      this.highlightNewGroups(this.queuedAlerts)
      this.queuedAlerts = []
    },
    highlightNewGroups (incomingAlerts) {
      const incomingIds = new Set(incomingAlerts.map(alert => alert.id))
      const groupIds = groupAlerts(this.alerts).filter(group => group.items.some(item => incomingIds.has(item.id))).map(group => group.id)
      this.highlightedGroupIds = [...new Set([...this.highlightedGroupIds, ...groupIds])]
      window.setTimeout(() => {
        this.highlightedGroupIds = this.highlightedGroupIds.filter(id => !groupIds.includes(id))
      }, 4000)
    },
    startPolling () {
      clearInterval(this.pollingTimer)
      this.pollingTimer = window.setInterval(() => {
        if (!document.hidden) this.loadAlerts(false)
      }, 20000)
    },
    isNearTop () {
      const container = this.$refs.listContainer
      return !container || container.scrollTop < 80
    },
    observeSentinel () {
      this.disconnectObserver()
      if (!this.$refs.loadMoreSentinel) return
      this.observer = new IntersectionObserver(entries => {
        const [entry] = entries
        if (entry?.isIntersecting && this.visibleCount < this.filteredGroups.length) {
          this.visibleCount += 20
        }
      }, { root: this.$refs.listContainer, threshold: 0.2 })
      this.observer.observe(this.$refs.loadMoreSentinel)
    },
    disconnectObserver () {
      if (this.observer) {
        this.observer.disconnect()
        this.observer = null
      }
    },
    updateSearch (value) {
      this.searchQuery = value
    },
    applyDatePreset (preset) {
      this.datePreset = preset
      if (preset !== 'custom') {
        this.customStart = ''
        this.customEnd = ''
      }
    },
    clearFilters () {
      this.searchQuery = ''
      this.debouncedSearch = ''
      this.severityFilter = ''
      this.readFilter = 'all'
      this.sourceFilter = ''
      this.applyDatePreset('24h')
    },
    removeChip (key) {
      if (key === 'severity') this.severityFilter = ''
      if (key === 'state') this.readFilter = 'all'
      if (key === 'source') this.sourceFilter = ''
      if (key === 'range') this.applyDatePreset('all')
      if (key === 'search') {
        this.searchQuery = ''
        this.debouncedSearch = ''
      }
    },
    persistDensity () {
      setDensityPreference(this.density)
    },
    saveCurrentFilter () {
      const name = window.prompt('Name this saved filter')
      if (!name) return
      this.savedFilters = [
        {
          id: `${Date.now()}`,
          name,
          state: {
            searchQuery: this.searchQuery,
            severityFilter: this.severityFilter,
            readFilter: this.readFilter,
            sourceFilter: this.sourceFilter,
            datePreset: this.datePreset,
            customStart: this.customStart,
            customEnd: this.customEnd
          }
        },
        ...this.savedFilters
      ].slice(0, 10)
      saveSavedFilters('alerts', this.savedFilters)
    },
    applySavedFilter (filter) {
      Object.assign(this, filter.state)
      this.debouncedSearch = filter.state.searchQuery
    },
    deleteSavedFilter (id) {
      this.savedFilters = this.savedFilters.filter(filter => filter.id !== id)
      saveSavedFilters('alerts', this.savedFilters)
    },
    ruleKey (alert) {
      return `${alert.severity || 'unknown'}::${alert.type || 'alert'}::${alert.source || 'system'}`
    },
    countBySeverity (severity) {
      return this.alerts.filter(alert => alert.severity === severity).length
    },
    severityState (severity) {
      return { emergency: 'critical', critical: 'error', warning: 'warn', info: 'info' }[severity] || 'muted'
    },
    severityLabel (severity) {
      return severity ? severity.charAt(0).toUpperCase() + severity.slice(1) : 'Alert'
    },
    severityIcon (severity) {
      return {
        emergency: 'mdi mdi-alert-octagon-outline',
        critical: 'mdi mdi-alert-circle-outline',
        warning: 'mdi mdi-alert-outline',
        info: 'mdi mdi-information-outline'
      }[severity] || 'mdi mdi-bell-outline'
    },
    formatRelative (ts) {
      return formatRelativeTime(ts)
    },
    summarize (alert) {
      return summarizeAlert(alert)
    },
    alertTags (alert) {
      return formatAlertMeta(alert)
    },
    alertIps (alert) {
      const ips = extractIPs(alert.message || '')
      if (alert.ip && !ips.includes(alert.ip)) ips.unshift(alert.ip)
      return ips.slice(0, 3)
    },
    openGroup (group) {
      this.selectedGroupId = group.id
      this.showDrawer = true
      this.scheduleSeen(group, true)
    },
    toggleExpanded (groupId) {
      this.expandedGroupId = this.expandedGroupId === groupId ? null : groupId
    },
    toggleMultiSelect () {
      this.multiSelect = !this.multiSelect
      if (!this.multiSelect) this.selectedGroupIds = []
    },
    toggleGroupSelected (groupId) {
      if (this.selectedGroupIds.includes(groupId)) {
        this.selectedGroupIds = this.selectedGroupIds.filter(id => id !== groupId)
      } else {
        this.selectedGroupIds = [...this.selectedGroupIds, groupId]
      }
    },
    async markGroupRead (group) {
      const ids = group.items.filter(item => !item.read).map(item => item.id)
      if (!ids.length) {
        this.showDrawer = false
        return
      }
      try {
        await api.markAlertsAsRead(ids)
        this.alerts = this.alerts.map(alert => ids.includes(alert.id) ? { ...alert, read: true } : alert)
      } catch (error) {
        this.errorMessage = error.response?.data?.detail || error.message || 'Unable to mark alerts as read.'
      }
      this.showDrawer = false
    },
    async markAllRead () {
      const ids = this.alerts.filter(alert => !alert.read).map(alert => alert.id)
      if (!ids.length) return
      try {
        await api.markAlertsAsRead(ids)
        this.alerts = this.alerts.map(alert => ({ ...alert, read: true }))
      } catch (error) {
        this.errorMessage = error.response?.data?.detail || error.message || 'Unable to mark alerts as read.'
      }
    },
    async markSelectedRead () {
      const selected = this.filteredGroups.filter(group => this.selectedGroupIds.includes(group.id))
      for (const group of selected) {
        await this.markGroupRead(group)
      }
      this.selectedGroupIds = []
    },
    snoozeGroup (group, durationMs) {
      this.snoozedRules = { ...this.snoozedRules, [this.ruleKey(group.base)]: Date.now() + durationMs }
      localStorage.setItem(SNOOZE_KEY, JSON.stringify(this.snoozedRules))
      this.showDrawer = false
    },
    snoozeSelected (durationMs) {
      this.filteredGroups.filter(group => this.selectedGroupIds.includes(group.id)).forEach(group => this.snoozeGroup(group, durationMs))
      this.selectedGroupIds = []
    },
    muteRule (group) {
      this.mutedRules = [...new Set([...this.mutedRules, this.ruleKey(group.base)])]
      localStorage.setItem(MUTE_KEY, JSON.stringify(this.mutedRules))
      this.showDrawer = false
    },
    async blockSourceIp (group) {
      if (!group.base.ip) return
      try {
        await api.addFirewallRule({ port: 'any', action: 'deny', from: group.base.ip })
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Blocked ${group.base.ip}`, showConfirmButton: false, timer: 2200 })
      } catch (error) {
        this.errorMessage = error.response?.data?.detail || error.message || 'Unable to block IP.'
      }
    },
    exportGroup (group) {
      const blob = new Blob([JSON.stringify(group.items, null, 2)], { type: 'application/json;charset=utf-8' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `alert-group-${group.base.source || 'system'}-${Date.now()}.json`
      link.click()
    },
    exportAlerts (selectedOnly = false) {
      const groups = selectedOnly ? this.filteredGroups.filter(group => this.selectedGroupIds.includes(group.id)) : this.filteredGroups
      const blob = new Blob([JSON.stringify(groups.flatMap(group => group.items), null, 2)], { type: 'application/json;charset=utf-8' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `alerts-${Date.now()}.json`
      link.click()
    },
    scheduleSeen (group, immediate = false) {
      this.cancelSeen(group.id)
      this.seenTimers[group.id] = window.setTimeout(() => {
        this.markGroupRead(group)
        delete this.seenTimers[group.id]
      }, immediate ? 0 : 3000)
    },
    cancelSeen (groupId) {
      if (this.seenTimers[groupId]) {
        clearTimeout(this.seenTimers[groupId])
        delete this.seenTimers[groupId]
      }
    },
    toggleNotifications () {
      this.notificationsEnabled = !this.notificationsEnabled
      localStorage.setItem(NOTIFICATION_KEY, String(this.notificationsEnabled))
      if (this.notificationsEnabled && 'Notification' in window && Notification.permission === 'default') {
        Notification.requestPermission()
      }
    },
    notifyIfNeeded (alerts) {
      if (!('Notification' in window) || Notification.permission !== 'granted') return
      const critical = alerts.filter(alert => ['critical', 'emergency'].includes(alert.severity))
      if (!critical.length) return
      const latest = critical[0]
      new Notification(`SentinelCore ${this.severityLabel(latest.severity)} alert`, {
        body: latest.message,
        tag: `sentinel-alert-${latest.id}`
      })
    },
    navigateDrawer (step) {
      if (!this.showDrawer || !this.selectedGroup) return
      const index = this.filteredGroups.findIndex(group => group.id === this.selectedGroup.id)
      const next = this.filteredGroups[index + step]
      if (next) this.openGroup(next)
    },
    filterByIp (ip) {
      this.searchQuery = `ip:${ip}`
      this.debouncedSearch = this.searchQuery
      this.showDrawer = false
    },
    filterBySource (source) {
      this.sourceFilter = source || ''
      this.showDrawer = false
    },
    handleKeyboardNavigation (event) {
      if (event.key === '?') {
        event.preventDefault()
        this.showShortcutHelp = true
        return
      }
      if (event.key === 'Escape') {
        this.showDrawer = false
        this.showShortcutHelp = false
        return
      }
      if (!this.visibleGroups.length) return
      if (['ArrowDown', 'j', 'J'].includes(event.key)) {
        event.preventDefault()
        this.focusedIndex = Math.min(this.focusedIndex + 1, this.visibleGroups.length - 1)
        return
      }
      if (['ArrowUp', 'k', 'K'].includes(event.key)) {
        event.preventDefault()
        this.focusedIndex = Math.max(this.focusedIndex - 1, 0)
        return
      }
      if (event.key === 'Enter') {
        event.preventDefault()
        this.openGroup(this.visibleGroups[this.focusedIndex])
        return
      }
      if (event.key === ' ' && this.multiSelect) {
        event.preventDefault()
        this.toggleGroupSelected(this.visibleGroups[this.focusedIndex].id)
      }
    }
  }
}
</script>

<style scoped>
.alerts-page {
  overscroll-behavior: contain;
}

.updates-banner,
.bulk-bar {
  border-color: var(--state-info-border);
  background: var(--state-info-bg);
  color: var(--state-info-fg);
}

.stat-row {
  --bs-gutter-y: var(--space-16);
}

.toolbar-select,
.toolbar-input {
  width: 150px;
}

.toolbar-select--narrow {
  width: 120px;
}

.toolbar-menu {
  position: relative;
}

.toolbar-menu summary {
  list-style: none;
}

.toolbar-menu summary::-webkit-details-marker {
  display: none;
}

.toolbar-menu[open] .toolbar-menu__body {
  display: grid;
}

.toolbar-menu__body {
  display: none;
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 220px;
  padding: var(--space-8);
  background: var(--surface-1);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  z-index: 6;
}

.toolbar-menu__empty {
  padding: var(--space-8);
  color: var(--text-tertiary);
  font-size: var(--font-size-12);
}

.saved-filter-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-8);
}

.alerts-shell {
  overflow: hidden;
}

.alerts-list-wrap {
  max-height: 72vh;
  overflow: auto;
}

.skeleton-wrap {
  display: grid;
  gap: var(--space-12);
  padding: var(--space-20);
}

.skeleton-row {
  height: 98px;
  border-radius: var(--radius-lg);
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.04), rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.04));
  background-size: 200% 100%;
  animation: alert-skeleton 1.5s linear infinite;
}

.alerts-list {
  display: grid;
}

.alert-row {
  position: relative;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-12);
  padding: var(--space-16) var(--space-20);
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background-color 0.2s ease, transform 0.2s ease;
}

.alert-row::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: transparent;
}

.alert-row.unread::before {
  background: var(--accent);
}

.alert-row.highlighted {
  background: color-mix(in srgb, var(--accent-muted) 70%, transparent);
}

.alert-row.active,
.alert-row:hover {
  background: var(--surface-2);
}

.alert-row.compact {
  padding-top: var(--space-12);
  padding-bottom: var(--space-12);
}

.alert-checkbox {
  padding-top: 4px;
}

.alert-row__content {
  display: grid;
  gap: var(--space-12);
}

.alert-row__top,
.alert-top-main,
.alert-top-actions,
.alert-row__chips {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  flex-wrap: wrap;
}

.alert-row__top {
  justify-content: space-between;
}

.alert-type {
  font-size: var(--font-size-16);
  font-weight: 600;
  color: var(--text-primary);
}

.alert-source,
.alert-time {
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.alert-row__summary {
  font-size: var(--font-size-14);
  color: var(--text-primary);
}

.alert-expansion {
  display: grid;
  gap: var(--space-8);
  padding: var(--space-12);
  border-radius: var(--radius-lg);
  background: var(--surface-3);
}

.alert-expansion__item {
  padding-bottom: var(--space-8);
  border-bottom: 1px solid var(--border-subtle);
}

.alert-expansion__item:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.alert-menu {
  position: relative;
}

.alert-menu__body {
  right: 0;
  left: auto;
}

.load-more-sentinel {
  height: 1px;
}

.drawer-grid {
  display: grid;
  gap: var(--space-16);
}

.drawer-panel {
  display: grid;
  gap: var(--space-12);
  padding: var(--space-16);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--surface-2);
}

.drawer-panel--wide {
  grid-column: 1 / -1;
}

.drawer-meta-row {
  display: flex;
  justify-content: space-between;
  gap: var(--space-12);
  align-items: flex-start;
}

.drawer-meta-row > span:first-child {
  color: var(--text-tertiary);
  min-width: 90px;
}

.drawer-stack,
.drawer-list,
.drawer-chip-row {
  display: grid;
  gap: var(--space-8);
}

.drawer-copy {
  margin: 0;
  color: var(--text-primary);
}

.drawer-chip-row {
  display: flex;
  flex-wrap: wrap;
}

.drawer-list {
  margin: 0;
  padding-left: 18px;
  color: var(--text-secondary);
}

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

@keyframes alert-skeleton {
  from { background-position: 0% 0%; }
  to { background-position: 200% 0%; }
}

@media (max-width: 767px) {
  .toolbar-select,
  .toolbar-input,
  .toolbar-select--narrow {
    width: 100%;
  }

  .alert-row {
    grid-template-columns: 1fr;
  }
}
</style>
