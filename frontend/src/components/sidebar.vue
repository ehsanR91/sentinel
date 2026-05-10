<template>
  <aside
    id="sidebar-menu"
    ref="sidebarRoot"
    :class="asideClasses"
    :data-density="sidebarDensity"
    :data-position="sidebarPosition"
    aria-label="Primary navigation"
    @touchstart.passive="onTouchStart"
    @touchmove.passive="onTouchMove"
    @touchend.passive="onTouchEnd"
    :style="sidebarStyle"
  >
    <div class="sidebar-header">
      <div ref="brandRow" class="sidebar-brand-row">
        <router-link
          to="/dashboard"
          class="sidebar-brand"
          @click="handleSidebarRouteClick('/dashboard')"
        >
          <div class="sidebar-brand__mark" :class="{ pulse: reconnectPulse && !reducedMotion }">
            <i class="mdi mdi-shield-half-full" aria-hidden="true"></i>
          </div>
          <div v-if="!effectiveCollapsed" class="sidebar-brand__copy">
            <span class="sidebar-brand__name">SentinelCore</span>
            <span class="sidebar-brand__sub">Ops console</span>
          </div>
        </router-link>

        <div class="sidebar-brand-actions">
          <button
            type="button"
            class="sidebar-icon-btn"
            aria-label="Open brand menu"
            :aria-expanded="brandMenuOpen ? 'true' : 'false'"
            @click.stop="toggleBrandMenu"
          >
            <i class="mdi mdi-chevron-down" aria-hidden="true"></i>
          </button>
          <Tooltip :label="effectiveCollapsed ? 'Expand sidebar' : 'Collapse sidebar'" :shortcut="effectiveCollapsed ? ']' : '['" as-child>
            <button
              type="button"
              class="collapse-btn-edge d-none d-lg-inline-flex"
              :aria-label="effectiveCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
              @click="toggleCollapse"
            >
              <i class="mdi" :class="effectiveCollapsed ? 'mdi-chevron-right' : 'mdi-chevron-left'" aria-hidden="true"></i>
            </button>
          </Tooltip>
        </div>

        <Teleport to="body">
          <div v-if="brandMenuOpen" class="sidebar-popover brand-popover" role="menu" :style="brandPopoverStyle">
            <div class="sidebar-popover__title">Workspace</div>
            <button type="button" class="sidebar-popover__item" @click="openCommandPalette">
              <i class="mdi mdi-magnify"></i>
              <span>Open command palette</span>
              <kbd>Ctrl K</kbd>
            </button>
            <button type="button" class="sidebar-popover__item" @click="goToDashboard">
              <i class="mdi mdi-view-dashboard-outline"></i>
              <span>Go to dashboard</span>
            </button>
            <div class="sidebar-popover__group">
              <label class="sidebar-pref-row">
                <span>Density</span>
                <ScSelect :model-value="sidebarDensity" :options="[{value:'comfortable',label:'Comfortable'},{value:'compact',label:'Compact'}]" size="sm" style="width:130px" @change="setSidebarDensity" />
              </label>
              <label class="sidebar-pref-row">
                <span>Position</span>
                <ScSelect :model-value="sidebarPosition" :options="[{value:'left',label:'Left'},{value:'right',label:'Right'}]" size="sm" style="width:130px" @change="setSidebarPosition" />
              </label>
            </div>
            <div class="sidebar-popover__group sidebar-popover__group--stacked">
              <div class="sidebar-popover__label">Visible sections</div>
              <label v-for="section in sectionDefinitions" :key="`pref-${section.id}`" class="sidebar-checkbox-row">
                <input
                  type="checkbox"
                  :checked="!hiddenSections.includes(section.id)"
                  @change="toggleSectionVisibility(section.id)"
                >
                <span>{{ section.label }}</span>
              </label>
              <button type="button" class="sidebar-reset-link" @click="resetSectionVisibility">Show all</button>
            </div>
            <div class="sidebar-popover__footer">
              <span>v{{ appVersion }}</span>
              <button type="button" class="sidebar-popover__link" @click="logout">Sign out</button>
            </div>
          </div>
        </Teleport>
      </div>

      <div v-show="!effectiveCollapsed" class="sidebar-search-wrap">
        <label for="sidebar-inline-search" class="visually-hidden">Search navigation</label>
        <div class="sidebar-search-field">
          <i class="mdi mdi-magnify" aria-hidden="true"></i>
          <input
            id="sidebar-inline-search"
            ref="searchInput"
            v-model.trim="searchQuery"
            type="search"
            autocomplete="off"
            spellcheck="false"
            placeholder="Find pages, tools, or settings"
            @keydown.esc.prevent="clearSearch"
          >
          <button v-if="searchQuery" type="button" class="sidebar-search-clear" aria-label="Clear sidebar search" @click="clearSearch">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
      </div>
    </div>

    <div ref="scrollContainer" class="sidebar-scroll" @scroll.passive="persistScrollPosition">
      <section v-if="pinnedItems.length" class="sidebar-section sidebar-section--pinned">
        <div class="sidebar-section-heading">
          <span id="sidebar-section-pinned">Pinned</span>
          <span class="sidebar-section-chip">{{ pinnedItems.length }}/6</span>
        </div>
        <nav aria-labelledby="sidebar-section-pinned">
          <ul class="sidebar-list">
            <li
              v-for="item in pinnedItems"
              :key="`pin-${item.id}`"
              class="sidebar-item"
              :class="{ 'is-orphan': item.orphan }"
              draggable="true"
              @dragstart="startPinDrag(item)"
              @dragover.prevent
              @drop="dropPinnedItem(item)"
            >
              <button
                v-if="item.orphan"
                type="button"
                class="sidebar-link sidebar-link--orphan"
                @click="unpinItem(item.id)"
              >
                <span class="sidebar-link__accent" aria-hidden="true"></span>
                <i class="mdi mdi-link-off"></i>
                <span class="sidebar-link__label">
                  <span class="sidebar-link__text sidebar-link__text--orphan">{{ item.label }}</span>
                  <span class="visually-hidden">Page no longer exists. Activate to unpin.</span>
                </span>
                <span class="sidebar-link__meta">
                  <span class="sidebar-pill sidebar-pill--ghost">Unpin</span>
                </span>
              </button>
              <router-link
                v-else
                :to="item.link"
                class="sidebar-link"
                :class="linkClasses(item)"
                :data-route="item.link"
                :aria-current="isRouteActive(item.link) ? 'page' : undefined"
                :aria-describedby="effectiveCollapsed && tooltip.item?.id === item.id ? 'sidebar-tooltip' : undefined"
                @click="handleSidebarRouteClick(item.link)"
                @contextmenu.prevent="openContextMenu($event, item)"
                @mouseenter="showTooltip($event, item)"
                @mouseleave="hideTooltip"
                @focus="showTooltip($event, item)"
                @blur="hideTooltip"
              >
                <span class="sidebar-link__accent" aria-hidden="true"></span>
                <i :class="resolvedItemIcon(item)" aria-hidden="true"></i>
                <span class="sidebar-link__label">
                  <span class="sidebar-link__text" v-html="highlightLabel(item.label)"></span>
                  <span class="visually-hidden">{{ accessibilityLabel(item) }}</span>
                </span>
                <span class="sidebar-link__meta">
                  <span v-if="itemStatusTone(item) && !badgeForItem(item)" class="sidebar-status-dot" :class="`is-${itemStatusTone(item)}`" aria-hidden="true"></span>
                  <span
                    v-if="badgeForItem(item)"
                    class="sidebar-counter-badge"
                    :class="`is-${badgeForItem(item).tone}`"
                    :aria-label="badgeForItem(item).ariaLabel"
                  >{{ badgeForItem(item).text }}</span>
                  <span v-if="tagForItem(item)" class="sidebar-pill sidebar-pill--tag">{{ tagForItem(item) }}</span>
                </span>
              </router-link>
            </li>
          </ul>
        </nav>
      </section>

      <section v-for="section in visibleSections" :key="section.id" class="sidebar-section">
        <button
          :id="`sidebar-section-${section.id}`"
          type="button"
          class="sidebar-section-toggle"
          :aria-expanded="isSectionExpanded(section) ? 'true' : 'false'"
          @click="toggleSection(section.id)"
        >
          <span class="sidebar-section-heading__text">{{ section.label }}</span>
          <span v-if="!isSectionExpanded(section) && sectionSummary(section)" class="sidebar-section-chip">{{ sectionSummary(section) }}</span>
          <i class="mdi mdi-chevron-down" :class="{ rotated: isSectionExpanded(section) }" aria-hidden="true"></i>
        </button>

        <nav :aria-labelledby="`sidebar-section-${section.id}`">
          <ul v-show="isSectionExpanded(section)" class="sidebar-list">
            <li
              v-for="item in section.items"
              :key="item.id"
              class="sidebar-item"
              :class="{ 'is-dimmed': shouldDimItem(item), 'is-active-branch': isItemActive(item) }"
            >
              <template v-if="item.children?.length">
                <button
                  type="button"
                  class="sidebar-link sidebar-link--button"
                  :class="linkClasses(item)"
                  :aria-expanded="isParentExpanded(item) ? 'true' : 'false'"
                  @click="toggleParent(item.id)"
                >
                  <span class="sidebar-link__accent" aria-hidden="true"></span>
                  <i :class="resolvedItemIcon(item)" aria-hidden="true"></i>
                  <span class="sidebar-link__label">
                    <span class="sidebar-link__text" v-html="highlightLabel(item.label)"></span>
                    <span class="visually-hidden">{{ accessibilityLabel(item) }}</span>
                  </span>
                  <span class="sidebar-link__meta">
                    <span v-if="badgeForItem(item)" class="sidebar-counter-badge" :class="`is-${badgeForItem(item).tone}`">{{ badgeForItem(item).text }}</span>
                    <i class="mdi mdi-chevron-down sidebar-chevron" :class="{ rotated: isParentExpanded(item) }" aria-hidden="true"></i>
                  </span>
                </button>

                <ul v-show="isParentExpanded(item)" class="sidebar-subnav">
                  <li v-for="child in item.children" :key="child.id" class="sidebar-subnav__item" :class="{ 'is-dimmed': shouldDimItem(child) }">
                    <router-link
                      :to="child.link"
                      class="sidebar-link sidebar-link--child"
                      :class="linkClasses(child)"
                      :data-route="child.link"
                      :aria-current="isRouteActive(child.link) ? 'page' : undefined"
                      :aria-describedby="effectiveCollapsed && tooltip.item?.id === child.id ? 'sidebar-tooltip' : undefined"
                      @click="handleSidebarRouteClick(child.link)"
                      @contextmenu.prevent="openContextMenu($event, child)"
                      @mouseenter="showTooltip($event, child)"
                      @mouseleave="hideTooltip"
                      @focus="showTooltip($event, child)"
                      @blur="hideTooltip"
                    >
                      <span class="sidebar-link__accent" aria-hidden="true"></span>
                      <span class="sidebar-subnav__connector" aria-hidden="true"></span>
                      <i :class="resolvedItemIcon(child)" aria-hidden="true"></i>
                      <span class="sidebar-link__label">
                        <span class="sidebar-link__text" v-html="highlightLabel(child.label)"></span>
                        <span class="visually-hidden">{{ accessibilityLabel(child) }}</span>
                      </span>
                      <span class="sidebar-link__meta">
                        <span v-if="tagForItem(child)" class="sidebar-pill sidebar-pill--tag">{{ tagForItem(child) }}</span>
                      </span>
                    </router-link>
                  </li>
                </ul>
              </template>

              <router-link
                v-else
                :to="item.link"
                class="sidebar-link"
                :class="linkClasses(item)"
                :data-route="item.link"
                :aria-current="isRouteActive(item.link) ? 'page' : undefined"
                :aria-describedby="effectiveCollapsed && tooltip.item?.id === item.id ? 'sidebar-tooltip' : undefined"
                @click="handleSidebarRouteClick(item.link)"
                @contextmenu.prevent="openContextMenu($event, item)"
                @mouseenter="showTooltip($event, item)"
                @mouseleave="hideTooltip"
                @focus="showTooltip($event, item)"
                @blur="hideTooltip"
              >
                <span class="sidebar-link__accent" aria-hidden="true"></span>
                <i :class="resolvedItemIcon(item)" aria-hidden="true"></i>
                <span class="sidebar-link__label">
                  <span class="sidebar-link__text" v-html="highlightLabel(item.label)"></span>
                  <span class="visually-hidden">{{ accessibilityLabel(item) }}</span>
                </span>
                <span class="sidebar-link__meta">
                  <span v-if="itemStatusTone(item) && !badgeForItem(item)" class="sidebar-status-dot" :class="`is-${itemStatusTone(item)}`" aria-hidden="true"></span>
                  <span
                    v-if="badgeForItem(item)"
                    class="sidebar-counter-badge"
                    :class="`is-${badgeForItem(item).tone}`"
                    :aria-label="badgeForItem(item).ariaLabel"
                  >{{ badgeForItem(item).text }}</span>
                  <span v-if="tagForItem(item)" class="sidebar-pill sidebar-pill--tag">{{ tagForItem(item) }}</span>
                </span>
              </router-link>
            </li>
          </ul>
        </nav>
      </section>
    </div>

    <div class="sidebar-footer">
      <div
        class="server-footer-card"
        :class="`is-${serverTone}`"
        role="button"
        tabindex="0"
        :aria-label="`Open server details for ${serverName}`"
        @click="serverDrawerOpen = true"
        @keydown.enter.prevent="serverDrawerOpen = true"
        @keydown.space.prevent="serverDrawerOpen = true"
        @mouseenter="showTooltip($event, { id: 'server-footer', label: serverName, provider: serverSubline })"
        @mouseleave="hideTooltip"
      >
        <div class="server-footer-card__top">
          <div class="server-footer-card__identity">
            <span class="sidebar-status-dot" :class="`is-${serverTone}`" aria-hidden="true"></span>
            <div v-if="!effectiveCollapsed" class="server-footer-card__title-wrap">
              <span class="server-footer-card__title">{{ serverName }}</span>
              <span class="server-footer-card__sub">{{ serverSubline }}</span>
            </div>
          </div>
          <button type="button" class="sidebar-icon-btn" aria-label="Server actions" @click.stop="toggleFooterMenu">
            <i class="mdi mdi-dots-horizontal"></i>
          </button>
        </div>

        <template v-if="!effectiveCollapsed">
          <div class="server-footer-card__metrics" :class="{ offline: !wsConnected }">
            <span>CPU {{ formatPercent(snap.cpu_pct) }}</span>
            <span>MEM {{ formatPercent(snap.ram_pct) }}</span>
            <span>{{ wsConnected ? `↑ ${formatRate(snap.net_tx_rate)}` : 'Reconnecting…' }}</span>
          </div>
          <div class="server-footer-card__health">
            <span>Health {{ health.score }}/100</span>
            <div class="server-health-bar" :class="`is-${serverTone}`" aria-hidden="true">
              <span :style="{ width: `${Math.max(0, Math.min(100, Number(health.score || 0)))}%` }"></span>
            </div>
          </div>
        </template>
        <div v-else class="server-footer-card__compact-bar" :class="`is-${serverTone}`" aria-hidden="true">
          <span :style="{ width: `${Math.max(0, Math.min(100, Number(health.score || 0)))}%` }"></span>
        </div>
      </div>

      <div v-if="footerMenuOpen" class="sidebar-popover footer-popover">
        <button type="button" class="sidebar-popover__item" @click="toggleServerSwitcher">
          <i class="mdi mdi-swap-horizontal"></i>
          <span>Switch server</span>
        </button>
        <button type="button" class="sidebar-popover__item" @click="refreshHealth">
          <i class="mdi mdi-heart-pulse"></i>
          <span>Run health check</span>
        </button>
        <button type="button" class="sidebar-popover__item" @click="copyConnectionString">
          <i class="mdi mdi-content-copy"></i>
          <span>Copy connection string</span>
        </button>
        <button type="button" class="sidebar-popover__item" @click="openServerDrawer">
          <i class="mdi mdi-information-outline"></i>
          <span>Open server drawer</span>
        </button>
      </div>

      <div v-if="serverSwitcherOpen" class="sidebar-popover footer-popover footer-popover--switcher">
        <div class="sidebar-popover__title">Switch server</div>
        <input v-model.trim="serverSearchQuery" class="sidebar-switcher-search" type="search" placeholder="Filter servers">
        <div class="sidebar-switcher-list">
          <button
            v-for="server in filteredKnownServers"
            :key="server.id"
            type="button"
            class="sidebar-switcher-item"
            :class="{ active: server.url === currentOrigin }"
            @click="switchServer(server)"
          >
            <span class="sidebar-status-dot" :class="server.url === currentOrigin ? `is-${serverTone}` : 'is-muted'"></span>
            <span class="sidebar-switcher-item__copy">
              <span>{{ server.name }}</span>
              <small>{{ server.url }}</small>
            </span>
          </button>
        </div>
        <button type="button" class="sidebar-popover__item" @click="promptAddServer">
          <i class="mdi mdi-plus"></i>
          <span>Add server</span>
        </button>
      </div>

      <button
        type="button"
        class="sidebar-footer-toggle d-none d-lg-inline-flex"
        :aria-label="effectiveCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        @click="toggleCollapse"
      >
        <i class="mdi" :class="effectiveCollapsed ? 'mdi-chevron-right' : 'mdi-chevron-left'"></i>
      </button>
    </div>

    <div
      v-if="contextMenu.item"
      class="sidebar-context-menu"
      :style="{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }"
      role="menu"
    >
      <button type="button" class="sidebar-context-menu__item" @click="togglePin(contextMenu.item)">
        <i class="mdi" :class="isPinned(contextMenu.item.id) ? 'mdi-pin-off-outline' : 'mdi-pin-outline'"></i>
        <span>{{ isPinned(contextMenu.item.id) ? 'Unpin from top' : 'Pin to top' }}</span>
      </button>
    </div>

    <div
      v-if="tooltip.item && effectiveCollapsed"
      id="sidebar-tooltip"
      class="sidebar-tooltip-card"
      :style="{ top: `${tooltip.y}px`, left: `${tooltip.x}px` }"
      role="tooltip"
    >
      <div class="sidebar-tooltip-card__title">{{ tooltip.item.label }}</div>
      <div class="sidebar-tooltip-card__meta">
        <span v-if="badgeForItem(tooltip.item)" class="sidebar-counter-badge" :class="`is-${badgeForItem(tooltip.item).tone}`">{{ badgeForItem(tooltip.item).text }}</span>
        <span v-if="tooltip.item.provider">{{ tooltip.item.provider }}</span>
        <span v-else-if="tooltip.item.parentLabel">{{ tooltip.item.parentLabel }}</span>
        <span v-else-if="itemStatusTone(tooltip.item)" class="sidebar-status-inline">{{ itemStatusTone(tooltip.item) }}</span>
      </div>
    </div>

    <div v-if="jumpHintVisible" class="sidebar-jump-hint">g + key</div>

    <DetailDrawer v-model="serverDrawerOpen" title="Server status" :subtitle="serverName">
      <div class="server-drawer-stack">
        <div class="server-drawer-card">
          <div class="server-drawer-card__row">
            <span>Health</span>
            <StatusBadge :state="serverTone === 'error' ? 'error' : serverTone === 'warn' ? 'warn' : serverTone === 'ok' ? 'ok' : 'muted'" :label="health.summary || 'Telemetry available'" />
          </div>
          <div class="server-drawer-grid">
            <div>
              <span class="server-drawer-label">Hostname</span>
              <strong>{{ serverName }}</strong>
            </div>
            <div>
              <span class="server-drawer-label">Origin</span>
              <strong>{{ currentOrigin }}</strong>
            </div>
            <div>
              <span class="server-drawer-label">CPU</span>
              <strong>{{ formatPercent(snap.cpu_pct) }}</strong>
            </div>
            <div>
              <span class="server-drawer-label">Memory</span>
              <strong>{{ formatPercent(snap.ram_pct) }}</strong>
            </div>
            <div>
              <span class="server-drawer-label">Inbound</span>
              <strong>{{ formatRate(snap.net_rx_rate) }}</strong>
            </div>
            <div>
              <span class="server-drawer-label">Outbound</span>
              <strong>{{ formatRate(snap.net_tx_rate) }}</strong>
            </div>
          </div>
        </div>

        <div class="server-drawer-card">
          <div class="server-drawer-card__row">
            <span>Quick actions</span>
          </div>
          <div class="server-drawer-actions">
            <button type="button" class="sidebar-action-btn" @click="refreshHealth">
              <i class="mdi mdi-heart-pulse"></i>
              <span>Run health check</span>
            </button>
            <button type="button" class="sidebar-action-btn" @click="copyConnectionString">
              <i class="mdi mdi-console-network-outline"></i>
              <span>Copy SSH target</span>
            </button>
            <button type="button" class="sidebar-action-btn" @click="toggleServerSwitcher">
              <i class="mdi mdi-swap-horizontal"></i>
              <span>Switch server</span>
            </button>
          </div>
        </div>

        <div class="server-drawer-card">
          <div class="server-drawer-card__row">
            <span>Health checks</span>
            <span>{{ health.checks.length }}</span>
          </div>
          <div class="server-check-list">
            <div v-for="check in health.checks" :key="check.name" class="server-check-item">
              <div>
                <strong>{{ check.name }}</strong>
                <p>{{ check.message }}</p>
              </div>
              <StatusBadge :state="normalizeCheckState(check.status)" :label="check.status" />
            </div>
          </div>
        </div>
      </div>
    </DetailDrawer>
  </aside>
