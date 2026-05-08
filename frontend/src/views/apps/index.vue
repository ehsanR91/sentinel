<template>
  <div class="sc-page-shell sc-focus-ring apps-page" tabindex="0" @keydown="handleKeyboardNavigation">
    <PageHeader title="Apps" icon="mdi mdi-apps" :items="[{ text: 'Apps', active: true, icon: 'mdi mdi-apps' }]">
      <template #actions>
        <AppButton variant="secondary" size="md" icon="mdi mdi-refresh" :loading="loading" label="Refresh" @click="refreshApps" />
        <AppButton variant="secondary" size="md" icon="mdi mdi-text-box-search-outline" :label="showLogPanel ? 'Hide Activity' : 'Show Activity'" @click="showLogPanel = !showLogPanel" />
        <AppButton variant="primary" size="md" icon="mdi mdi-package-variant-closed" label="Open Catalog" @click="showDrawer = true; selectedAppName = filteredApps[0]?.name || ''" />
      </template>
    </PageHeader>

    <div v-if="updatesCount" class="sc-inline-error updates-banner">
      <div class="d-flex flex-wrap gap-2 align-items-center">
        <strong>{{ updatesCount }} apps have updates available</strong>
        <span>Filter to upgrades or queue them from the selection bar.</span>
      </div>
      <div class="d-flex flex-wrap gap-2">
        <AppButton variant="secondary" size="sm" icon="mdi mdi-arrow-up-circle-outline" label="Show updates" @click="installState = 'updates'" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-close" label="Dismiss" @click="updatesCount && (installState = 'all')" />
      </div>
    </div>

    <div v-if="showLogPanel && (opLogs.length || opRunning || queueItems.length)" class="sc-surface op-panel">
      <div class="op-panel__header">
        <div>
          <h5 class="mb-1">Operation Activity</h5>
          <div class="text-muted text-sm">
            {{ opRunning ? `${currentOpLabel} running` : 'No active operation' }}
            <span v-if="queueItems.length"> · {{ queueItems.length }} queued</span>
          </div>
        </div>
        <div class="d-flex flex-wrap gap-2">
          <StatusBadge :state="opRunning ? 'pending' : (opError ? 'error' : 'ok')" :label="opRunning ? 'Running' : (opError ? 'Failed' : 'Idle')" />
          <AppButton variant="ghost" size="sm" icon="mdi mdi-delete-outline" label="Clear" @click="opLogs = []" />
        </div>
      </div>
      <div v-if="opRunning" class="op-progress-bar"><span></span></div>
      <div ref="logWindow" class="op-log-window">
        <div v-for="(line, index) in opLogs" :key="`${index}-${line.ts}`" class="op-log-line" :class="`op-log-line--${line.type}`">
          <span class="op-log-time" :title="formatLogTimeTitle(line.ts)">{{ formatLogTime(line.ts) }}</span>
          <span>{{ line.text }}</span>
        </div>
        <div v-if="opError" class="op-log-line op-log-line--error">
          <span class="op-log-time">ERR</span>
          <span>{{ opError }}</span>
        </div>
      </div>
    </div>

    <div class="row g-3 stat-row">
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Catalog" :value="apps.length" sub="Managed packages" icon="mdi mdi-apps" clickable @click="installState = 'all'" />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Installed" :value="installedCount" sub="Present on this server" icon="mdi mdi-check-circle-outline" tone="ok" clickable @click="installState = 'installed'" />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Updates" :value="updatesCount" sub="Ready to upgrade" icon="mdi mdi-arrow-up-circle-outline" :tone="updatesCount ? 'warn' : 'default'" clickable @click="installState = 'updates'" />
      </div>
      <div class="col-6 col-xl-3 col-md-6">
        <StatCard label="Not Installed" :value="notInstalledCount" sub="Available to add" icon="mdi mdi-package-variant-plus" clickable @click="installState = 'not-installed'" />
      </div>
    </div>

    <FilterToolbar
      :search-query="searchQuery"
      search-placeholder="Search app, category, binary alias, install method… Try gh, node, python3"
      :active-chips="activeChips"
      :result-label="resultLabel"
      @update:search-query="updateSearch"
      @remove-chip="removeChip"
      @clear-all="clearFilters"
    >
      <template #controls>
        <select v-model="categoryFilter" class="form-select sc-focus-ring toolbar-select">
          <option value="">All categories</option>
          <option v-for="category in categories" :key="category" :value="category">{{ categoryLabel(category) }}</option>
        </select>
        <select v-model="installState" class="form-select sc-focus-ring toolbar-select">
          <option value="all">All states</option>
          <option value="installed">Installed</option>
          <option value="not-installed">Not installed</option>
          <option value="updates">Updates only</option>
        </select>
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
        <select v-model="viewMode" class="form-select sc-focus-ring toolbar-select toolbar-select--narrow" @change="persistViewMode">
          <option value="grid">Grid</option>
          <option value="list">List</option>
        </select>
        <select v-model="density" class="form-select sc-focus-ring toolbar-select toolbar-select--narrow" @change="persistDensity">
          <option value="comfortable">Comfortable</option>
          <option value="compact">Compact</option>
        </select>
        <AppButton :variant="multiSelect ? 'primary' : 'secondary'" size="md" icon="mdi mdi-checkbox-multiple-marked-outline" :label="multiSelect ? 'Selecting' : 'Select'" @click="toggleMultiSelect" />
      </template>
    </FilterToolbar>

    <ErrorState v-if="errorMessage" title="Apps refresh failed" :description="errorMessage">
      <template #actions>
        <AppButton variant="secondary" size="sm" icon="mdi mdi-refresh" label="Retry" @click="refreshApps" />
      </template>
    </ErrorState>

    <div v-if="multiSelect && selectedNames.length" class="sc-inline-error bulk-bar">
      <div class="d-flex flex-wrap gap-2 align-items-center">
        <strong>{{ selectedNames.length }} apps selected</strong>
        <span>{{ bulkSummary }}</span>
      </div>
      <div class="d-flex flex-wrap gap-2">
        <AppButton variant="secondary" size="sm" icon="mdi mdi-package-down" label="Install selected" :disabled="!selectedInstallable.length || anyOpRunning" @click="queueSelected('install')" />
        <AppButton variant="secondary" size="sm" icon="mdi mdi-arrow-up-circle-outline" label="Update selected" :disabled="!selectedUpdatable.length || anyOpRunning" @click="queueSelected('update')" />
        <AppButton variant="secondary" size="sm" icon="mdi mdi-delete-outline" label="Remove selected" :disabled="!selectedRemovable.length || anyOpRunning" @click="queueSelected('uninstall')" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-download" label="Export" @click="exportApps(true)" />
      </div>
    </div>

    <div class="sc-surface apps-shell">
      <div v-if="loading" class="skeleton-wrap" aria-busy="true">
        <div v-for="n in 8" :key="n" class="skeleton-row"></div>
      </div>

      <EmptyState
        v-else-if="!apps.length"
        icon="mdi mdi-package-variant-closed-remove"
        title="No apps available"
        description="The managed catalog could not be loaded from the server."
      >
        <template #actions>
          <AppButton variant="secondary" size="md" icon="mdi mdi-refresh" label="Refresh" @click="refreshApps" />
        </template>
      </EmptyState>

      <EmptyState
        v-else-if="!filteredApps.length"
        icon="mdi mdi-filter-off-outline"
        title="No apps match these filters"
        description="Clear the active filters or widen the search to see more of the catalog."
      >
        <template #actions>
          <AppButton variant="secondary" size="md" icon="mdi mdi-filter-remove-outline" label="Clear filters" @click="clearFilters" />
        </template>
      </EmptyState>

      <template v-else>
        <div v-if="viewMode === 'list'" class="d-none d-md-block">
          <table class="table apps-table mb-0" :class="{ compact: density === 'compact' }">
            <thead>
              <tr>
                <th v-if="multiSelect" class="checkbox-col"></th>
                <th>App</th>
                <th>Binary</th>
                <th>Versions</th>
                <th>Status</th>
                <th class="text-end">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(app, index) in filteredApps" :key="app.name" class="apps-row" :class="{ active: focusedIndex === index }" @click="openApp(app)">
                <td v-if="multiSelect" class="checkbox-col" @click.stop>
                  <input type="checkbox" :checked="selectedNames.includes(app.name)" @change="toggleSelected(app.name)" />
                </td>
                <td>
                  <div class="app-cell">
                    <div class="app-icon" :class="`app-icon--${app.category}`"><i :class="categoryIcon(app.category)"></i></div>
                    <div>
                      <div class="app-title">{{ app.label }}</div>
                      <div class="app-subtitle">{{ app.description }}</div>
                      <div class="app-meta-row">
                        <span class="sc-chip">{{ categoryLabel(app.category) }}</span>
                        <span class="sc-chip">{{ methodLabel(app.install_method) }}</span>
                      </div>
                    </div>
                  </div>
                </td>
                <td>
                  <span class="binary-pill">{{ app.binary || app.name }}</span>
                </td>
                <td>
                  <div class="version-stack">
                    <span class="text-muted text-sm">Installed: {{ app.version || '—' }}</span>
                    <span class="text-muted text-sm">Latest: {{ app.new_version || (app.installed ? app.version || 'Unknown' : '—') }}</span>
                  </div>
                </td>
                <td>
                  <div class="d-flex flex-wrap gap-2 align-items-center">
                    <StatusBadge :state="statusState(app)" :label="statusLabel(app)" :icon="statusIcon(app)" />
                    <StatusBadge v-if="app.update_avail" state="warn" label="Update" />
                  </div>
                  <div v-if="isBusy(app)" class="mini-progress"><span></span></div>
                </td>
                <td class="text-end" @click.stop>
                  <div class="action-row justify-content-end">
                    <AppButton
                      :variant="primaryAction(app).variant"
                      size="sm"
                      :icon="primaryAction(app).icon"
                      :label="primaryAction(app).label"
                      :disabled="primaryAction(app).disabled"
                      :loading="primaryAction(app).loading"
                      @click="runPrimaryAction(app)"
                    />
                    <details class="app-menu">
                      <summary class="sc-button sc-button--ghost sc-button--sm sc-button--icon-only">
                        <i class="mdi mdi-dots-horizontal"></i>
                      </summary>
                      <div class="toolbar-menu__body app-menu__body">
                        <button type="button" class="dropdown-item" @click="openApp(app)">View details</button>
                        <button type="button" class="dropdown-item" :disabled="!app.homepage" @click="openHomepage(app)">Open homepage</button>
                        <button type="button" class="dropdown-item" @click="copyBinary(app)">Copy binary</button>
                        <button type="button" class="dropdown-item" :disabled="!app.installed || anyOpRunning" @click="queueAction(app, 'uninstall')">Uninstall</button>
                      </div>
                    </details>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else class="apps-grid">
          <article v-for="(app, index) in filteredApps" :key="`grid-${app.name}`" class="app-card" :class="{ active: focusedIndex === index, compact: density === 'compact', busy: isBusy(app), update: app.update_avail, installed: app.installed }" @click="openApp(app)">
            <div class="app-card__top">
              <div class="app-card__identity">
                <div v-if="multiSelect" class="alert-checkbox" @click.stop>
                  <input type="checkbox" :checked="selectedNames.includes(app.name)" @change="toggleSelected(app.name)" />
                </div>
                <div class="app-icon" :class="`app-icon--${app.category}`"><i :class="categoryIcon(app.category)"></i></div>
                <div>
                  <div class="app-title">{{ app.label }}</div>
                  <div class="app-meta-row">
                    <span class="sc-chip">{{ categoryLabel(app.category) }}</span>
                    <span class="sc-chip">{{ methodLabel(app.install_method) }}</span>
                    <span class="binary-pill">{{ app.binary || app.name }}</span>
                  </div>
                </div>
              </div>
              <div class="action-row" @click.stop>
                <StatusBadge :state="statusState(app)" :label="statusLabel(app)" :icon="statusIcon(app)" />
                <details class="app-menu">
                  <summary class="sc-button sc-button--ghost sc-button--sm sc-button--icon-only">
                    <i class="mdi mdi-dots-horizontal"></i>
                  </summary>
                  <div class="toolbar-menu__body app-menu__body">
                    <button type="button" class="dropdown-item" @click="openApp(app)">View details</button>
                    <button type="button" class="dropdown-item" :disabled="!app.homepage" @click="openHomepage(app)">Open homepage</button>
                    <button type="button" class="dropdown-item" @click="copyBinary(app)">Copy binary</button>
                    <button type="button" class="dropdown-item" :disabled="!app.installed || anyOpRunning" @click="queueAction(app, 'uninstall')">Uninstall</button>
                  </div>
                </details>
              </div>
            </div>

            <p class="app-subtitle app-subtitle--clamped">{{ app.description }}</p>

            <div v-if="app.update_avail" class="update-banner-row">
              <i class="mdi mdi-arrow-up-bold-circle-outline"></i>
              Update available: {{ app.version || 'Unknown' }} → {{ app.new_version || 'Latest' }}
            </div>

            <div class="version-stack">
              <span class="text-muted text-sm">Installed: {{ app.version || '—' }}</span>
              <span class="text-muted text-sm">Latest: {{ app.new_version || (app.installed ? app.version || 'Unknown' : '—') }}</span>
            </div>

            <div v-if="isBusy(app)" class="mini-progress"><span></span></div>

            <div class="action-row" @click.stop>
              <AppButton
                :variant="primaryAction(app).variant"
                size="sm"
                :icon="primaryAction(app).icon"
                :label="primaryAction(app).label"
                :disabled="primaryAction(app).disabled"
                :loading="primaryAction(app).loading"
                @click="runPrimaryAction(app)"
              />
              <AppButton variant="secondary" size="sm" icon="mdi mdi-open-in-new" :disabled="!app.homepage" label="Docs" @click="openHomepage(app)" />
            </div>
          </article>
        </div>
      </template>
    </div>

    <DetailDrawer
      :model-value="showDrawer"
      :title="selectedApp ? selectedApp.label : 'App details'"
      :subtitle="selectedApp ? statusLabel(selectedApp) : ''"
      @update:model-value="showDrawer = $event"
      @navigate="navigateDrawer"
    >
      <template #nav>
        <AppButton variant="ghost" size="sm" icon="mdi mdi-chevron-left" aria-label="Previous app" icon-only @click="navigateDrawer(-1)" />
        <AppButton variant="ghost" size="sm" icon="mdi mdi-chevron-right" aria-label="Next app" icon-only @click="navigateDrawer(1)" />
      </template>
      <div v-if="selectedApp" class="drawer-grid">
        <section class="drawer-panel">
          <h6>App Overview</h6>
          <div class="drawer-stack">
            <div class="drawer-meta-row"><span>Status</span><StatusBadge :state="statusState(selectedApp)" :label="statusLabel(selectedApp)" :icon="statusIcon(selectedApp)" /></div>
            <div class="drawer-meta-row"><span>Category</span><span>{{ categoryLabel(selectedApp.category) }}</span></div>
            <div class="drawer-meta-row"><span>Binary</span><span class="binary-pill">{{ selectedApp.binary || selectedApp.name }}</span></div>
            <div class="drawer-meta-row"><span>Install method</span><span>{{ methodLabel(selectedApp.install_method) }}</span></div>
          </div>
        </section>

        <section class="drawer-panel">
          <h6>Versions</h6>
          <div class="drawer-stack">
            <div class="drawer-meta-row"><span>Installed</span><span>{{ selectedApp.version || '—' }}</span></div>
            <div class="drawer-meta-row"><span>Latest</span><span>{{ selectedApp.new_version || (selectedApp.installed ? selectedApp.version || 'Unknown' : '—') }}</span></div>
            <div class="drawer-meta-row"><span>Homepage</span><a v-if="selectedApp.homepage" :href="selectedApp.homepage" target="_blank" rel="noopener">{{ selectedApp.homepage }}</a><span v-else>—</span></div>
          </div>
        </section>

        <section class="drawer-panel drawer-panel--wide">
          <h6>Description</h6>
          <p class="drawer-copy">{{ selectedApp.description }}</p>
        </section>

        <section class="drawer-panel drawer-panel--wide" v-if="currentOpAppName === selectedApp.name && opLogs.length">
          <h6>Current Operation</h6>
          <div class="op-log-window op-log-window--drawer">
            <div v-for="(line, index) in opLogs" :key="`drawer-log-${index}`" class="op-log-line" :class="`op-log-line--${line.type}`">
              <span class="op-log-time" :title="formatLogTimeTitle(line.ts)">{{ formatLogTime(line.ts) }}</span>
              <span>{{ line.text }}</span>
            </div>
          </div>
        </section>

        <section class="drawer-panel drawer-panel--wide">
          <details>
            <summary>Raw JSON</summary>
            <pre>{{ JSON.stringify(selectedApp, null, 2) }}</pre>
          </details>
        </section>
      </div>
      <template #footer>
        <div class="d-flex flex-wrap gap-2">
          <AppButton
            v-if="selectedApp"
            :variant="primaryAction(selectedApp).variant"
            size="sm"
            :icon="primaryAction(selectedApp).icon"
            :label="primaryAction(selectedApp).label"
            :disabled="primaryAction(selectedApp).disabled"
            :loading="primaryAction(selectedApp).loading"
            @click="runPrimaryAction(selectedApp)"
          />
          <AppButton variant="secondary" size="sm" icon="mdi mdi-open-in-new" :disabled="!selectedApp?.homepage" label="Open docs" @click="selectedApp && openHomepage(selectedApp)" />
          <AppButton variant="destructive" size="sm" icon="mdi mdi-delete-outline" :disabled="!selectedApp?.installed || anyOpRunning" label="Uninstall" @click="selectedApp && queueAction(selectedApp, 'uninstall')" />
        </div>
      </template>
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
import api from '@/services/api'
import {
  formatTimestamp,
  getAppsViewPreference,
  getDensityPreference,
  loadSavedFilters,
  matchesQuery,
  saveSavedFilters,
  setAppsViewPreference,
  setDensityPreference
} from '@/utils/formatters'

