<template>
  <div class="sc-page-shell sc-focus-ring audit-page" tabindex="0" @keydown="handleKeyboardNavigation">
    <PageHeader title="Audit Logs" icon="mdi mdi-history" :items="[{ text: 'Audit Logs', active: true, icon: 'mdi mdi-file-document-check' }]">
      <template #actions>
        <AppButton variant="secondary" size="md" icon="mdi mdi-refresh" :loading="loading" label="Refresh" @click="loadAuditLogs" />
        <AppButton variant="secondary" size="md" icon="mdi mdi-keyboard-outline" label="Shortcuts" @click="showShortcutHelp = true" />
        <AppButton variant="primary" size="md" icon="mdi mdi-download" label="Export" @click="exportLogs" />
      </template>
    </PageHeader>

    <div class="row g-3 stat-row">
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard
          label="Most Frequent IP"
          :value="topIp.ip || '—'"
          :sub="`${topIp.count} attempts`"
          icon="mdi mdi-lan"
          tone="info"
          :clickable="!!topIp.ip"
          @click="topIp.ip && filterByIp(topIp.ip)"
        />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard
          label="Successful IPs"
          :value="successIps.length"
          sub="Unique successful sources"
          icon="mdi mdi-check-circle-outline"
          tone="ok"
          clickable
          @click="filterResult = 'success'"
        />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard
          label="Failed IPs"
          :value="failedIps.length"
          sub="IPs with failed auth"
          icon="mdi mdi-alert-outline"
          tone="warn"
          clickable
          @click="filterResult = 'failure'"
        />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard
          label="Brute Force"
          :value="bruteForceIps.length"
          sub="Flagged IPs"
          icon="mdi mdi-shield-alert-outline"
          :tone="bruteForceIps.length ? 'critical' : 'default'"
          clickable
          @click="applyBruteForceFilter"
        />
      </div>
    </div>

    <FilterToolbar
      :search-query="searchQuery"
      search-placeholder="Search user, ip, reason, device… Use user:alice ip:1.2.3.4 result:failed"
      :active-chips="activeChips"
      :result-label="resultLabel"
      @update:search-query="updateSearch"
      @remove-chip="removeChip"
      @clear-all="clearFilters"
    >
      <template #controls>
        <select v-model="filterUser" class="form-select sc-focus-ring toolbar-select">
          <option value="">All users</option>
          <option v-for="user in uniqueUsers" :key="user" :value="user">{{ user }}</option>
        </select>
        <select v-model="filterResult" class="form-select sc-focus-ring toolbar-select">
          <option value="">All results</option>
          <option value="success">Success</option>
          <option value="failure">Failure</option>
        </select>
        <select v-model="datePreset" class="form-select sc-focus-ring toolbar-select" @change="applyDatePreset(datePreset)">
          <option value="15m">Last 15m</option>
          <option value="1h">Last 1h</option>
          <option value="6h">Last 6h</option>
          <option value="24h">Last 24h</option>
          <option value="7d">Last 7d</option>
          <option value="custom">Custom</option>
          <option value="all">All time</option>
        </select>
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
        <details class="toolbar-menu">
          <summary class="sc-button sc-button--secondary sc-button--md">
            <i class="mdi mdi-table-column"></i>
            Columns
          </summary>
          <div class="toolbar-menu__body">
            <label v-for="column in columnOptions" :key="column.key" class="toolbar-check">
              <input v-model="visibleColumns" type="checkbox" :value="column.key" />
              <span>{{ column.label }}</span>
            </label>
          </div>
        </details>
      </template>
      <template #meta>
        <span class="sc-inline-note">{{ resultLabel }}</span>
        <span class="sc-inline-note">Last updated {{ lastUpdatedLabel }}</span>
        <select v-model="timeFormat" class="form-select sc-focus-ring toolbar-select toolbar-select--narrow" @change="persistTimeFormat">
          <option value="relative">Relative</option>
          <option value="absolute">Absolute</option>
          <option value="both">Both</option>
        </select>
        <AppButton :variant="multiSelect ? 'primary' : 'secondary'" size="md" icon="mdi mdi-checkbox-multiple-marked-outline" :label="multiSelect ? 'Selecting' : 'Select'" @click="toggleMultiSelect" />
      </template>
    </FilterToolbar>

    <ErrorState v-if="errorMessage" title="Audit log refresh failed" :description="errorMessage">
      <template #actions>
        <AppButton variant="secondary" size="sm" icon="mdi mdi-refresh" label="Retry" @click="loadAuditLogs" />
      </template>
    </ErrorState>

    <div v-if="multiSelect && selectedIds.length" class="sc-inline-error bulk-bar">
      <div class="d-flex flex-wrap gap-2 align-items-center">
        <strong>{{ selectedIds.length }} selected</strong>
        <span>Export or inspect the selected events.</span>
      </div>
      <div class="d-flex flex-wrap gap-2">
        <AppButton variant="secondary" size="sm" icon="mdi mdi-download" label="Export selected" @click="exportLogs(true)" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-close" label="Clear" @click="selectedIds = []" />
      </div>
    </div>

    <div class="sc-surface data-shell">
      <div class="card-body p-0">
        <div v-if="loading" class="skeleton-wrap" aria-busy="true">
          <div v-for="n in 8" :key="n" class="skeleton-row"></div>
        </div>

        <EmptyState
          v-else-if="!logs.length"
          icon="mdi mdi-file-search-outline"
          title="No audit activity yet"
          description="Authentication and terminal events will appear here once users start interacting with the admin panel."
        >
          <template #actions>
            <AppButton variant="secondary" size="md" icon="mdi mdi-refresh" label="Refresh" @click="loadAuditLogs" />
          </template>
        </EmptyState>

        <EmptyState
          v-else-if="!pagedLogs.length"
          icon="mdi mdi-filter-off-outline"
          title="No results match these filters"
          description="Try widening the time range or clear the active filters to see more audit activity."
        >
          <template #actions>
            <AppButton variant="secondary" size="md" icon="mdi mdi-filter-remove-outline" label="Clear filters" @click="clearFilters" />
          </template>
        </EmptyState>

        <template v-else>
          <div class="d-none d-md-block">
            <table class="table audit-table mb-0" :class="{ compact: density === 'compact' }">
              <thead>
                <tr>
                  <th v-if="multiSelect" class="checkbox-col">
                    <input type="checkbox" :checked="allSelected" @change="toggleSelectAll($event.target.checked)" />
                  </th>
                  <th>Time</th>
                  <th>User</th>
                  <th>IP</th>
                  <th v-if="isVisible('reason')">Reason</th>
                  <th v-if="isVisible('device')">Device</th>
                  <th>Result</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(log, index) in pagedLogs"
                  :key="log.id"
                  class="audit-row"
                  :class="{ active: focusedIndex === index, 'audit-row--bruteforce': isBruteForce(log) }"
                  @click="openLog(log)"
                >
                  <td v-if="multiSelect" class="checkbox-col" @click.stop>
                    <input type="checkbox" :checked="selectedIds.includes(log.id)" @change="toggleSelected(log.id)" />
                  </td>
                  <td>
                    <TimeDisplay :value="log.ts" :mode="timeFormat" />
                  </td>
                  <td>
                    <UserChip
                      :user="log.username"
                      :email="userMeta(log.username).email"
                      :role="userMeta(log.username).role"
                      :recent-count="recentCountForUser(log.username)"
                      @click.stop="filterUser = log.username"
                    />
                  </td>
                  <td>
                    <IpChip :ip="log.ip" :tooltip="ipTooltip(log.ip)" @click.stop="filterByIp(log.ip)" />
                  </td>
                  <td v-if="isVisible('reason')" class="reason-cell">
                    <CodeLabel :code="log.reason" />
                  </td>
                  <td v-if="isVisible('device')" :title="log.user_agent || 'Unknown user agent'">
                    {{ deviceLabel(log.user_agent) }}
                  </td>
                  <td>
                    <div class="d-flex align-items-center gap-2 flex-wrap">
                      <StatusBadge :state="log.success ? 'ok' : (isBruteForce(log) ? 'critical' : 'error')" :label="log.success ? 'Success' : 'Failed'" />
                      <StatusBadge v-if="isBruteForce(log)" state="critical" label="Brute Force" />
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <ul class="d-md-none mobile-audit-list">
            <li v-for="(log, index) in pagedLogs" :key="`mobile-${log.id}`" class="mobile-audit-card" :class="{ 'mobile-audit-card--bruteforce': isBruteForce(log), active: focusedIndex === index }" @click="openLog(log)">
              <div class="mobile-audit-card__top">
                <TimeDisplay :value="log.ts" :mode="timeFormat" />
                <StatusBadge :state="log.success ? 'ok' : 'error'" :label="log.success ? 'Success' : 'Failed'" />
              </div>
              <div class="mobile-audit-card__body">
                <UserChip :user="log.username" @click.stop="filterUser = log.username" />
                <IpChip :ip="log.ip" :tooltip="ipTooltip(log.ip)" @click.stop="filterByIp(log.ip)" />
              </div>
              <div class="mobile-audit-card__meta">
                <CodeLabel :code="log.reason" />
                <span class="text-muted text-sm">{{ deviceLabel(log.user_agent) }}</span>
              </div>
            </li>
          </ul>

          <div class="table-footer">
            <div class="d-flex flex-wrap gap-2 align-items-center">
              <span class="text-muted text-sm">{{ resultLabel }}</span>
              <select v-model.number="pageSize" class="form-select sc-focus-ring toolbar-select toolbar-select--narrow">
                <option :value="25">25 / page</option>
                <option :value="50">50 / page</option>
                <option :value="100">100 / page</option>
                <option :value="250">250 / page</option>
              </select>
              <select v-model="density" class="form-select sc-focus-ring toolbar-select toolbar-select--narrow" @change="persistDensity">
                <option value="comfortable">Comfortable</option>
                <option value="compact">Compact</option>
              </select>
            </div>
            <div class="d-flex align-items-center gap-2">
              <span class="text-muted text-sm">Page {{ currentPage }} / {{ totalPages }}</span>
              <AppButton variant="secondary" size="sm" icon="mdi mdi-chevron-left" aria-label="Previous page" :disabled="currentPage === 1" icon-only @click="currentPage--" />
              <AppButton variant="secondary" size="sm" icon="mdi mdi-chevron-right" aria-label="Next page" :disabled="currentPage === totalPages" icon-only @click="currentPage++" />
            </div>
          </div>
        </template>
      </div>
    </div>

    <DetailDrawer
      :model-value="showDrawer"
      :title="selectedLog ? `${selectedLog.username || 'Unknown user'} · ${selectedLog.ip}` : 'Audit event'"
      :subtitle="selectedLog ? reasonMeta(selectedLog.reason).label : ''"
      @update:model-value="showDrawer = $event"
      @navigate="navigateDrawer"
    >
      <template #nav>
        <AppButton variant="ghost" size="sm" icon="mdi mdi-chevron-left" aria-label="Previous record" icon-only @click="navigateDrawer(-1)" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-chevron-right" aria-label="Next record" icon-only @click="navigateDrawer(1)" />
      </template>

      <div v-if="selectedLog" class="drawer-grid">
        <section class="drawer-panel">
          <h6>Event Summary</h6>
          <div class="drawer-stack">
            <div class="drawer-meta-row"><span>Time</span><TimeDisplay :value="selectedLog.ts" :mode="timeFormat" /></div>
            <div class="drawer-meta-row"><span>User</span><UserChip :user="selectedLog.username" /></div>
            <div class="drawer-meta-row"><span>IP</span><IpChip :ip="selectedLog.ip" :tooltip="ipTooltip(selectedLog.ip)" @click="filterByIp(selectedLog.ip)" /></div>
            <div class="drawer-meta-row"><span>Result</span><StatusBadge :state="selectedLog.success ? 'ok' : 'error'" :label="selectedLog.success ? 'Success' : 'Failed'" /></div>
            <div class="drawer-meta-row"><span>Reason</span><CodeLabel :code="selectedLog.reason" /></div>
            <div class="drawer-meta-row"><span>Device</span><span>{{ deviceLabel(selectedLog.user_agent) }}</span></div>
          </div>
        </section>

        <section class="drawer-panel">
          <h6>Decision Trail</h6>
          <ul class="drawer-list">
            <li>Outcome: {{ selectedLog.success ? 'Request allowed' : 'Request denied' }}</li>
            <li>Reason: {{ reasonMeta(selectedLog.reason).description }}</li>
            <li v-if="isBruteForce(selectedLog)">Brute-force heuristics were triggered for this IP in the active window.</li>
            <li v-else>Brute-force heuristics were not triggered for this record.</li>
          </ul>
        </section>

        <section class="drawer-panel">
          <h6>IP Intelligence</h6>
          <div v-if="ipEntry(selectedLog.ip).loading" class="text-muted">Loading IP intelligence…</div>
          <div v-else-if="ipEntry(selectedLog.ip).error" class="text-muted">{{ ipEntry(selectedLog.ip).error }}</div>
          <div v-else class="drawer-stack">
            <div class="drawer-meta-row"><span>Country</span><span>{{ ipEntry(selectedLog.ip).data.country || '—' }}</span></div>
            <div class="drawer-meta-row"><span>Region</span><span>{{ ipEntry(selectedLog.ip).data.region || '—' }}</span></div>
            <div class="drawer-meta-row"><span>City</span><span>{{ ipEntry(selectedLog.ip).data.city || '—' }}</span></div>
            <div class="drawer-meta-row"><span>ASN</span><span>{{ ipEntry(selectedLog.ip).data.asn || '—' }}</span></div>
            <div class="drawer-meta-row"><span>Org</span><span>{{ ipEntry(selectedLog.ip).data.org || ipEntry(selectedLog.ip).data.isp || '—' }}</span></div>
            <div class="drawer-meta-row"><span>Timezone</span><span>{{ ipEntry(selectedLog.ip).data.timezone || '—' }}</span></div>
          </div>
        </section>

        <section class="drawer-panel drawer-panel--wide">
          <h6>Related Events (±5m)</h6>
          <ul class="drawer-list" v-if="relatedLogs.length">
            <li v-for="item in relatedLogs" :key="item.id">
              <strong>{{ item.username }}</strong>
              <span class="text-muted"> · {{ formatRelative(item.ts) }}</span>
              <span> · {{ reasonMeta(item.reason).label }}</span>
            </li>
          </ul>
          <div v-else class="text-muted">No related events in the active history window.</div>
        </section>

        <section class="drawer-panel drawer-panel--wide">
          <details>
            <summary>Raw JSON</summary>
            <pre>{{ JSON.stringify(selectedLog, null, 2) }}</pre>
          </details>
        </section>
      </div>
      <template #footer>
        <div class="d-flex flex-wrap gap-2">
          <AppButton variant="secondary" size="sm" icon="mdi mdi-filter-outline" label="Filter user" @click="filterUser = selectedLog.username; showDrawer = false" />
          <AppButton variant="secondary" size="sm" icon="mdi mdi-lan" label="Filter IP" @click="filterByIp(selectedLog.ip); showDrawer = false" />
          <AppButton variant="destructive" size="sm" icon="mdi mdi-close" label="Close" @click="showDrawer = false" />
        </div>
      </template>
    </DetailDrawer>

    <DetailDrawer :model-value="showShortcutHelp" title="Audit Log Shortcuts" subtitle="Keyboard navigation" @update:model-value="showShortcutHelp = $event">
      <ul class="drawer-list">
        <li><span class="sc-kbd">↑</span> / <span class="sc-kbd">↓</span> move through visible rows</li>
        <li><span class="sc-kbd">J</span> / <span class="sc-kbd">K</span> Vim-style next/previous row</li>
        <li><span class="sc-kbd">Enter</span> open the focused row</li>
        <li><span class="sc-kbd">Space</span> toggle selection in multi-select mode</li>
        <li><span class="sc-kbd">Esc</span> close the detail drawer</li>
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
import UserChip from '@/components/ui/user-chip.vue'
import IpChip from '@/components/ui/ip-chip.vue'
import CodeLabel from '@/components/ui/code-label.vue'
import api from '@/services/api'
import {
  formatRelativeTime,
  formatTimestamp,
  getDensityPreference,
  getReasonMeta,
  getTimeFormatPreference,
  loadSavedFilters,
  matchesQuery,
  parseUserAgent,
  saveSavedFilters,
  setDensityPreference,
  setTimeFormatPreference
} from '@/utils/formatters'

