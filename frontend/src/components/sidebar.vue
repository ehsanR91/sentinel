<template>
<div id="sidebar-menu" :class="{ collapsed: sidebarCollapsed }">
<!-- Logo + collapse toggle -->
<div class="sidebar-logo">
<router-link to="/dashboard" class="logo-text">
<div class="logo-icon">
<i class="mdi mdi-shield-half-full"></i>
</div>
<span class="logo-name">Sentinel<span>Core</span></span>
</router-link>
<!-- Collapse button moved to edge -->
<button class="collapse-btn-edge d-none d-lg-flex" @click="toggleCollapse" :title="sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'" aria-label="Toggle sidebar" tabindex="0">
<i class="mdi" :class="sidebarCollapsed ? 'mdi-chevron-right' : 'mdi-chevron-left'" aria-hidden="true"></i>
</button>
</div>

    <!-- Nav -->
    <div class="sidebar-scroll">
      <ul class="sidebar-nav">
        <template v-for="item in menu" :key="item.id">
          <!-- Section title -->
          <li v-if="item.isTitle" class="menu-section-title">
            <span v-if="!sidebarCollapsed">{{ item.label }}</span>
            <span v-else class="section-divider"></span>
          </li>

          <!-- Item with subItems -->
          <li v-else-if="item.subItems" class="nav-item">
          <a
          class="nav-link"
          :class="{ active: isParentActive(item) }"
          href="#"
          :title="sidebarCollapsed ? item.label : ''"
          @click.prevent="toggleMenu(item.id)"
          >
          <i :class="item.icon"></i>
          <span class="nav-label">{{ item.label }}</span>
          <span
          v-if="getBadgeText(item) && !sidebarCollapsed"
          class="badge rounded-pill ms-auto badge-clearable"
          :class="`bg-${item.badge.variant}`"
          @click.stop="clearBadge(item)"
          :title="'Click to clear'"
          >{{ getBadgeText(item) }}</span>
          <i v-if="!sidebarCollapsed" class="mdi arrow" :class="openMenus.includes(item.id) ? 'mdi-chevron-down' : 'mdi-chevron-right'"></i>
          </a>
          <ul v-show="openMenus.includes(item.id) && !sidebarCollapsed" class="sub-nav">
          <li v-for="sub in item.subItems" :key="sub.id">
          <router-link class="nav-link" :to="sub.link" active-class="active">
          <span>{{ sub.label }}</span>
          </router-link>
          </li>
          </ul>
          </li>

          <!-- Simple link -->
          <li v-else class="nav-item">
          <router-link
          class="nav-link"
          :to="item.link"
          active-class="active"
          :title="sidebarCollapsed ? item.label : ''"
          >
          <i :class="item.icon"></i>
          <span class="nav-label">{{ item.label }}</span>
          <span
          v-if="getBadgeText(item) && !sidebarCollapsed"
          class="badge rounded-pill ms-auto badge-clearable"
          :class="`bg-${item.badge.variant}`"
          @click.stop="clearBadge(item)"
          :title="'Click to clear'"
          style="font-size:0.62rem;cursor:pointer"
          >{{ getBadgeText(item) }}</span>
          </router-link>
          </li>
        </template>
      </ul>
    </div>

    <!-- Footer: server info -->
    <div class="sidebar-footer">
      <div class="server-info-chip" :class="{ collapsed: sidebarCollapsed }">
        <i class="mdi mdi-server" style="font-size:0.9rem;flex-shrink:0"></i>
        <div v-if="!sidebarCollapsed">
          <div class="chip-label">Server</div>
          <div class="chip-value">{{ hostname }}</div>
        </div>
        <span class="status-dot online" :class="sidebarCollapsed ? '' : 'ms-auto'"></span>
      </div>
    </div>
  </div>
</template>

<script>
import { menuItems } from './menu'
import api from '@/services/api'
import config from '@/app.config.json'

export default {
  name: 'Sidebar',
  data() {
    return {
      menu: menuItems,
      openMenus: [],
      hostname: window.location.hostname || 'server',
      badgeCounts: {
        alerts: 0,
        security: 0,
        bans: 0
      },
      refreshTimer: null,
      abortController: null
    }
  },
  computed: {
    sidebarCollapsed() {
      return this.$store.state.layout.sidebarCollapsed
    }
  },
  async mounted() {
    this.abortController = new AbortController()
    await this.fetchBadgeCounts()
    // Refresh badge counts using configurable interval
    this.refreshTimer = setInterval(() => {
      this.fetchBadgeCounts()
    }, config.pollIntervalMs)
  },
  beforeUnmount() {
    if (this.refreshTimer) clearInterval(this.refreshTimer)
    if (this.abortController) {
      this.abortController.abort()
    }
  },
  methods: {
    async fetchBadgeCounts() {
      if (!this.$store.getters['auth/loggedIn']) return
      try {
        const api = (await import('@/services/api')).default
        // Fetch alerts count using lightweight count endpoint
        const alertsCountRes = await api.getAlertCount({ signal: this.abortController?.signal })
        this.badgeCounts.alerts = alertsCountRes.data?.count || 0
        // Fetch security stats
        const secRes = await api.getSecurityStatus({ signal: this.abortController?.signal })
        const sec = secRes.data || {}
        this.badgeCounts.security = sec.active_bans || 0
        this.badgeCounts.bans = sec.active_bans || 0
      } catch (err) {
        if (err.name !== 'AbortError' && err.response?.status !== 401) {
          console.error('Failed to fetch badge counts:', err)
        }
      }
    },
    clearBadge(item) {
      // Clear badge for specific menu item
      if (item.badge) {
        item.badge.text = null
        item.badge.cleared = true
      }
    },
    toggleCollapse() {
      this.$store.commit('layout/TOGGLE_COLLAPSED')
    },
    toggleMenu(id) {
      const idx = this.openMenus.indexOf(id)
      if (idx === -1) this.openMenus.push(id)
      else this.openMenus.splice(idx, 1)
    },
    isParentActive(item) {
      if (!item.subItems) return false
      return item.subItems.some(sub => this.$route.path.startsWith(sub.link))
    },
    getBadgeText(item) {
      // Return dynamic badge text based on item
      if (item.id === 13) { // Alerts
        return this.badgeCounts.alerts > 0 ? this.badgeCounts.alerts : null
      }
      if (item.id === 11) { // Security
        return this.badgeCounts.security > 0 ? this.badgeCounts.security : null
      }
      // Fallback to static badge
      return item.badge ? item.badge.text : null
    }
  },
  watch: {
    $route() {
      this.menu.forEach(item => {
        if (item.subItems) {
          const isActive = item.subItems.some(sub => this.$route.path.startsWith(sub.link))
          if (isActive && !this.openMenus.includes(item.id)) {
            this.openMenus.push(item.id)
          }
        }
      })
      // Clear badge when navigating to the corresponding page
      if (this.$route.path === '/alerts') {
        this.badgeCounts.alerts = 0
      }
      if (this.$route.path === '/security') {
        this.badgeCounts.security = 0
      }
    }
  }
}
</script>
