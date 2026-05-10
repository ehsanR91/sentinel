<template>
  <teleport to="body">
    <div v-if="open" class="command-palette-backdrop" @click.self="closePalette">
      <div class="command-palette" role="dialog" aria-modal="true" aria-label="Command palette">
        <div class="command-palette__header">
          <i class="mdi mdi-magnify" aria-hidden="true"></i>
          <input
            ref="input"
            v-model.trim="query"
            class="command-palette__input"
            type="search"
            autocomplete="off"
            spellcheck="false"
            placeholder="Search navigation, settings, recent pages, or live records"
          >
          <button type="button" class="command-palette__close" aria-label="Close command palette" @click="closePalette">
            <i class="mdi mdi-close"></i>
          </button>
        </div>

        <div class="command-palette__body">
          <div v-if="flattenedResults.length === 0" class="command-palette__empty">
            No results for "{{ query }}"
          </div>

          <template v-for="group in groupedResults">
            <section v-if="group.items.length" :key="group.label" class="command-palette__group">
              <div class="command-palette__group-title">{{ group.label }}</div>
              <button
                v-for="item in group.items"
                :key="item.key"
                type="button"
                class="command-palette__item"
                :class="{ active: flattenedResults[selectedIndex]?.key === item.key }"
                @mouseenter="setActiveByKey(item.key)"
                @click="activateItem(item)"
              >
                <i :class="item.icon || 'mdi mdi-arrow-top-right'" aria-hidden="true"></i>
                <span class="command-palette__copy">
                  <strong>{{ item.label }}</strong>
                  <small>{{ item.description }}</small>
                </span>
                <span class="command-palette__meta">{{ item.meta }}</span>
              </button>
            </section>
          </template>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script>
import api from '@/services/api'
import { navigationSearchEntries, settingsCommandEntries } from '@/components/menu'
import { useMetricsStore } from '@/stores/metrics'

const RECENT_KEY = 'command-palette:recent'

function safeParse(value, fallback) {
  try {
    return JSON.parse(value ?? '')
  } catch {
    return fallback
  }
}

function includesQuery(value, query) {
  return String(value || '').toLowerCase().includes(query)
}

function routeBoost(route, liveSummary, failingServices) {
  if (route === '/alerts') {
    return Math.min(48, Number(liveSummary.unreadAlerts || 0) * 4)
  }
  if (route === '/security') {
    return Math.min(36, Number(liveSummary.activeBans || 0) * 6)
  }
  if (route === '/services') {
    return Math.min(30, Number(failingServices || 0) * 6)
  }
  return 0
}

function scoreItem(item, query, recentRoutes, liveSummary, failingServices) {
  const normalizedQuery = query.trim().toLowerCase()
  const label = String(item.label || '').toLowerCase()
  const description = String(item.description || '').toLowerCase()
  const meta = String(item.meta || '').toLowerCase()
  const keywords = Array.isArray(item.keywords) ? item.keywords.map(keyword => String(keyword).toLowerCase()) : []
  let score = routeBoost(item.route, liveSummary, failingServices)

  if (!normalizedQuery) {
    score += item.group === 'Recent' ? 120 : 40
  } else {
    if (label === normalizedQuery) score += 180
    if (label.startsWith(normalizedQuery)) score += 130
    if (label.includes(normalizedQuery)) score += 90
    if (description.includes(normalizedQuery)) score += 30
    if (meta.includes(normalizedQuery)) score += 45
    if (keywords.some(keyword => keyword.startsWith(normalizedQuery))) score += 55
    if (keywords.some(keyword => keyword.includes(normalizedQuery))) score += 25
  }

  const recentIndex = recentRoutes.indexOf(item.route)
  if (recentIndex >= 0) {
    score += Math.max(6, 30 - recentIndex * 4)
  }

  return score
}