const defaultColumns = ['reason', 'device']

export default {
  name: 'AuditLogsPage',
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
    UserChip,
    IpChip,
    CodeLabel
  },
  data () {
    return {
      loading: false,
      errorMessage: '',
      logs: [],
      users: [],
      searchQuery: '',
      debouncedSearch: '',
      searchTimer: null,
      filterUser: '',
      filterResult: '',
      datePreset: '24h',
      customStart: '',
      customEnd: '',
      currentPage: 1,
      pageSize: 50,
      density: getDensityPreference(),
      timeFormat: getTimeFormatPreference(),
      visibleColumns: [...defaultColumns],
      multiSelect: false,
      selectedIds: [],
      savedFilters: loadSavedFilters('audit_logs'),
      showDrawer: false,
      selectedLogId: null,
      showShortcutHelp: false,
      focusedIndex: 0,
      ipCache: {},
      lastLoadedAt: null,
      columnOptions: [
        { key: 'reason', label: 'Reason' },
        { key: 'device', label: 'Device' }
      ]
    }
  },
  computed: {
    uniqueUsers () {
      return [...new Set(this.logs.map(log => log.username).filter(Boolean))].sort()
    },
    filteredLogs () {
      const range = this.activeRange
      return this.logs.filter(log => {
        if (this.filterUser && log.username !== this.filterUser) return false
        if (this.filterResult) {
          const wanted = this.filterResult === 'success'
          if (log.success !== wanted) return false
        }
        if (range.start && log.ts * 1000 < range.start.getTime()) return false
        if (range.end && log.ts * 1000 > range.end.getTime()) return false
        return matchesQuery(log, this.debouncedSearch, {
          fields: ['username', 'ip', 'reason', 'user_agent'],
          operators: {
            user: entry => entry.username,
            ip: entry => entry.ip,
            result: entry => (entry.success ? 'success' : 'failed'),
            reason: entry => entry.reason
          }
        })
      })
    },
    pagedLogs () {
      const start = (this.currentPage - 1) * this.pageSize
      return this.filteredLogs.slice(start, start + this.pageSize)
    },
    totalPages () {
      return Math.max(1, Math.ceil(this.filteredLogs.length / this.pageSize))
    },
    selectedLog () {
      return this.logs.find(log => log.id === this.selectedLogId) || null
    },
    relatedLogs () {
      if (!this.selectedLog) return []
      return this.logs
        .filter(log => log.id !== this.selectedLog.id)
        .filter(log => Math.abs((log.ts || 0) - (this.selectedLog.ts || 0)) <= 300)
        .filter(log => log.username === this.selectedLog.username || log.ip === this.selectedLog.ip)
        .slice(0, 10)
    },
    ipCounts () {
      return this.logs.reduce((acc, log) => {
        if (log.ip) acc[log.ip] = (acc[log.ip] || 0) + 1
        return acc
      }, {})
    },
    topIp () {
      const entries = Object.entries(this.ipCounts)
      if (!entries.length) return { ip: '', count: 0 }
      const [ip, count] = entries.sort((left, right) => right[1] - left[1])[0]
      return { ip, count }
    },
    successIps () {
      return [...new Set(this.logs.filter(log => log.success && log.ip).map(log => log.ip))]
    },
    failedIps () {
      return [...new Set(this.logs.filter(log => !log.success && log.ip).map(log => log.ip))]
    },
    bruteForceIps () {
      return [...new Set(this.logs.filter(log => this.isBruteForce(log)).map(log => log.ip))]
    },
    activeRange () {
      if (this.datePreset === 'all') return { start: null, end: null }
      if (this.datePreset === 'custom') {
        return {
          start: this.customStart ? new Date(this.customStart) : null,
          end: this.customEnd ? new Date(this.customEnd) : null
        }
      }
      const hours = { '15m': 0.25, '1h': 1, '6h': 6, '24h': 24, '7d': 168 }[this.datePreset] || 24
      return {
        start: new Date(Date.now() - hours * 60 * 60 * 1000),
        end: null
      }
    },
    resultLabel () {
      return this.filteredLogs.length === this.logs.length
        ? `${this.logs.length} records`
        : `${this.filteredLogs.length} of ${this.logs.length} records`
    },
    lastUpdatedLabel () {
      return this.lastLoadedAt ? formatRelativeTime(this.lastLoadedAt) : 'Never'
    },
    activeChips () {
      const chips = []
      if (this.filterUser) chips.push({ key: 'user', label: `User: ${this.filterUser}` })
      if (this.filterResult) chips.push({ key: 'result', label: `Result: ${this.filterResult}` })
      if (this.datePreset !== 'all') chips.push({ key: 'range', label: `Range: ${this.datePreset}` })
      if (this.debouncedSearch) chips.push({ key: 'search', label: `Search: ${this.debouncedSearch}` })
      return chips
    },
    allSelected () {
      return this.pagedLogs.length > 0 && this.pagedLogs.every(log => this.selectedIds.includes(log.id))
    }
  },
  watch: {
    searchQuery () {
      clearTimeout(this.searchTimer)
      this.searchTimer = setTimeout(() => {
        this.debouncedSearch = this.searchQuery
      }, 150)
    },
    filterUser () { this.currentPage = 1 },
    filterResult () { this.currentPage = 1 },
    datePreset () { this.currentPage = 1 },
    customStart () { this.currentPage = 1 },
    customEnd () { this.currentPage = 1 },
    debouncedSearch () { this.currentPage = 1 },
    pageSize () { this.currentPage = 1 },
    filteredLogs () {
      if (this.currentPage > this.totalPages) this.currentPage = this.totalPages
      if (this.focusedIndex >= this.pagedLogs.length) this.focusedIndex = Math.max(0, this.pagedLogs.length - 1)
    }
  },
  mounted () {
    this.applyDatePreset(this.datePreset)
    this.loadAuditLogs()
    this.loadUsers()
  },
  beforeUnmount () {
    clearTimeout(this.searchTimer)
  },
  methods: {
    async loadAuditLogs () {
      this.loading = true
      this.errorMessage = ''
      try {
        const { data } = await api.getAuditLogs({ limit: 500 })
        this.logs = Array.isArray(data) ? data : (data.logs || [])
        this.lastLoadedAt = Date.now()
      } catch (error) {
        this.errorMessage = error.response?.data?.detail || error.message || 'Unable to fetch audit logs.'
      } finally {
        this.loading = false
      }
    },
    async loadUsers () {
      try {
        const { data } = await api.getUsers()
        this.users = Array.isArray(data) ? data : []
      } catch {
        this.users = []
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
    applyBruteForceFilter () {
      this.filterResult = 'failure'
      this.searchQuery = 'reason:rate_limited'
      this.debouncedSearch = this.searchQuery
    },
    clearFilters () {
      this.searchQuery = ''
      this.debouncedSearch = ''
      this.filterUser = ''
      this.filterResult = ''
      this.applyDatePreset('24h')
    },
    removeChip (key) {
      if (key === 'user') this.filterUser = ''
      if (key === 'result') this.filterResult = ''
      if (key === 'range') this.applyDatePreset('all')
      if (key === 'search') {
        this.searchQuery = ''
        this.debouncedSearch = ''
      }
    },
    persistDensity () {
      setDensityPreference(this.density)
    },
    persistTimeFormat () {
      setTimeFormatPreference(this.timeFormat)
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
            filterUser: this.filterUser,
            filterResult: this.filterResult,
            datePreset: this.datePreset,
            customStart: this.customStart,
            customEnd: this.customEnd
          }
        },
        ...this.savedFilters
      ].slice(0, 10)
      saveSavedFilters('audit_logs', this.savedFilters)
    },
    applySavedFilter (filter) {
      Object.assign(this, filter.state)
      this.debouncedSearch = filter.state.searchQuery
    },
    deleteSavedFilter (id) {
      this.savedFilters = this.savedFilters.filter(filter => filter.id !== id)
      saveSavedFilters('audit_logs', this.savedFilters)
    },
    userMeta (username) {
      return this.users.find(user => user.username === username) || {}
    },
    recentCountForUser (username) {
      return this.logs.filter(log => log.username === username).length
    },
    deviceLabel (userAgent) {
      return parseUserAgent(userAgent).label
    },
    reasonMeta (reason) {
      return getReasonMeta(reason)
    },
    formatRelative (ts) {
      return formatTimestamp(ts).primary
    },
    isVisible (column) {
      return this.visibleColumns.includes(column)
    },
    isBruteForce (log) {
      if (!log || !log.ip) return false
      if (log.reason === 'rate_limited') return true
      const windowStart = (log.ts || 0) - 300
      const failures = this.logs.filter(entry => entry.ip === log.ip && !entry.success && (entry.ts || 0) >= windowStart && (entry.ts || 0) <= (log.ts || 0)).length
      return failures >= 5
    },
    toggleMultiSelect () {
      this.multiSelect = !this.multiSelect
      if (!this.multiSelect) this.selectedIds = []
    },
    toggleSelected (id) {
      if (this.selectedIds.includes(id)) {
        this.selectedIds = this.selectedIds.filter(item => item !== id)
      } else {
        this.selectedIds = [...this.selectedIds, id]
      }
    },
    toggleSelectAll (checked) {
      this.selectedIds = checked ? this.pagedLogs.map(log => log.id) : []
    },
    openLog (log) {
      this.selectedLogId = log.id
      this.showDrawer = true
      this.fetchIpInfo(log.ip)
    },
    navigateDrawer (step) {
      if (!this.showDrawer || !this.selectedLog) return
      const index = this.pagedLogs.findIndex(log => log.id === this.selectedLog.id)
      if (index === -1) return
      const next = this.pagedLogs[index + step]
      if (next) this.openLog(next)
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
      if (!this.pagedLogs.length) return
      if (['ArrowDown', 'j', 'J'].includes(event.key)) {
        event.preventDefault()
        this.focusedIndex = Math.min(this.focusedIndex + 1, this.pagedLogs.length - 1)
        return
      }
      if (['ArrowUp', 'k', 'K'].includes(event.key)) {
        event.preventDefault()
        this.focusedIndex = Math.max(this.focusedIndex - 1, 0)
        return
      }
      if (event.key === 'Enter') {
        event.preventDefault()
        this.openLog(this.pagedLogs[this.focusedIndex])
        return
      }
      if (event.key === ' ' && this.multiSelect) {
        event.preventDefault()
        this.toggleSelected(this.pagedLogs[this.focusedIndex].id)
      }
    },
    filterByIp (ip) {
      this.searchQuery = `ip:${ip}`
      this.debouncedSearch = this.searchQuery
    },
    ipEntry (ip) {
      return this.ipCache[ip] || { loading: false, error: '', data: {} }
    },
    ipTooltip (ip) {
      const entry = this.ipEntry(ip)
      if (entry.loading) return `${ip}\nLoading reputation…`
      if (entry.data.country || entry.data.asn) {
        return `${ip}\n${entry.data.country || 'Unknown country'} · ${entry.data.asn || 'Unknown ASN'}`
      }
      return `${ip}\nFilter by this IP`
    },
    async fetchIpInfo (ip) {
      if (!ip || this.ipCache[ip]?.loading || this.ipCache[ip]?.data?.country || this.ipCache[ip]?.error) return
      this.ipCache = { ...this.ipCache, [ip]: { loading: true, error: '', data: {} } }
      try {
        const token = import.meta.env.VITE_IPINFO_TOKEN || ''
        const url = token ? `https://ipinfo.io/${ip}/json?token=${token}` : `https://ipinfo.io/${ip}/json`
        const response = await fetch(url)
        if (!response.ok) throw new Error('Unable to fetch IP intelligence')
        const data = await response.json()
        this.ipCache = { ...this.ipCache, [ip]: { loading: false, error: '', data } }
      } catch {
        this.ipCache = { ...this.ipCache, [ip]: { loading: false, error: 'IP intelligence unavailable right now.', data: {} } }
      }
    },
    exportLogs (selectedOnly = false) {
      const rows = selectedOnly ? this.filteredLogs.filter(log => this.selectedIds.includes(log.id)) : this.filteredLogs
      const header = ['time', 'user', 'ip', 'reason', 'device', 'result']
      const body = rows.map(log => [
        formatTimestamp(log.ts, { mode: 'absolute' }).primary,
        log.username,
        log.ip,
        log.reason,
        this.deviceLabel(log.user_agent),
        log.success ? 'success' : 'failed'
      ].join(','))
      const blob = new Blob([[header.join(','), ...body].join('\n')], { type: 'text/csv;charset=utf-8' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `audit-${Date.now()}.csv`
      link.click()
    }
  }
}
</script>