const CATEGORY_LABELS = {
  cli: 'CLI Tools',
  runtime: 'Runtimes',
  web: 'Web',
  database: 'Database',
  build: 'Build Tools',
  devtool: 'Developer Tools',
  shell: 'Shell'
}

const CATEGORY_ICONS = {
  cli: 'mdi mdi-console-line',
  runtime: 'mdi mdi-code-braces',
  web: 'mdi mdi-web',
  database: 'mdi mdi-database',
  build: 'mdi mdi-hammer-wrench',
  devtool: 'mdi mdi-tools',
  shell: 'mdi mdi-bash'
}

const METHOD_LABELS = {
  pkg: 'Package',
  script: 'Script',
  binary: 'Binary',
  rustup: 'rustup'
}

export default {
  name: 'AppsPage',
  components: {
    PageHeader,
    StatCard,
    AppButton,
    StatusBadge,
    FilterToolbar,
    DetailDrawer,
    EmptyState,
    ErrorState
  },
  data () {
    return {
      loading: false,
      errorMessage: '',
      apps: [],
      searchQuery: '',
      debouncedSearch: '',
      searchTimer: null,
      categoryFilter: '',
      installState: 'all',
      savedFilters: loadSavedFilters('apps'),
      viewMode: getAppsViewPreference(),
      density: getDensityPreference(),
      multiSelect: false,
      selectedNames: [],
      showDrawer: false,
      selectedAppName: '',
      focusedIndex: 0,
      showLogPanel: true,
      opRunning: false,
      opLogs: [],
      opError: '',
      currentOpAppName: '',
      currentOpLabel: '',
      currentOpKind: '',
      opSeenCount: 0,
      pollTimer: null,
      queueItems: []
    }
  },
  computed: {
    categories () {
      return [...new Set(this.apps.map(app => app.category).filter(Boolean))].sort()
    },
    filteredApps () {
      return [...this.apps]
        .filter(app => !this.categoryFilter || app.category === this.categoryFilter)
        .filter(app => {
          if (this.installState === 'installed') return app.installed
          if (this.installState === 'not-installed') return !app.installed
          if (this.installState === 'updates') return app.update_avail
          return true
        })
        .filter(app => matchesQuery(app, this.debouncedSearch, {
          fields: ['name', 'label', 'description', 'category', 'install_method', 'homepage', 'binary'],
          operators: {
            binary: entry => entry.binary || entry.name,
            category: entry => entry.category,
            method: entry => entry.install_method,
            state: entry => this.statusLabel(entry).toLowerCase()
          }
        }))
        .sort((left, right) => {
          if (left.update_avail !== right.update_avail) return Number(right.update_avail) - Number(left.update_avail)
          if (left.installed !== right.installed) return Number(right.installed) - Number(left.installed)
          return left.label.localeCompare(right.label)
        })
    },
    selectedApp () {
      return this.filteredApps.find(app => app.name === this.selectedAppName) || this.apps.find(app => app.name === this.selectedAppName) || null
    },
    installedCount () {
      return this.apps.filter(app => app.installed).length
    },
    updatesCount () {
      return this.apps.filter(app => app.update_avail).length
    },
    notInstalledCount () {
      return this.apps.filter(app => !app.installed).length
    },
    resultLabel () {
      return this.filteredApps.length === this.apps.length
        ? `${this.apps.length} apps`
        : `${this.filteredApps.length} of ${this.apps.length} apps`
    },
    activeChips () {
      const chips = []
      if (this.categoryFilter) chips.push({ key: 'category', label: `Category: ${this.categoryLabel(this.categoryFilter)}` })
      if (this.installState !== 'all') chips.push({ key: 'state', label: `State: ${this.installState}` })
      if (this.debouncedSearch) chips.push({ key: 'search', label: `Search: ${this.debouncedSearch}` })
      return chips
    },
    anyOpRunning () {
      return this.opRunning || this.queueItems.length > 0
    },
    selectedApps () {
      return this.filteredApps.filter(app => this.selectedNames.includes(app.name))
    },
    selectedInstallable () {
      return this.selectedApps.filter(app => !app.installed)
    },
    selectedUpdatable () {
      return this.selectedApps.filter(app => app.installed && app.update_avail)
    },
    selectedRemovable () {
      return this.selectedApps.filter(app => app.installed)
    },
    bulkSummary () {
      return `${this.selectedInstallable.length} installable, ${this.selectedUpdatable.length} updatable, ${this.selectedRemovable.length} removable.`
    }
  },
  watch: {
    searchQuery () {
      clearTimeout(this.searchTimer)
      this.searchTimer = setTimeout(() => {
        this.debouncedSearch = this.searchQuery
      }, 150)
    },
    filteredApps () {
      if (this.focusedIndex >= this.filteredApps.length) {
        this.focusedIndex = Math.max(0, this.filteredApps.length - 1)
      }
    }
  },
  async mounted () {
    await this.loadApps()
    try {
      const { data } = await api.getAppOpLogs()
      if (data.running) {
        this.opRunning = true
        this.currentOpAppName = data.app || ''
        this.currentOpLabel = this.apps.find(app => app.name === data.app)?.label || data.app || ''
        this.currentOpKind = data.kind || ''
        this.showLogPanel = true
        this.startPolling()
      }
    } catch {
      // Ignore warm-start polling failures.
    }
  },
  beforeUnmount () {
    clearTimeout(this.searchTimer)
    this.stopPolling()
  },
  methods: {
    async loadApps () {
      this.loading = true
      this.errorMessage = ''
      try {
        const { data } = await api.getApps()
        this.apps = Array.isArray(data) ? data : []
      } catch (error) {
        this.errorMessage = error.response?.data?.error || error.message || 'Unable to load apps.'
      } finally {
        this.loading = false
      }
    },
    async refreshApps () {
      await this.loadApps()
    },
    updateSearch (value) {
      this.searchQuery = value
    },
    clearFilters () {
      this.searchQuery = ''
      this.debouncedSearch = ''
      this.categoryFilter = ''
      this.installState = 'all'
    },
    removeChip (key) {
      if (key === 'category') this.categoryFilter = ''
      if (key === 'state') this.installState = 'all'
      if (key === 'search') {
        this.searchQuery = ''
        this.debouncedSearch = ''
      }
    },
    persistViewMode () {
      setAppsViewPreference(this.viewMode)
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
            categoryFilter: this.categoryFilter,
            installState: this.installState,
            viewMode: this.viewMode,
            density: this.density
          }
        },
        ...this.savedFilters
      ].slice(0, 10)
      saveSavedFilters('apps', this.savedFilters)
    },
    applySavedFilter (filter) {
      Object.assign(this, filter.state)
      this.debouncedSearch = filter.state.searchQuery
      this.persistViewMode()
      this.persistDensity()
    },
    deleteSavedFilter (id) {
      this.savedFilters = this.savedFilters.filter(filter => filter.id !== id)
      saveSavedFilters('apps', this.savedFilters)
    },
    categoryLabel (category) {
      return CATEGORY_LABELS[category] || category || 'Unknown'
    },
    categoryIcon (category) {
      return CATEGORY_ICONS[category] || 'mdi mdi-package-variant-closed'
    },
    methodLabel (method) {
      return METHOD_LABELS[method] || method || 'Unknown'
    },
    statusLabel (app) {
      if (['installing', 'updating', 'uninstalling'].includes(app.status)) {
        return app.status.charAt(0).toUpperCase() + app.status.slice(1)
      }
      if (app.status === 'failed') return 'Failed'
      if (app.installed && app.update_avail) return 'Update Available'
      if (app.installed) return 'Installed'
      return 'Not Installed'
    },
    statusState (app) {
      if (['installing', 'updating', 'uninstalling'].includes(app.status)) return 'pending'
      if (app.status === 'failed') return 'error'
      if (app.installed && app.update_avail) return 'warn'
      if (app.installed) return 'ok'
      return 'muted'
    },
    statusIcon (app) {
      if (app.status === 'installing') return 'mdi mdi-loading mdi-spin'
      if (app.status === 'updating') return 'mdi mdi-loading mdi-spin'
      if (app.status === 'uninstalling') return 'mdi mdi-loading mdi-spin'
      if (app.status === 'failed') return 'mdi mdi-alert-circle-outline'
      if (app.installed && app.update_avail) return 'mdi mdi-arrow-up-circle-outline'
      if (app.installed) return 'mdi mdi-check-circle-outline'
      return 'mdi mdi-package-variant-closed'
    },
    isBusy (app) {
      return ['installing', 'updating', 'uninstalling'].includes(app.status) || this.currentOpAppName === app.name && this.opRunning
    },
    primaryAction (app) {
      if (app.status === 'installing') return { label: 'Installing', icon: 'mdi mdi-loading mdi-spin', variant: 'secondary', disabled: true, loading: true }
      if (app.status === 'updating') return { label: 'Updating', icon: 'mdi mdi-loading mdi-spin', variant: 'secondary', disabled: true, loading: true }
      if (app.status === 'uninstalling') return { label: 'Removing', icon: 'mdi mdi-loading mdi-spin', variant: 'secondary', disabled: true, loading: true }
      if (!app.installed) return { label: 'Install', icon: 'mdi mdi-package-down', variant: 'primary', disabled: this.anyOpRunning, loading: false }
      if (app.update_avail) return { label: 'Update', icon: 'mdi mdi-arrow-up-circle-outline', variant: 'secondary', disabled: this.anyOpRunning, loading: false }
      return { label: 'Installed', icon: 'mdi mdi-check-circle-outline', variant: 'ghost', disabled: true, loading: false }
    },
    runPrimaryAction (app) {
      if (!app.installed) return this.queueAction(app, 'install')
      if (app.update_avail) return this.queueAction(app, 'update')
    },
    toggleMultiSelect () {
      this.multiSelect = !this.multiSelect
      if (!this.multiSelect) this.selectedNames = []
    },
    toggleSelected (name) {
      if (this.selectedNames.includes(name)) {
        this.selectedNames = this.selectedNames.filter(item => item !== name)
      } else {
        this.selectedNames = [...this.selectedNames, name]
      }
    },
    openApp (app) {
      this.selectedAppName = app.name
      this.showDrawer = true
    },
    formatLogTime (value) {
      return formatTimestamp(value).primary
    },
    formatLogTimeTitle (value) {
      return formatTimestamp(value).title
    },
    navigateDrawer (step) {
      if (!this.selectedApp) return
      const index = this.filteredApps.findIndex(app => app.name === this.selectedApp.name)
      const next = this.filteredApps[index + step]
      if (next) this.openApp(next)
    },
    async queueAction (app, kind) {
      const labels = { install: 'Install', update: 'Update', uninstall: 'Uninstall' }
      const confirmed = await this.$swal({
        icon: kind === 'uninstall' ? 'warning' : 'question',
        title: `${labels[kind]} ${app.label}?`,
        text: kind === 'uninstall' ? 'This removes the managed package from the server.' : app.description,
        showCancelButton: true,
        confirmButtonText: labels[kind]
      })
      if (!confirmed.isConfirmed) return
      this.queueItems = [...this.queueItems, { name: app.name, kind }]
      if (!this.opRunning) {
        this.processQueue()
      }
    },
    async queueSelected (kind) {
      const map = {
        install: this.selectedInstallable,
        update: this.selectedUpdatable,
        uninstall: this.selectedRemovable
      }
      const items = map[kind]
      if (!items.length) return
      const confirmed = await this.$swal({
        icon: kind === 'uninstall' ? 'warning' : 'question',
        title: `${kind.charAt(0).toUpperCase() + kind.slice(1)} ${items.length} apps?`,
        text: 'Operations will run one at a time in the queue.',
        showCancelButton: true,
        confirmButtonText: 'Queue operations'
      })
      if (!confirmed.isConfirmed) return
      this.queueItems = [...this.queueItems, ...items.map(app => ({ name: app.name, kind }))]
      if (!this.opRunning) {
        this.processQueue()
      }
    },
    async processQueue () {
      if (this.opRunning || !this.queueItems.length) return
      const next = this.queueItems[0]
      const app = this.apps.find(item => item.name === next.name)
      if (!app) {
        this.queueItems = this.queueItems.slice(1)
        return this.processQueue()
      }
      this.opRunning = true
      this.currentOpAppName = app.name
      this.currentOpLabel = app.label
      this.currentOpKind = next.kind
      this.opError = ''
      this.showLogPanel = true
      this.opLogs = this.queueItems.length === 1 ? [] : this.opLogs
      this.addLog(`Queued ${next.kind} for ${app.label}`, 'info')
      this.apps = this.apps.map(item => item.name === app.name ? { ...item, status: `${next.kind}ing` } : item)
      try {
        if (next.kind === 'install') await api.installApp(app.name)
        if (next.kind === 'update') await api.updateApp(app.name)
        if (next.kind === 'uninstall') await api.uninstallApp(app.name)
        this.addLog(`Started ${next.kind} for ${app.label}`, 'info')
        this.startPolling()
      } catch (error) {
        this.opError = error.response?.data?.error || error.message || 'Operation failed.'
        this.addLog(this.opError, 'error')
        this.opRunning = false
        this.queueItems = this.queueItems.slice(1)
        await this.loadApps()
        this.processQueue()
      }
    },
    startPolling () {
      this.stopPolling()
      this.pollTimer = window.setInterval(() => this.pollLogs(), 1500)
    },
    stopPolling () {
      if (this.pollTimer) {
        clearInterval(this.pollTimer)
        this.pollTimer = null
      }
    },
    async pollLogs () {
      try {
        const { data } = await api.getAppOpLogs()
        if (Array.isArray(data.logs) && data.logs.length > this.opSeenCount) {
          const incoming = data.logs.slice(this.opSeenCount)
          this.opSeenCount = data.logs.length
          incoming.forEach(line => {
            const lower = line.toLowerCase()
            const type = lower.includes('error') || lower.includes('fail') ? 'error' : lower.includes('warn') ? 'warn' : lower.includes('success') || lower.includes('installed') || lower.includes('complete') ? 'success' : 'info'
            this.addLog(line, type)
          })
        }
        if (data.done) {
          this.stopPolling()
          if (data.error) {
            this.opError = data.error
            this.addLog(`Operation failed: ${data.error}`, 'error')
          } else {
            this.addLog(`${this.currentOpLabel} ${this.currentOpKind} complete`, 'success')
          }
          this.opRunning = false
          this.queueItems = this.queueItems.slice(1)
          this.opSeenCount = 0
          await this.loadApps()
          this.processQueue()
        }
      } catch {
        // Ignore polling failures until the next tick.
      }
    },
    addLog (text, type = 'info') {
      this.opLogs.push({ ts: Date.now(), text, type })
      this.$nextTick(() => {
        const element = this.$refs.logWindow
        if (element) element.scrollTop = element.scrollHeight
      })
    },
    openHomepage (app) {
      if (app.homepage) window.open(app.homepage, '_blank', 'noopener')
    },
    async copyBinary (app) {
      try {
        await navigator.clipboard.writeText(app.binary || app.name)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Binary copied', showConfirmButton: false, timer: 2000 })
      } catch {
        this.errorMessage = 'Unable to copy binary to clipboard.'
      }
    },
    exportApps (selectedOnly = false) {
      const rows = selectedOnly ? this.selectedApps : this.filteredApps
      const blob = new Blob([JSON.stringify(rows, null, 2)], { type: 'application/json;charset=utf-8' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `apps-${Date.now()}.json`
      link.click()
    },
    handleKeyboardNavigation (event) {
      if (!this.filteredApps.length) return
      if (['ArrowDown', 'j', 'J'].includes(event.key)) {
        event.preventDefault()
        this.focusedIndex = Math.min(this.focusedIndex + 1, this.filteredApps.length - 1)
        return
      }
      if (['ArrowUp', 'k', 'K'].includes(event.key)) {
        event.preventDefault()
        this.focusedIndex = Math.max(this.focusedIndex - 1, 0)
        return
      }
      if (event.key === 'Enter') {
        event.preventDefault()
        this.openApp(this.filteredApps[this.focusedIndex])
        return
      }
      if (event.key === ' ' && this.multiSelect) {
        event.preventDefault()
        this.toggleSelected(this.filteredApps[this.focusedIndex].name)
      }
      if (event.key === 'Escape') {
        this.showDrawer = false
      }
    }
  }
}
</script>