export default {
  name: 'CommandPalette',
  setup() {
    return {
      metricsStore: useMetricsStore()
    }
  },
  data() {
    return {
      open: false,
      query: '',
      selectedIndex: 0,
      recentPages: safeParse(localStorage.getItem(RECENT_KEY), []),
      serverResults: {
        alerts: [],
        services: [],
        containers: []
      },
      searchTimer: null,
      loadingServerResults: false
    }
  },
  computed: {
    liveSummary() {
      return this.metricsStore.liveSummary || { unreadAlerts: 0, activeBans: 0 }
    },
    failingServices() {
      const services = this.metricsStore.services || []
      return services.filter(service => service.status && service.status !== 'active').length
    },
    navigationItems() {
      return navigationSearchEntries().map(item => ({
        key: `nav-${item.id}`,
        label: item.label,
        description: item.sectionLabel,
        meta: this.routeMeta(item.route),
        icon: item.icon,
        route: item.route,
        group: 'Navigation',
        keywords: item.keywords || []
      }))
    },
    settingsItems() {
      return settingsCommandEntries.map(item => ({
        key: item.id,
        label: item.label,
        description: 'Configuration surface',
        meta: item.route,
        icon: 'mdi mdi-cog-outline',
        route: item.route,
        group: 'Settings',
        keywords: item.keywords || []
      }))
    },
    recentItems() {
      return this.recentPages.map(item => ({
        key: `recent-${item.route}`,
        label: item.label,
        description: 'Recently visited',
        meta: item.route,
        icon: item.icon,
        route: item.route,
        group: 'Recent',
        keywords: []
      }))
    },
    filteredNavigationItems() {
      return this.rankItems(this.navigationItems, 10)
    },
    filteredSettingsItems() {
      return this.rankItems(this.settingsItems, 5)
    },
    filteredRecentItems() {
      return this.rankItems(this.recentItems, 6)
    },
    groupedResults() {
      const groups = [
        { label: 'Recent', items: this.filteredRecentItems },
        { label: 'Navigation', items: this.filteredNavigationItems },
        { label: 'Settings', items: this.filteredSettingsItems }
      ]

      if (this.query.trim().length > 2) {
        groups.push(
          { label: 'Alerts', items: this.serverResults.alerts },
          { label: 'Services', items: this.serverResults.services },
          { label: 'Containers', items: this.serverResults.containers }
        )
      }

      return groups
    },
    flattenedResults() {
      return this.groupedResults.flatMap(group => group.items)
    }
  },
  watch: {
    '$route.fullPath': {
      immediate: true,
      handler(path) {
        const match = this.navigationItems.find(item => item.route === path)
        if (!match) return
        this.recentPages = [match, ...this.recentPages.filter(item => item.route !== match.route)].slice(0, 6)
        localStorage.setItem(RECENT_KEY, JSON.stringify(this.recentPages.map(item => ({
          route: item.route,
          label: item.label,
          icon: item.icon
        }))))
      }
    },
    query() {
      this.selectedIndex = 0
      window.clearTimeout(this.searchTimer)
      if (this.query.trim().length <= 2) {
        this.serverResults = { alerts: [], services: [], containers: [] }
        return
      }

      this.searchTimer = window.setTimeout(() => {
        this.searchServerRecords()
      }, 180)
    }
  },
  mounted() {
    window.addEventListener('sentinel:command-palette-open', this.openPalette)
    window.addEventListener('sentinel:command-palette-close', this.closePalette)
    document.addEventListener('keydown', this.onGlobalKeyDown)
  },
  beforeUnmount() {
    window.removeEventListener('sentinel:command-palette-open', this.openPalette)
    window.removeEventListener('sentinel:command-palette-close', this.closePalette)
    document.removeEventListener('keydown', this.onGlobalKeyDown)
    window.clearTimeout(this.searchTimer)
  },
  methods: {
    routeMeta(route) {
      if (route === '/alerts' && this.liveSummary.unreadAlerts) {
        return `${route} · ${this.liveSummary.unreadAlerts} unread`
      }
      if (route === '/security' && this.liveSummary.activeBans) {
        return `${route} · ${this.liveSummary.activeBans} active bans`
      }
      if (route === '/services' && this.failingServices) {
        return `${route} · ${this.failingServices} degraded`
      }
      return route
    },
    rankItems(items, limit) {
      const query = this.query.trim().toLowerCase()
      const recentRoutes = this.recentPages.map(item => item.route)
      return items
        .filter(item => {
          if (!query) return true
          return includesQuery(item.label, query) || includesQuery(item.description, query) || includesQuery(item.meta, query) || item.keywords.some(keyword => includesQuery(keyword, query))
        })
        .map(item => ({
          ...item,
          score: scoreItem(item, query, recentRoutes, this.liveSummary, this.failingServices)
        }))
        .sort((left, right) => right.score - left.score || left.label.localeCompare(right.label))
        .slice(0, limit)
    },
    openPalette() {
      this.open = true
      this.query = ''
      this.selectedIndex = 0
      this.$nextTick(() => {
        this.$refs.input?.focus()
      })
    },
    closePalette() {
      this.open = false
    },
    onGlobalKeyDown(event) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        this.openPalette()
        return
      }
      if (!this.open) return
      if (event.key === 'Escape') {
        event.preventDefault()
        this.closePalette()
        return
      }
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        this.selectedIndex = (this.selectedIndex + 1) % Math.max(this.flattenedResults.length, 1)
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault()
        this.selectedIndex = (this.selectedIndex - 1 + Math.max(this.flattenedResults.length, 1)) % Math.max(this.flattenedResults.length, 1)
      }
      if (event.key === 'Enter') {
        event.preventDefault()
        const item = this.flattenedResults[this.selectedIndex]
        if (item) {
          this.activateItem(item)
        }
      }
    },
    setActiveByKey(key) {
      const index = this.flattenedResults.findIndex(item => item.key === key)
      if (index >= 0) {
        this.selectedIndex = index
      }
    },
    activateItem(item) {
      this.closePalette()
      if (item.route) {
        this.$router.push(item.route)
      }
    },
    async searchServerRecords() {
      if (!this.$store.getters['auth/loggedIn']) return
      const query = this.query.trim().toLowerCase()
      this.loadingServerResults = true
      try {
        const [alertsRes, servicesRes, containersRes] = await Promise.all([
          api.getAlerts({ limit: 25 }),
          api.getManagedServices(),
          api.getContainers()
        ])

        const alerts = Array.isArray(alertsRes.data) ? alertsRes.data : []
        const services = Array.isArray(servicesRes.data) ? servicesRes.data : []
        const containers = Array.isArray(containersRes.data) ? containersRes.data : []

        this.serverResults = {
          alerts: alerts
            .filter(alert => includesQuery(alert.summary || alert.message || alert.source, query))
            .slice(0, 4)
            .map(alert => ({
              key: `alert-${alert.id}`,
              label: alert.summary || alert.message || 'Alert',
              description: alert.source || 'Alert record',
              meta: '/alerts',
              icon: 'mdi mdi-bell-alert-outline',
              route: '/alerts'
            })),
          services: services
            .filter(service => includesQuery(service.name, query) || includesQuery(service.displayName, query))
            .slice(0, 4)
            .map(service => ({
              key: `service-${service.name}`,
              label: service.displayName || service.name,
              description: service.status || 'Managed service',
              meta: '/services',
              icon: 'mdi mdi-cog-refresh-outline',
              route: '/services'
            })),
          containers: containers
            .filter(container => includesQuery(container.name, query) || includesQuery(container.image, query))
            .slice(0, 4)
            .map(container => ({
              key: `container-${container.id}`,
              label: container.name || container.id,
              description: container.image || 'Container',
              meta: '/containers',
              icon: 'mdi mdi-docker',
              route: '/containers'
            }))
        }
      } catch (err) {
        if (err.response?.status !== 401) {
          console.error('Command palette server search failed:', err)
        }
      } finally {
        this.loadingServerResults = false
      }
    }
  }
}
</script>