<style scoped>
.audit-page {
  overscroll-behavior: contain;
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
  z-index: 5;
}

.toolbar-menu__empty {
  padding: var(--space-8);
  color: var(--text-tertiary);
  font-size: var(--font-size-12);
}

.toolbar-check,
.saved-filter-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-8);
}

.toolbar-check {
  padding: var(--space-8);
  color: var(--text-secondary);
}

.bulk-bar {
  border-color: var(--state-info-border);
  background: var(--state-info-bg);
  color: var(--state-info-fg);
}

.data-shell {
  overflow: hidden;
}

.skeleton-wrap {
  display: grid;
  gap: var(--space-12);
  padding: var(--space-20);
}

.skeleton-row {
  height: 64px;
  border-radius: var(--radius-lg);
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.04), rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.04));
  background-size: 200% 100%;
  animation: audit-skeleton 1.5s linear infinite;
}

.audit-table.compact :deep(td),
.audit-table.compact :deep(th) {
  padding-top: var(--space-8);
  padding-bottom: var(--space-8);
}

.audit-row {
  cursor: pointer;
  transition: background-color 0.18s ease;
}

.audit-row.active {
  background: var(--accent-muted);
}

.audit-row--bruteforce td:first-child,
.audit-row--bruteforce .checkbox-col + td {
  box-shadow: inset 3px 0 0 var(--state-critical);
}