<style scoped>
.apps-page {
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

.toolbar-select {
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

.op-panel {
  display: grid;
  gap: var(--space-12);
}

.op-panel__header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-12);
  flex-wrap: wrap;
  align-items: center;
}

.op-progress-bar,
.mini-progress {
  height: 6px;
  border-radius: 999px;
  background: var(--surface-3);
  overflow: hidden;
}

.op-progress-bar span,
.mini-progress span {
  display: block;
  height: 100%;
  width: 30%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--accent), color-mix(in srgb, var(--accent) 40%, white));
  animation: progress-slide 1.2s ease-in-out infinite;
}

.op-log-window {
  max-height: 260px;
  overflow: auto;
  padding: var(--space-12);
  border-radius: var(--radius-lg);
  background: var(--surface-2);
  display: grid;
  gap: var(--space-6);
}

.op-log-window--drawer {
  max-height: 320px;
}

.op-log-line {
  display: grid;
  grid-template-columns: 86px 1fr;
  gap: var(--space-10);
  font-family: var(--font-mono, 'SFMono-Regular', Consolas, monospace);
  font-size: var(--font-size-12);
}

.op-log-line--error {
  color: var(--state-error-fg);
}

.op-log-line--warn {
  color: var(--state-warn-fg);
}

.op-log-line--success {
  color: var(--state-ok-fg);
}