<style scoped>
.command-palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: min(10vh, 6rem) 1rem 1rem;
  background: rgba(8, 12, 20, 0.58);
  backdrop-filter: blur(10px);
}

.command-palette {
  width: min(720px, 100%);
  max-height: 78vh;
  overflow: hidden;
  border: 1px solid var(--sc-border);
  border-radius: 18px;
  background: color-mix(in srgb, var(--sc-bg-card) 88%, black 12%);
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.45);
}

.command-palette__header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.1rem;
  border-bottom: 1px solid var(--sc-border);
}

.command-palette__header > i,
.command-palette__close {
  color: var(--sc-text-muted);
}

.command-palette__input {
  flex: 1;
  border: 0;
  background: transparent;
  color: var(--sc-text);
  font-size: 0.98rem;
}

.command-palette__input:focus {
  outline: none;
}

.command-palette__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 10px;
  background: transparent;
}

.command-palette__body {
  max-height: calc(78vh - 74px);
  overflow: auto;
  padding: 0.5rem;
}

.command-palette__group {
  padding: 0.5rem;
}

.command-palette__group-title {
  padding: 0.15rem 0.5rem 0.55rem;
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--sc-text-muted);
}

.command-palette__item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.8rem 0.85rem;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--sc-text);
  text-align: left;
}

.command-palette__item.active,
.command-palette__item:hover {
  background: rgba(74, 158, 255, 0.12);
}

.command-palette__copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.command-palette__copy small,
.command-palette__meta,
.command-palette__empty {
  color: var(--sc-text-muted);
}

.command-palette__copy strong,
.command-palette__copy small,
.command-palette__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-palette__meta {
  margin-left: auto;
  font-family: var(--bs-font-monospace);
  font-size: 0.76rem;
}

.command-palette__empty {
  padding: 2rem 1rem;
  text-align: center;
}
</style>