.checkbox-col {
  width: 42px;
}

.reason-cell {
  max-width: 240px;
}

.table-footer {
  display: flex;
  justify-content: space-between;
  gap: var(--space-12);
  flex-wrap: wrap;
  padding: var(--space-16) var(--space-20);
  border-top: 1px solid var(--border-subtle);
}

.mobile-audit-list {
  list-style: none;
  margin: 0;
  padding: var(--space-16);
  display: grid;
  gap: var(--space-12);
}

.mobile-audit-card {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  background: var(--surface-2);
  padding: var(--space-16);
  display: grid;
  gap: var(--space-12);
}

.mobile-audit-card--bruteforce {
  border-left: 3px solid var(--state-critical);
}

.mobile-audit-card.active {
  background: var(--surface-3);
}

.mobile-audit-card__top,
.mobile-audit-card__body,
.mobile-audit-card__meta {
  display: flex;
  gap: var(--space-8);
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
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

.drawer-stack,
.drawer-list {
  display: grid;
  gap: var(--space-8);
}

.drawer-list {
  margin: 0;
  padding-left: 18px;
  color: var(--text-secondary);
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

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

@keyframes audit-skeleton {
  from { background-position: 0% 0%; }
  to { background-position: 200% 0%; }
}

@media (max-width: 1099px) {
  .reason-cell {
    max-width: 180px;
  }
}

@media (max-width: 767px) {
  .toolbar-select,
  .toolbar-input,
  .toolbar-select--narrow {
    width: 100%;
  }
}
</style>