.op-log-time {
  color: var(--text-tertiary);
}

.apps-shell {
  overflow: hidden;
}

.skeleton-wrap {
  display: grid;
  gap: var(--space-12);
  padding: var(--space-20);
}

.skeleton-row {
  height: 120px;
  border-radius: var(--radius-lg);
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.04), rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.04));
  background-size: 200% 100%;
  animation: app-skeleton 1.5s linear infinite;
}

.apps-table.compact :deep(td),
.apps-table.compact :deep(th) {
  padding-top: var(--space-8);
  padding-bottom: var(--space-8);
}

.apps-row {
  cursor: pointer;
}

.apps-row.active,
.apps-row:hover {
  background: var(--surface-2);
}

.checkbox-col {
  width: 42px;
}

.app-cell {
  display: grid;
  grid-template-columns: 40px 1fr;
  gap: var(--space-12);
  align-items: start;
}

.app-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 1.1rem;
  color: var(--text-primary);
  background: var(--surface-3);
}

.app-icon--cli { background: color-mix(in srgb, var(--accent) 15%, transparent); color: var(--accent); }
.app-icon--runtime { background: color-mix(in srgb, var(--state-ok) 16%, transparent); color: var(--state-ok); }
.app-icon--web { background: color-mix(in srgb, var(--state-info) 16%, transparent); color: var(--state-info); }
.app-icon--database { background: color-mix(in srgb, var(--state-warn) 16%, transparent); color: var(--state-warn); }
.app-icon--build { background: color-mix(in srgb, var(--state-error) 16%, transparent); color: var(--state-error); }
.app-icon--devtool { background: color-mix(in srgb, var(--state-pending) 16%, transparent); color: var(--state-pending); }
.app-icon--shell { background: color-mix(in srgb, var(--state-muted) 22%, transparent); color: var(--text-secondary); }