</template>

<script>
import { useDocumentVisibility } from '@vueuse/core'
import { useAuthStore } from '@/stores/auth'
import { useLayoutStore } from '@/stores/layout'
import { useMetricsStore } from '@/stores/metrics'
import api from '@/services/api'
import appConfig from '@/app.config.json'
import DetailDrawer from '@/components/ui/detail-drawer.vue'
import StatusBadge from '@/components/ui/status-badge.vue'
import Tooltip from '@/components/ui/tooltip.vue'
import { getHealthTone } from '@/utils/health'
import {
  sidebarSections,
  flattenSidebarItems,
  findSidebarItemById,
  findSidebarItemByRoute
} from './menu'

const SIDEBAR_SCROLL_KEY = 'sidebar:scroll'
const SECTION_STATE_KEY = 'sidebar:sections'
const PARENT_STATE_KEY = 'sidebar:parents'
const PINNED_KEY = 'sidebar:pinned'
const VISITED_KEY = 'sidebar:visited'
const HIDDEN_SECTIONS_KEY = 'sidebar:hidden-sections'
const SERVERS_KEY = 'sidebar:servers'

function safeParse(value, fallback) {
  try {
    return JSON.parse(value ?? '')
  } catch {
    return fallback
  }
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function capCount(value, limit = 99) {
  if (!value) return ''
  return value > limit ? `${limit}+` : String(value)
}

function formatRate(value) {
  const amount = Number(value) || 0
  if (amount >= 1024 * 1024) return `${(amount / (1024 * 1024)).toFixed(1)} MB/s`
  if (amount >= 1024) return `${Math.round(amount / 1024)} KB/s`
  return `${Math.round(amount)} B/s`
}

export default {
  name: 'Sidebar',
  setup() {
    return {
      authStore: useAuthStore(),
      documentVisibility: useDocumentVisibility(),
      layoutStore: useLayoutStore(),
      metricsStore: useMetricsStore()
    }
  },
  components: { DetailDrawer, StatusBadge, Tooltip },
  data() {
    return {
      badgeCounts: {
        alerts: 0,
        security: 0
      },
      health: {
        overall_status: 'unknown',
        score: 0,
        summary: '',
        checks: []
      },
      searchQuery: '',
      sectionState: safeParse(localStorage.getItem(SECTION_STATE_KEY), {}),
      parentState: safeParse(localStorage.getItem(PARENT_STATE_KEY), {}),
      autoExpandedParents: {},
      hiddenSections: safeParse(localStorage.getItem(HIDDEN_SECTIONS_KEY), []),
      pinnedIds: safeParse(localStorage.getItem(PINNED_KEY), []),
      visitedItems: safeParse(localStorage.getItem(VISITED_KEY), {}),
      knownServers: safeParse(localStorage.getItem(SERVERS_KEY), []),
      serverSearchQuery: '',
      sidebarNavigationIntent: null,
      restoredScrollPosition: false,
      refreshTimer: null,
      healthTimer: null,
      tooltip: { item: null, x: 0, y: 0 },
      contextMenu: { item: null, x: 0, y: 0 },
      draggingPinId: null,
      brandMenuOpen: false,
      brandPopoverTop: 0,
      footerMenuOpen: false,
      serverSwitcherOpen: false,
      serverDrawerOpen: false,
      reconnectPulse: false,
      jumpHintVisible: false,
      pendingJumpChord: false,
      pendingJumpTimer: null,
      flashItemId: null,
      viewportWidth: typeof window === 'undefined' ? 1440 : window.innerWidth,
      touchStartX: 0,
      touchCurrentX: 0,
      touchStartTime: 0,
      reducedMotion: typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
    }
  },
  computed: {
    sectionDefinitions() {
      return sidebarSections
    },
    flattenedItems() {
      return flattenSidebarItems()
    },
    currentOrigin() {
      return window.location.origin
    },
    wsConnected() {
      return this.metricsStore.wsConnected
    },
    liveSummary() {
      return this.metricsStore.liveSummary || { unreadAlerts: 0, activeBans: 0 }
    },
    snap() {
      return this.metricsStore.snap || {}
    },
    sidebarDensity() {
      return this.layoutStore.sidebarDensity
    },
    sidebarPosition() {
      return this.layoutStore.sidebarPosition
    },
    sidebarHidden() {
      return this.layoutStore.sidebarHidden
    },
    sidebarOpen() {
      return this.layoutStore.sidebarOpen
    },
    storedCollapsed() {
      return this.layoutStore.sidebarCollapsed
    },
    isMobileViewport() {
      return this.viewportWidth < 992
    },
    isCompactViewport() {
      return this.viewportWidth < 1100 && !this.isMobileViewport
    },
    effectiveCollapsed() {
      return !this.isMobileViewport && (this.storedCollapsed || this.isCompactViewport)
    },
    asideClasses() {
      return {
        collapsed: this.effectiveCollapsed,
        open: this.sidebarOpen && this.isMobileViewport,
        'mobile-interacting': this.touchStartX > 0 && this.isMobileViewport,
        'is-hidden': this.sidebarHidden,
        'is-right': this.sidebarPosition === 'right'
      }
    },
    sidebarStyle() {
      if (!this.isMobileViewport) return {}
      const isRight = this.sidebarPosition === 'right'
      const sidebarW = Math.min(window.innerWidth * 0.86, 320)

      // ── Open gesture (main.vue drives this via store) ──────────────────────
      const openDrag = this.layoutStore.swipeOpenDrag
      if (!this.sidebarOpen && openDrag !== null) {
        const offPx = sidebarW * 1.02
        const translateX = isRight
          ? offPx * (1 - openDrag)     // right edge → slide in from right
          : -offPx * (1 - openDrag)    // left  edge → slide in from left
        return { transform: `translateX(${translateX}px)`, transition: 'none' }
      }

      // ── Close gesture (sidebar own touch, live follow) ─────────────────────
      if (this.sidebarOpen && this.touchCurrentX > 0 && this.touchStartX > 0) {
        const diff = isRight
          ? Math.max(0, this.touchCurrentX - this.touchStartX)
          : Math.min(0, this.touchCurrentX - this.touchStartX)
        return { transform: `translateX(${diff}px)`, transition: 'none' }
      }

      // ── Resting state ──────────────────────────────────────────────────────
      if (this.sidebarOpen) {
        return { transform: 'translateX(0)', transition: 'transform 0.32s cubic-bezier(0.2, 0.8, 0.2, 1)' }
      }
      return {}
    },
    visibleSections() {
      return this.sectionDefinitions.filter(section => {
        if (this.sectionHasActive(section) || this.sectionHasMatch(section)) {
          return true
        }
        return !this.hiddenSections.includes(section.id)
      })
    },
    pinnedItems() {
      return this.pinnedIds.slice(0, 6).map(id => {
        const item = findSidebarItemById(id)
        return item || { id, label: 'Page no longer exists', orphan: true }
      })
    },
    filteredKnownServers() {
      const query = this.serverSearchQuery.trim().toLowerCase()
      if (!query) return this.knownServers
      return this.knownServers.filter(server => `${server.name} ${server.url}`.toLowerCase().includes(query))
    },
    serverName() {
      return this.snap.hostname || window.location.hostname || 'server'
    },
    serverSubline() {
      return window.location.host || 'connected host'
    },
    serverTone() {
      if (!this.wsConnected) return 'warn'
      if (!this.health || !this.health.score) return 'muted'
      return getHealthTone(this.health.score)
    },
    appVersion() {
      return appConfig.version || '1.0.0'
    },
    brandPopoverStyle() {
      const top = this.brandPopoverTop || 60
      if (this.sidebarPosition === 'right') {
        return { top: `${top}px`, left: 'auto', right: '10px' }
      }
      return { top: `${top}px`, left: '10px', right: 'auto' }
    }
  },
  watch: {
    '$route.path': {
      immediate: true,
      handler(path) {
        this.syncExpandedState()
        this.markRouteVisited(path)

        const isSidebarNavigation = this.sidebarNavigationIntent === this.normalizeRoutePath(path)
        this.sidebarNavigationIntent = null

        if (!isSidebarNavigation) {
          this.$nextTick(() => {
            requestAnimationFrame(() => {
              this.ensureActiveRouteVisibility()
            })
          })
        }

        this.persistScrollPosition()
      }
    },
    wsConnected(isConnected, wasConnected) {
      if (isConnected && wasConnected === false) {
        this.reconnectPulse = true
        window.setTimeout(() => {
          this.reconnectPulse = false
        }, 1200)
        this.applyLiveSummary()
      }
      if (!isConnected) {
        this.fetchBadgeCounts()
      }
    },
    liveSummary: {
      deep: true,
      handler() {
        if (this.wsConnected) {
          this.applyLiveSummary()
        }
      }
    },
    documentVisibility(value) {
      if (value === 'visible') {
        this.refreshHealth()
      }
    },
    searchQuery() {
      this.closeContextMenus()
    }
  },
  async mounted() {
    this.seedKnownServers()
    this.syncExpandedState()
    await Promise.all([this.fetchBadgeCounts(), this.refreshHealth()])
    this.$nextTick(() => {
      this.restoreScrollPosition()
      if (!this.restoredScrollPosition) {
        requestAnimationFrame(() => {
          this.ensureActiveRouteVisibility()
        })
      }
    })
    this.healthTimer = window.setInterval(() => {
      if (this.documentVisibility !== 'visible') return
      this.refreshHealth()
    }, Math.max(15000, appConfig.pollIntervalMs))
    window.addEventListener('resize', this.handleResize, { passive: true })
    document.addEventListener('keydown', this.onGlobalKeyDown)
    document.addEventListener('click', this.onGlobalClick, true)
    window.addEventListener('sentinel:close-sidebar-menus', this.closeContextMenus)
  },
  beforeUnmount() {
    this.persistScrollPosition()
    window.removeEventListener('resize', this.handleResize)
    document.removeEventListener('keydown', this.onGlobalKeyDown)
    document.removeEventListener('click', this.onGlobalClick, true)
    window.removeEventListener('sentinel:close-sidebar-menus', this.closeContextMenus)
    window.clearInterval(this.refreshTimer)
    window.clearInterval(this.healthTimer)
    window.clearTimeout(this.pendingJumpTimer)
  },
  methods: {
    normalizeRoutePath(path) {
      return String(path || '').split('?')[0].split('#')[0]
    },
    handleResize() {
      this.viewportWidth = window.innerWidth
      if (!this.isMobileViewport) {
        this.layoutStore.closeSidebar()
      }
      this.hideTooltip()
    },
    saveState(key, value) {
      localStorage.setItem(key, JSON.stringify(value))
    },
    toggleBrandMenu() {
      const willOpen = !this.brandMenuOpen
      this.closeContextMenus()
      if (willOpen) {
        const rect = this.$refs.brandRow?.getBoundingClientRect()
        this.brandPopoverTop = rect ? rect.bottom + 8 : 60
        this.brandMenuOpen = true
      }
    },
    closeContextMenus() {
      this.contextMenu = { item: null, x: 0, y: 0 }
      this.brandMenuOpen = false
      this.footerMenuOpen = false
      this.serverSwitcherOpen = false
    },
    onGlobalClick(event) {
      if (!this.$refs.sidebarRoot?.contains(event.target) && !event.target.closest('.brand-popover')) {
        this.closeContextMenus()
        this.hideTooltip()
      }
    },
    onTouchStart(event) {
      this.touchStartX = event.changedTouches?.[0]?.clientX || 0
      this.touchCurrentX = this.touchStartX
      this.touchStartTime = Date.now()
    },
    onTouchMove(event) {
      if (!this.isMobileViewport || !this.sidebarOpen) return
      this.touchCurrentX = event.changedTouches?.[0]?.clientX || 0
    },
    onTouchEnd(event) {
      if (!this.isMobileViewport || !this.sidebarOpen) return
      const currentX = event.changedTouches?.[0]?.clientX || 0
      const dx = currentX - this.touchStartX
      const dt = Date.now() - this.touchStartTime
      const velocity = Math.abs(dx / dt)

      const isRight = this.sidebarPosition === 'right'
      const closedDx = isRight ? dx > 60 : dx < -60
      const closedVelocity = velocity > 0.4 && (isRight ? dx > 0 : dx < 0)

      if (closedDx || closedVelocity) {
        this.layoutStore.closeSidebar()
      }

      this.touchStartX = 0
      this.touchCurrentX = 0
      this.touchStartTime = 0
    },
    persistScrollPosition() {
      const scrollContainer = this.$refs.scrollContainer
      if (!scrollContainer) return
      try {
        sessionStorage.setItem(SIDEBAR_SCROLL_KEY, String(scrollContainer.scrollTop || 0))
      } catch {
        // Ignore storage failures.
      }
    },
    restoreScrollPosition() {
      const scrollContainer = this.$refs.scrollContainer
      if (!scrollContainer) return
      const savedScroll = sessionStorage.getItem(SIDEBAR_SCROLL_KEY)
      if (savedScroll == null) return
      scrollContainer.scrollTop = Number(savedScroll) || 0
      this.restoredScrollPosition = true
    },
    ensureActiveRouteVisibility() {
      const scrollContainer = this.$refs.scrollContainer
      const routePath = this.normalizeRoutePath(this.$route.path)
      if (!scrollContainer || !routePath) return

      const activeLink = scrollContainer.querySelector(`[data-route="${routePath}"]`)
      if (!activeLink) return

      const containerRect = scrollContainer.getBoundingClientRect()
      const activeRect = activeLink.getBoundingClientRect()
      const isAbove = activeRect.top < containerRect.top
      const isBelow = activeRect.bottom > containerRect.bottom

      if (isAbove || isBelow) {
        activeLink.scrollIntoView({ block: 'nearest', behavior: this.reducedMotion ? 'auto' : 'smooth' })
        if (this.effectiveCollapsed) {
          const activeItem = findSidebarItemByRoute(routePath)
          this.flashItemId = activeItem?.id || null
          window.setTimeout(() => {
            this.flashItemId = null
          }, 900)
        }
      }
    },
    isRouteActive(link) {
      const path = this.normalizeRoutePath(this.$route.path)
      return path === link || path.startsWith(`${link}/`)
    },
    isItemActive(item) {
      if (item.link) {
        return this.isRouteActive(item.link)
      }
      return item.children?.some(child => this.isItemActive(child)) || false
    },
    linkClasses(item) {
      return {
        active: this.isItemActive(item),
        'is-flashing': this.flashItemId === item.id
      }
    },
    resolvedItemIcon(item) {
      return item.icon || 'mdi mdi-radiobox-blank'
    },
    markSidebarNavigationIntent(path) {
      this.sidebarNavigationIntent = this.normalizeRoutePath(path)
    },
    handleSidebarRouteClick(path) {
      this.markSidebarNavigationIntent(path)
      this.closeContextMenus()
      if (this.isMobileViewport) {
        this.layoutStore.closeSidebar()
      }
    },
    toggleCollapse() {
      this.layoutStore.toggleCollapsed()
      this.hideTooltip()
    },
    setSidebarDensity(value) {
      this.layoutStore.setSidebarDensity(value)
    },
    setSidebarPosition(value) {
      this.layoutStore.setSidebarPosition(value)
    },
    toggleSection(sectionId) {
      this.sectionState = {
        ...this.sectionState,
        [sectionId]: !this.sectionState[sectionId]
      }
      this.saveState(SECTION_STATE_KEY, this.sectionState)
    },
    sectionHasActive(section) {
      return section.items.some(item => this.isItemActive(item))
    },
    sectionHasMatch(section) {
      if (!this.searchQuery) return false
      return section.items.some(item => this.itemMatchesQuery(item))
    },
    isSectionExpanded(section) {
      if (this.sectionHasActive(section) || this.sectionHasMatch(section)) return true
      return this.sectionState[section.id] !== true
    },
    sectionSummary(section) {
      if (section.id === 'security') {
        const total = (this.badgeCounts.alerts || 0) + (this.badgeCounts.security || 0)
        if (total) return capCount(total)
      }
      return section.items.reduce((count, item) => count + (item.children?.length || 1), 0)
    },
    toggleSectionVisibility(sectionId) {
      if (this.hiddenSections.includes(sectionId)) {
        this.hiddenSections = this.hiddenSections.filter(id => id !== sectionId)
      } else {
        this.hiddenSections = [...this.hiddenSections, sectionId]
      }
      this.saveState(HIDDEN_SECTIONS_KEY, this.hiddenSections)
    },
    resetSectionVisibility() {
      this.hiddenSections = []
      this.saveState(HIDDEN_SECTIONS_KEY, this.hiddenSections)
    },
    toggleParent(parentId) {
      this.parentState = {
        ...this.parentState,
        [parentId]: !this.parentState[parentId]
      }
      this.saveState(PARENT_STATE_KEY, this.parentState)
    },
    isParentExpanded(item) {
      if (this.searchQuery && item.children?.some(child => this.itemMatchesQuery(child))) {
        return true
      }
      return this.parentState[item.id] === true || this.autoExpandedParents[item.id] === true
    },
    syncExpandedState() {
      const next = {}
      this.sectionDefinitions.forEach(section => {
        section.items.forEach(item => {
          if (item.children?.some(child => this.isItemActive(child))) {
            next[item.id] = true
          }
        })
      })
      this.autoExpandedParents = next
    },
    itemSearchText(item) {
      return [item.label, item.sectionLabel, item.parentLabel, ...(item.keywords || [])].filter(Boolean).join(' ').toLowerCase()
    },
    itemMatchesQuery(item) {
      if (!this.searchQuery) return true
      const query = this.searchQuery.toLowerCase()
      if (this.itemSearchText(item).includes(query)) return true
      return item.children?.some(child => this.itemMatchesQuery(child)) || false
    },
    shouldDimItem(item) {
      return !!this.searchQuery && !this.itemMatchesQuery(item)
    },
    highlightLabel(label) {
      if (!this.searchQuery) return escapeHtml(label)
      const query = escapeRegExp(this.searchQuery)
      const matcher = new RegExp(`(${query})`, 'ig')
      return escapeHtml(label).replace(matcher, '<mark>$1</mark>')
    },
    badgeForItem(item) {
      if (item.badgeKey === 'alerts' && this.badgeCounts.alerts > 0) {
        return {
          text: capCount(this.badgeCounts.alerts),
          tone: this.badgeCounts.alerts > 9 ? 'critical' : 'warn',
          ariaLabel: `${this.badgeCounts.alerts} unread alerts`
        }
      }
      if (item.badgeKey === 'security' && this.badgeCounts.security > 0) {
        return {
          text: capCount(this.badgeCounts.security),
          tone: this.badgeCounts.security > 3 ? 'critical' : 'warn',
          ariaLabel: `${this.badgeCounts.security} active security issues`
        }
      }
      if (item.children?.length) {
        const childCount = item.children.reduce((total, child) => {
          const badge = this.badgeForItem(child)
          return total + (badge ? Number.parseInt(badge.text, 10) || 0 : 0)
        }, 0)
        if (childCount) {
          return { text: capCount(childCount), tone: 'warn', ariaLabel: `${childCount} pending items` }
        }
      }
      return null
    },
    tagForItem(item) {
      if (item.tag === 'new' && this.visitedItems[item.id]) return null
      if (!item.tag) return null
      return item.tag.toUpperCase()
    },
    itemStatusTone(item) {
      if (item.statusKey === 'live') {
        return this.wsConnected ? 'ok' : 'warn'
      }
      if (item.statusKey === 'health') {
        return this.serverTone
      }
      if (item.statusKey === 'services') {
        const services = this.metricsStore.services || []
        const failing = services.filter(service => service.status && service.status !== 'active').length
        if (!services.length) return 'muted'
        return failing ? 'warn' : 'ok'
      }
      return null
    },
    accessibilityLabel(item) {
      const parts = [item.label]
      const badge = this.badgeForItem(item)
      const tag = this.tagForItem(item)
      if (badge) parts.push(badge.ariaLabel)
      if (tag) parts.push(tag)
      return parts.join(', ')
    },
    openContextMenu(event, item) {
      window.dispatchEvent(new CustomEvent('sentinel:close-sidebar-menus'))
      this.contextMenu = {
        item,
        x: Math.min(event.clientX, window.innerWidth - 220),
        y: Math.min(event.clientY, window.innerHeight - 80)
      }
    },
    isPinned(id) {
      return this.pinnedIds.includes(id)
    },
    togglePin(item) {
      if (!item?.id) return
      if (this.isPinned(item.id)) {
        this.unpinItem(item.id)
      } else if (this.pinnedIds.length < 6) {
        this.pinnedIds = [...this.pinnedIds, item.id]
        this.saveState(PINNED_KEY, this.pinnedIds)
      }
      this.contextMenu = { item: null, x: 0, y: 0 }
    },
    unpinItem(id) {
      this.pinnedIds = this.pinnedIds.filter(itemId => itemId !== id)
      this.saveState(PINNED_KEY, this.pinnedIds)
    },
    startPinDrag(item) {
      this.draggingPinId = item.id
    },
    dropPinnedItem(targetItem) {
      if (!this.draggingPinId || this.draggingPinId === targetItem.id) return
      const next = [...this.pinnedIds]
      const fromIndex = next.indexOf(this.draggingPinId)
      const toIndex = next.indexOf(targetItem.id)
      if (fromIndex === -1 || toIndex === -1) return
      next.splice(toIndex, 0, next.splice(fromIndex, 1)[0])
      this.pinnedIds = next
      this.draggingPinId = null
      this.saveState(PINNED_KEY, this.pinnedIds)
    },
    clearSearch() {
      this.searchQuery = ''
      this.$refs.searchInput?.focus()
    },
    focusSearch() {
      if (this.effectiveCollapsed) {
        window.dispatchEvent(new CustomEvent('sentinel:command-palette-open'))
        return
      }
      this.$nextTick(() => {
        this.$refs.searchInput?.focus()
      })
    },
    showTooltip(event, item) {
      if (!this.effectiveCollapsed) return
      const rect = event.currentTarget?.getBoundingClientRect?.()
      if (!rect) return
      this.tooltip = {
        item,
        x: rect.right + 12,
        y: rect.top + rect.height / 2
      }
    },
    hideTooltip() {
      this.tooltip = { item: null, x: 0, y: 0 }
    },
    async fetchBadgeCounts() {
      if (!this.authStore.loggedIn) return
      try {
        const [alertsCountRes, securityRes] = await Promise.all([
          api.getAlertCount(),
          api.getSecurityStatus()
        ])
        this.badgeCounts.alerts = alertsCountRes.data?.count || 0
        this.badgeCounts.security = securityRes.data?.active_bans || 0
      } catch (err) {
        if (err.response?.status !== 401) {
          console.error('Failed to fetch sidebar counts:', err)
        }
      }
    },
    applyLiveSummary() {
      this.badgeCounts = {
        alerts: Number(this.liveSummary.unreadAlerts || 0),
        security: Number(this.liveSummary.activeBans || 0)
      }
    },
    async refreshHealth() {
      try {
        const { data } = await api.getHealth()
        this.health = {
          overall_status: data?.overall_status || 'unknown',
          score: Number(data?.score) || 0,
          summary: data?.summary || '',
          checks: Array.isArray(data?.checks) ? data.checks : []
        }
      } catch (err) {
        this.health = {
          overall_status: 'warning',
          score: 0,
          summary: 'Health data unavailable',
          checks: []
        }
        if (err.response?.status !== 401) {
          console.error('Failed to fetch health:', err)
        }
      }
    },
    formatPercent(value) {
      return `${Number(value || 0).toFixed(1)}%`
    },
    formatRate,
    toggleFooterMenu() {
      this.closeContextMenus()
      this.footerMenuOpen = !this.footerMenuOpen
    },
    toggleServerSwitcher() {
      this.footerMenuOpen = false
      this.serverSwitcherOpen = !this.serverSwitcherOpen
    },
    seedKnownServers() {
      if (!this.knownServers.length) {
        this.knownServers = [{
          id: 'current',
          name: this.serverName,
          url: window.location.origin
        }]
        this.saveState(SERVERS_KEY, this.knownServers)
        return
      }

      if (!this.knownServers.some(server => server.url === window.location.origin)) {
        this.knownServers = [{ id: 'current', name: this.serverName, url: window.location.origin }, ...this.knownServers]
        this.saveState(SERVERS_KEY, this.knownServers)
      }
    },
    promptAddServer() {
      const name = window.prompt('Server label')
      if (!name) return
      const url = window.prompt('Server URL (for example https://sentinel.example.com)')
      if (!url) return

      try {
        const normalized = new URL(url).origin
        if (this.knownServers.some(server => server.url === normalized)) return
        this.knownServers = [...this.knownServers, {
          id: `${Date.now()}`,
          name,
          url: normalized
        }]
        this.saveState(SERVERS_KEY, this.knownServers)
      } catch {
        window.alert('Enter a valid URL for the target SentinelCore instance.')
      }
    },
    switchServer(server) {
      if (!server?.url) return
      if (server.url === this.currentOrigin) {
        this.serverSwitcherOpen = false
        return
      }
      window.location.assign(`${server.url}${this.$route.fullPath}`)
    },
    copyConnectionString() {
      navigator.clipboard.writeText(`ssh ${window.location.hostname}`).catch(() => {})
      this.closeContextMenus()
    },
    openServerDrawer() {
      this.serverDrawerOpen = true
      this.closeContextMenus()
    },
    openCommandPalette() {
      window.dispatchEvent(new CustomEvent('sentinel:command-palette-open'))
      this.closeContextMenus()
    },
    goToDashboard() {
      this.handleSidebarRouteClick('/dashboard')
      this.$router.push('/dashboard')
    },
    markRouteVisited(path) {
      const matchedItem = findSidebarItemByRoute(this.normalizeRoutePath(path))
      if (!matchedItem) return
      this.visitedItems = {
        ...this.visitedItems,
        [matchedItem.id]: Date.now()
      }
      this.saveState(VISITED_KEY, this.visitedItems)
      if (matchedItem.id === 'alerts') {
        this.badgeCounts.alerts = 0
      }
      if (matchedItem.id === 'security-center') {
        this.badgeCounts.security = 0
      }
    },
    normalizeCheckState(status) {
      if (status === 'healthy') return 'ok'
      if (status === 'warning') return 'warn'
      if (status === 'critical') return 'critical'
      return 'muted'
    },
    async logout() {
      try {
        await api.logout()
      } catch {
        // Ignore logout transport errors.
      }
      this.authStore.logout()
      this.$router.push('/login')
    },
    onGlobalKeyDown(event) {
      if (event.key === 'Escape') {
        this.closeContextMenus()
        this.hideTooltip()
        return
      }

      const isTypingTarget = ['INPUT', 'TEXTAREA', 'SELECT'].includes(event.target?.tagName) || event.target?.isContentEditable
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'b') {
        event.preventDefault()
        this.layoutStore.toggleVisibility()
        return
      }
      if (event.key === '[' && !isTypingTarget) {
        event.preventDefault()
        this.layoutStore.setCollapsed(true)
        return
      }
      if (event.key === ']' && !isTypingTarget) {
        event.preventDefault()
        this.layoutStore.setCollapsed(false)
        return
      }
      if (event.key === '/' && !isTypingTarget) {
        event.preventDefault()
        this.focusSearch()
        return
      }
      if (isTypingTarget) return

      if (event.key.toLowerCase() === 'g') {
        this.pendingJumpChord = true
        this.jumpHintVisible = true
        window.clearTimeout(this.pendingJumpTimer)
        this.pendingJumpTimer = window.setTimeout(() => {
          this.pendingJumpChord = false
          this.jumpHintVisible = false
        }, 1200)
        return
      }

      if (this.pendingJumpChord && /^[a-z]$/i.test(event.key)) {
        event.preventDefault()
        this.pendingJumpChord = false
        this.jumpHintVisible = false
        this.executeJump(event.key.toLowerCase())
        return
      }

      if (this.$refs.sidebarRoot?.contains(document.activeElement)) {
        this.handleKeyboardNavigation(event)
      }
    },
    executeJump(key) {
      const routes = {
        d: '/dashboard',
        a: '/alerts',
        s: '/services',
        m: '/monitoring',
        l: '/logs',
        u: '/users',
        t: '/terminal',
        f: '/firewall',
        g: '/settings'
      }
      const route = routes[key]
      if (!route) return
      this.handleSidebarRouteClick(route)
      this.$router.push(route)
    },
    handleKeyboardNavigation(event) {
      const focusable = [...this.$refs.sidebarRoot.querySelectorAll('a.sidebar-link, button.sidebar-link--button, button.sidebar-section-toggle, button.server-footer-card, button.sidebar-footer-toggle')]
        .filter(element => element.offsetParent !== null)
      if (!focusable.length) return

      const currentIndex = focusable.indexOf(document.activeElement)
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        focusable[(currentIndex + 1 + focusable.length) % focusable.length].focus()
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault()
        focusable[(currentIndex - 1 + focusable.length) % focusable.length].focus()
      }
      if (event.key === 'ArrowRight' && document.activeElement?.classList.contains('sidebar-link--button')) {
        event.preventDefault()
        const parentId = this.findParentIdByLabel(document.activeElement)
        if (parentId && this.parentState[parentId] !== true) {
          this.toggleParent(parentId)
        }
      }
      if (event.key === 'ArrowLeft' && document.activeElement?.classList.contains('sidebar-link--button')) {
        event.preventDefault()
        const parentId = this.findParentIdByLabel(document.activeElement)
        if (parentId && this.parentState[parentId] === true) {
          this.toggleParent(parentId)
        }
      }
      if (event.key === 'Enter' && document.activeElement) {
        document.activeElement.click()
      }
    },
    findParentIdByLabel(element) {
      const label = element.querySelector('.sidebar-link__text')?.textContent?.trim()
      return this.flattenedItems.find(item => item.label === label && item.children)?.id || null
    }
  }
}
</script>