.app-title {
  font-size: var(--font-size-15);
  font-weight: 600;
  color: var(--text-primary);
}

.app-subtitle {
  font-size: var(--font-size-13);
  color: var(--text-secondary);
}

.app-subtitle--clamped {
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.app-meta-row,
.action-row,
.version-stack {
  display: flex;
  gap: var(--space-8);
  flex-wrap: wrap;
  align-items: center;
}

.version-stack {
  flex-direction: column;
  align-items: flex-start;
}

.binary-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
  background: var(--surface-3);
  border: 1px solid var(--border-subtle);
  color: var(--text-primary);
  font-family: var(--font-mono, 'SFMono-Regular', Consolas, monospace);
  font-size: var(--font-size-12);
}

.apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: var(--space-16);
  padding: var(--space-20);
}

.app-card {
  display: grid;
  gap: var(--space-12);
  padding: var(--space-16);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--surface-2);
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease, background-color 0.18s ease;
}

.app-card:hover,
.app-card.active {
  transform: translateY(-1px);
  border-color: color-mix(in srgb, var(--accent) 40%, var(--border-default));
  background: var(--surface-3);
}

.app-card.installed {
  border-color: color-mix(in srgb, var(--state-ok) 20%, var(--border-default));
}

.app-card.update {
  border-color: color-mix(in srgb, var(--state-warn) 35%, var(--border-default));
}

.app-card.busy {
  border-color: color-mix(in srgb, var(--accent) 40%, var(--border-default));
}

.app-card.compact {
  padding: var(--space-12);
}

.app-card__top,
.app-card__identity {
  display: flex;
  gap: var(--space-12);
  align-items: flex-start;
  justify-content: space-between;
}

.update-banner-row {
  display: inline-flex;
  gap: var(--space-8);
  align-items: center;
  padding: 0.45rem 0.65rem;
  border-radius: var(--radius-md);
  background: var(--state-warn-bg);
  border: 1px solid var(--state-warn-border);
  color: var(--state-warn-fg);
  font-size: var(--font-size-12);
}

.app-menu {
  position: relative;
}

.app-menu__body {
  right: 0;
  left: auto;
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

.drawer-stack {
  display: grid;
  gap: var(--space-8);
}

.drawer-copy,
pre {
  margin: 0;
}

pre {
  white-space: pre-wrap;
  word-break: break-word;
}

@keyframes progress-slide {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(320%); }
}

@keyframes app-skeleton {
  from { background-position: 0% 0%; }
  to { background-position: 200% 0%; }
}

@media (max-width: 767px) {
  .toolbar-select,
  .toolbar-select--narrow {
    width: 100%;
  }

  .apps-grid {
    grid-template-columns: 1fr;
  }

  .app-card__top,
  .app-card__identity {
    flex-direction: column;
  }
}
</style>
