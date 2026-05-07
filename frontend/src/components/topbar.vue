<template>
  <div id="page-topbar" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <div class="topbar-left">
      <!-- Desktop collapse toggle -->
      <button class="topbar-btn d-none d-lg-flex" @click="toggleCollapse" :title="sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'" aria-label="Toggle sidebar" tabindex="0">
        <i class="mdi mdi-menu" style="font-size:1.1rem" aria-hidden="true"></i>
      </button>
      <!-- Mobile menu toggle -->
      <button class="topbar-btn d-lg-none" @click="$emit('toggle-sidebar')" aria-label="Toggle sidebar" tabindex="0">
        <i class="mdi mdi-menu" style="font-size:1.2rem" aria-hidden="true"></i>
      </button>

      <!-- Breadcrumb -->
      <nav aria-label="breadcrumb" class="d-none d-md-flex align-items-center">
        <span class="text-muted" style="font-size:0.8rem">
          <i class="mdi mdi-slash-forward" style="font-size:0.7rem;opacity:0.4"></i>
          {{ currentPage }}
        </span>
      </nav>
    </div>

    <div class="topbar-right">
      <!-- WS connection status -->
      <div class="ws-status me-2" :class="{ disconnected: !wsConnected }">
        <div class="dot"></div>
        <span class="d-none d-md-inline">{{ wsConnected ? 'Live' : 'Offline' }}</span>
      </div>

      <!-- Theme switcher -->
      <div class="position-relative" v-click-outside="() => showThemeMenu = false">
        <button class="topbar-btn" @click="showThemeMenu = !showThemeMenu" title="Switch theme" data-dropdown-trigger aria-label="Switch theme" tabindex="0">
          <i class="mdi" :class="themeIcon" style="font-size:1.1rem" aria-hidden="true"></i>
        </button>
        <div v-show="showThemeMenu" class="dropdown-menu dropdown-menu-end show" style="min-width:160px;top:44px;right:0;position:absolute">
          <a
            v-for="t in themeOptions"
            :key="t.value"
            href="#"
            class="dropdown-item d-flex align-items-center gap-2"
            :class="{ active: currentThemePref === t.value }"
            @click.prevent="setTheme(t.value)"
            :aria-label="`Switch to ${t.label} theme`"
          >
            <i :class="t.icon" style="font-size:1rem;width:18px" aria-hidden="true"></i>
            <span>{{ t.label }}</span>
            <i v-if="currentThemePref === t.value" class="mdi mdi-check ms-auto" style="color:#4a9eff" aria-hidden="true"></i>
          </a>
        </div>
      </div>

      <!-- Lock screen button -->
      <button
        v-if="lockPinSet"
        class="topbar-btn"
        title="Lock screen (Space)"
        aria-label="Lock screen"
        tabindex="0"
        @click="lockScreen"
      >
        <i class="mdi mdi-lock-outline" style="font-size:1.1rem"></i>
      </button>

      <!-- Quick Mount -->
      <div class="position-relative" v-click-outside="() => showQuickMount = false">
        <button
          class="topbar-btn d-none d-md-flex align-items-center justify-content-center"
          style="width:36px;height:36px;border-radius:8px;background:rgba(34,214,124,0.06);border:1px solid rgba(34,214,124,0.18);color:#22d67c"
          @click="toggleQuickMount"
          title="Quick Mount — SSH port forward a server app to your machine"
          aria-label="Quick Mount"
          data-dropdown-trigger
        >
          <i class="mdi mdi-lan-connect" style="font-size:1rem"></i>
        </button>

        <div v-show="showQuickMount" class="dropdown-menu dropdown-menu-end show quick-mount-panel" style="top:44px;right:0;position:absolute;min-width:420px;max-width:96vw;padding:0">
          <!-- Header -->
          <div class="d-flex align-items-center justify-content-between px-3 py-2" style="border-bottom:1px solid var(--sc-border, #1e2d4a)">
            <span style="font-weight:600;font-size:0.82rem;display:flex;align-items:center;gap:6px">
              <i class="mdi mdi-lan-connect" style="color:#22d67c"></i> Quick Mount
            </span>
            <div class="d-flex align-items-center gap-2">
              <button class="btn btn-sm p-0" style="color:#4a9eff;font-size:0.72rem" @click="openQuickMountHelp" title="How Quick Mount works">
                <i class="mdi mdi-help-circle-outline"></i>
              </button>
              <button class="btn btn-sm p-0" style="color:#5a7499;font-size:0.72rem" @click="refreshTunnelApps" :disabled="loadingTunnels">
                <i :class="`mdi mdi-refresh${loadingTunnels ? ' mdi-spin' : ''}`"></i>
              </button>
              <button class="btn btn-sm p-0" style="color:#5a7499" @click="showQuickMount=false"><i class="mdi mdi-close"></i></button>
            </div>
          </div>

          <!-- SSH connection settings -->
          <div class="px-3 py-2" style="background:rgba(74,158,255,0.04);border-bottom:1px solid var(--sc-border, #1e2d4a);font-size:0.72rem">
            <div class="d-flex align-items-center gap-2 flex-wrap">
              <span style="color:#5a7499;white-space:nowrap">SSH user</span>
              <input v-model="mountSshUser" type="text" class="form-control form-control-sm" style="width:90px;height:24px;font-size:0.72rem;font-family:monospace;padding:1px 6px" :placeholder="detectedSshUser || 'user'" />
              <span style="color:#5a7499;white-space:nowrap">port</span>
              <input v-model="mountSshPort" type="text" class="form-control form-control-sm" style="width:55px;height:24px;font-size:0.72rem;font-family:monospace;padding:1px 6px" :placeholder="String(detectedSshPort || 22)" />
              <span style="color:#5a7499;white-space:nowrap">host</span>
              <input v-model="mountSshHost" type="text" class="form-control form-control-sm" style="width:130px;height:24px;font-size:0.72rem;font-family:monospace;padding:1px 6px" :placeholder="detectedSshHost || serverHost" />
            </div>
            <div class="mt-1" style="font-size:0.65rem;color:#5a7499">
              Auto-detected: <code style="color:#8aa4c8">{{ effectiveSshUser }}</code>@<code style="color:#8aa4c8">{{ effectiveSshHost }}</code> -p <code style="color:#8aa4c8">{{ effectiveSshPort }}</code>
              <span v-if="detectedSshUserSource" style="margin-left:6px;opacity:0.85">({{ detectedSshUserSource }})</span>
            </div>
          </div>

          <!-- App list -->
          <div style="max-height:360px;overflow-y:auto;padding:8px 0">
            <div v-if="loadingTunnels" class="text-center py-4" style="font-size:0.78rem;color:#5a7499">
              <span class="spinner-border spinner-border-sm me-1"></span>Detecting apps…
            </div>
            <div v-else-if="!tunnelApps.length" class="text-center py-4 px-3" style="font-size:0.78rem;color:#5a7499">
              <i class="mdi mdi-lan-disconnect d-block mb-1" style="font-size:1.4rem;opacity:0.4"></i>
              No web-accessible apps detected.<br>
              <span style="font-size:0.68rem">Grafana, Portainer, Prometheus, Netdata etc. will appear here when running.</span>
            </div>
            <template v-else>
              <!-- Group by category -->
              <div v-for="cat in mountCategories" :key="cat">
                <div class="px-3 py-1" style="font-size:0.65rem;font-weight:600;color:#5a7499;text-transform:uppercase;letter-spacing:.06em">{{ cat }}</div>
                <div
                  v-for="app in tunnelApps.filter(a => a.category === cat)"
                  :key="`${app.name}-${app.port}`"
                  class="d-flex align-items-center justify-content-between px-3 py-2 quick-mount-row"
                >
                  <div class="d-flex align-items-center gap-2" style="min-width:0">
                    <i :class="`mdi ${app.icon}`" :style="`color:${app.color};font-size:1.05rem;flex-shrink:0`"></i>
                    <div style="min-width:0">
                      <div style="font-size:0.78rem;font-weight:500;color:var(--sc-text, #c9d8f0)">{{ app.name }}</div>
                      <code style="font-size:0.65rem;color:#5a7499">localhost:{{ app.port }} → server:{{ app.port }}</code>
                    </div>
                    <span class="badge ms-1" :style="`background:rgba(${app.source==='docker'?'36,150,237':'34,214,124'},0.1);color:${app.source==='docker'?'#2496ed':'#22d67c'};font-size:0.58rem;padding:1px 5px`">{{ app.source }}</span>
                  </div>
                  <div class="d-flex align-items-center gap-1 ms-2">
                    <button
                      class="btn btn-sm flex-shrink-0"
                      style="background:rgba(245,166,35,0.08);color:#f5a623;border:1px solid rgba(245,166,35,0.2);font-size:0.65rem;white-space:nowrap;padding:2px 8px"
                      :disabled="grantingPort === app.port"
                      @click="grantAccess(app)"
                      title="Grant browser access for your IP (choose duration)"
                    >
                      <i :class="`mdi mdi-${grantingPort===app.port?'loading mdi-spin':'shield-key-outline'} me-1`"></i>Grant Access
                    </button>
                    <button
                      class="btn btn-sm flex-shrink-0"
                      :style="`background:rgba(34,214,124,0.07);color:${mountCopied===app.name+app.port?'#22d67c':'#5a7499'};border:1px solid rgba(34,214,124,0.15);font-size:0.65rem;white-space:nowrap;padding:2px 8px`"
                      @click="copyMount(app)"
                    >
                      <i :class="`mdi mdi-${mountCopied===app.name+app.port?'check':'content-copy'} me-1`"></i>{{ mountCopied === app.name+app.port ? 'Copied!' : 'Copy' }}
                    </button>
                  </div>
                </div>
              </div>
            </template>
          </div>

          <!-- Footer hint -->
          <div class="px-3 py-2" style="border-top:1px solid var(--sc-border, #1e2d4a);font-size:0.67rem;color:#5a7499">
            <i class="mdi mdi-information-outline me-1"></i>Run the copied command on <strong style="color:#8aa4c8">your local machine</strong>, then open <code style="color:#4a9eff">http://localhost:&lt;port&gt;</code> in your browser.
          </div>
        </div>
      </div>

      <!-- Global Search -->
      <div class="position-relative me-2 d-none d-md-block">
        <div class="search-input-wrapper">
          <i class="mdi mdi-magnify search-icon"></i>
          <input
            v-model="searchQuery"
            type="text"
            class="search-input"
            placeholder="Search..."
            @focus="showSearchResults = true"
            @keydown.esc="showSearchResults = false; searchQuery = ''"
            @keydown.down="navigateSearch(1)"
            @keydown.up="navigateSearch(-1)"
            @keydown.enter="selectSearchResult"
          />
        </div>
        <div v-show="showSearchResults && searchResults.length > 0" class="search-results-dropdown">
          <div class="search-results-header">
            <span>Quick Navigation</span>
          </div>
          <div v-for="(result, idx) in searchResults" :key="idx"
               class="search-result-item"
               :class="{ active: searchActiveIndex === idx }"
               @click="navigateToSearchResult(result)"
               @mouseenter="searchActiveIndex = idx">
            <i :class="result.icon"></i>
            <span>{{ result.label }}</span>
            <span class="search-result-path">{{ result.path }}</span>
          </div>
          <div v-if="searchQuery && searchResults.length === 0" class="search-no-results">
            No results found for "{{ searchQuery }}"
          </div>
        </div>
      </div>

      <!-- Notifications -->
      <div class="position-relative" v-click-outside="() => showNotifs = false">
        <button class="topbar-btn" @click="toggleNotifs" data-dropdown-trigger aria-label="Notifications" tabindex="0">
          <i class="mdi mdi-bell-outline" style="font-size:1.1rem" aria-hidden="true"></i>
          <span v-if="unreadCount > 0" class="badge-dot"></span>
        </button>

        <div v-show="showNotifs" class="dropdown-menu dropdown-menu-end show alert-dropdown">
          <div class="d-flex align-items-center justify-content-between px-3 py-2">
            <span style="font-weight:600;font-size:0.8rem">Alerts</span>
            <div class="d-flex gap-2">
              <button v-if="unreadCount > 0" class="btn btn-sm btn-link p-0" style="font-size:0.72rem;color:#4a9eff" @click="markAllAsRead">
                <i class="mdi mdi-check-all me-1"></i>Mark all read
              </button>
              <span v-if="unreadCount > 0" class="badge bg-danger" style="font-size:0.65rem">{{ unreadCount }} new</span>
            </div>
          </div>
          <div class="dropdown-divider m-0"></div>
          <div v-if="loadingAlerts" class="text-center py-3" style="font-size:0.78rem;color:#5a7499">
            <span class="spinner-border spinner-border-sm me-1"></span>Loading…
          </div>
          <template v-else>
            <div
              v-for="alert in recentAlerts"
              :key="alert.id"
              class="dropdown-item d-flex align-items-start gap-2 py-2 alert-item"
              @click="markAlertAsRead(alert.id)"
            >
              <i :class="severityIcon(alert.severity)" :style="`color:${severityColor(alert.severity)};font-size:1rem;margin-top:2px`"></i>
              <div style="flex:1;min-width:0">
                <div style="font-size:0.78rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{{ alert.message }}</div>
                <div style="font-size:0.68rem;color:#5a7499">{{ timeAgo(alert.ts) }}</div>
              </div>
              <i v-if="!alert.read" class="mdi mdi-circle-small" style="color:#4a9eff;font-size:0.6rem"></i>
            </div>
            <div v-if="!recentAlerts.length" class="text-center py-3" style="font-size:0.78rem;color:#5a7499">
              No recent alerts
            </div>
          </template>
          <div class="dropdown-divider m-0"></div>
          <router-link to="/alerts" class="dropdown-item text-center py-2" style="font-size:0.78rem;color:#4a9eff" @click="showNotifs=false">
            View all alerts <i class="mdi mdi-arrow-right ms-1"></i>
          </router-link>
        </div>
      </div>

      <!-- User menu -->
      <div class="position-relative" v-click-outside="() => showUserMenu = false">
        <div class="user-menu" @click="showUserMenu = !showUserMenu" data-dropdown-trigger role="button" tabindex="0" aria-label="User menu">
          <div class="user-avatar">{{ userInitials }}</div>
          <span class="user-name d-none d-md-inline">{{ username }}</span>
          <i class="mdi mdi-chevron-down d-none d-md-inline" style="font-size:0.8rem;color:#5a7499" aria-hidden="true"></i>
        </div>

        <div v-show="showUserMenu" class="dropdown-menu dropdown-menu-end show" style="top:44px;right:0;position:absolute;min-width:200px">
          <div class="px-3 py-2" style="border-bottom:1px solid var(--sc-border, #1e2d4a)">
            <div style="font-size:0.82rem;font-weight:600">{{ username }}</div>
            <div style="font-size:0.72rem;color:#5a7499">{{ userRole }}</div>
          </div>
          <router-link to="/settings" class="dropdown-item" @click="showUserMenu=false">
            <i class="mdi mdi-cog-outline"></i> Settings
          </router-link>
          <router-link to="/audit-logs" class="dropdown-item" @click="showUserMenu=false">
            <i class="mdi mdi-history"></i> Audit Logs
          </router-link>
          <a href="#" class="dropdown-item" @click.prevent="lockScreen">
            <i class="mdi mdi-lock-outline"></i> Lock Screen
          </a>
          <div class="dropdown-divider"></div>
          <a href="#" class="dropdown-item" style="color:#f04040" @click.prevent="logout">
            <i class="mdi mdi-logout" style="color:#f04040"></i> Logout
          </a>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showQuickMountHelp" class="quick-mount-help-backdrop" @click.self="showQuickMountHelp = false">
        <div class="quick-mount-help-modal">
        <div class="quick-mount-help-header">
          <h6 class="mb-0 d-flex align-items-center gap-2">
            <i class="mdi mdi-help-circle-outline" style="color:#4a9eff"></i>
            How To Use Quick Mount
          </h6>
          <button class="btn btn-sm p-0" style="color:#5a7499" @click="showQuickMountHelp = false">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div class="quick-mount-help-body">
          <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:10px">
            Quick Mount creates an SSH local tunnel so apps running on your server become reachable from your computer.
          </p>
          <ol class="quick-mount-help-steps">
            <li>
              Set your SSH connection values in Quick Mount:
              <code>user</code>, <code>port</code>, and <code>host</code>. These are auto-detected and prefilled.
            </li>
            <li>
              Click <strong>Copy</strong> next to any app. You will get a command like:
              <code>{{ mountExampleCommand }}</code>
            </li>
            <li>
              Paste and run that command in your local terminal (your own PC/macOS/Linux machine).
            </li>
            <li>
              Keep that SSH session open, then visit:
              <code>http://localhost:&lt;port&gt;</code>
            </li>
          </ol>
          <div class="quick-mount-help-note">
            <i class="mdi mdi-shield-lock-outline"></i>
            <span>The tunnel is encrypted over SSH and closes automatically when you end the SSH session.</span>
          </div>
          <div class="quick-mount-help-example">
            <div style="font-size:0.7rem;color:#5a7499;margin-bottom:5px">Example command</div>
            <code>{{ mountExampleCommand }}</code>
          </div>
        </div>
        <div class="quick-mount-help-footer">
          <button class="btn btn-sm" style="background:rgba(34,214,124,0.08);color:#22d67c;border:1px solid rgba(34,214,124,0.2)" @click="copyMountExample">
            <i :class="`mdi mdi-${mountHelpCopied ? 'check' : 'content-copy'} me-1`"></i>{{ mountHelpCopied ? 'Copied' : 'Copy Example Command' }}
          </button>
          <button class="btn btn-sm btn-sc-primary" @click="showQuickMountHelp = false">
            Got it
          </button>
        </div>
      </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import api from '@/services/api'

export default {
  name: 'Topbar',
  emits: ['toggle-sidebar'],
  data() {
    return {
      searchQuery: '',
      showSearchResults: false,
      searchActiveIndex: -1,
      showNotifs: false,
      showUserMenu: false,
      showThemeMenu: false,
      themeOptions: [
        { value: 'system', label: 'System', icon: 'mdi mdi-monitor' },
        { value: 'light', label: 'Light', icon: 'mdi mdi-weather-sunny' },
        { value: 'dark', label: 'Dark', icon: 'mdi mdi-weather-night' }
      ],
      recentAlerts: [],
      loadingAlerts: false,
      abortController: null,
      handleKeyboardShortcuts: null,
      handleGlobalClick: null,
      // Quick Mount
      showQuickMount: false,
      showQuickMountHelp: false,
      mountHelpCopied: false,
      tunnelApps: [],
      loadingTunnels: false,
      mountCopied: '',
      grantingPort: null,
      mountClientIp: '',
      mountSshUser: '',
      mountSshPort: '22',
      mountSshHost: '',
      detectedSshUser: '',
      detectedSshUserSource: '',
      detectedSshPort: 22,
      detectedSshHost: '',
    }
  },
  computed: {
    wsConnected() {
      return this.$store.getters['metrics/wsConnected']
    },
    sidebarCollapsed() {
      return this.$store.state.layout.sidebarCollapsed
    },
    currentThemePref() {
      return this.$store.state.layout.theme
    },
    themeIcon() {
      const t = this.currentThemePref
      if (t === 'light') return 'mdi-weather-sunny'
      if (t === 'dark') return 'mdi-weather-night'
      return 'mdi-monitor'
    },
    currentPage() {
      const names = {
        '/dashboard': 'Dashboard', '/security': 'Security Center',
        '/firewall': 'Firewall', '/monitoring': 'Monitoring',
        '/containers': 'Containers', '/logs': 'Logs',
        '/tasks': 'Tasks', '/alerts': 'Alerts',
        '/users': 'Users', '/settings': 'Settings',
        '/terminal': 'Terminal', '/audit-logs': 'Audit Logs',
        '/security-tools': 'Security Tools',
        '/updates': 'Updates'
      }
      return names[this.$route.path] || 'SentinelCore'
    },
    username() {
      try { return JSON.parse(sessionStorage.getItem('sc_user') || '{}').username || 'admin' } catch (e) { console.error('Failed to parse user from sessionStorage:', e); return 'admin' }
    },
    userRole() {
      try { return JSON.parse(sessionStorage.getItem('sc_user') || '{}').role || '' } catch (e) { console.error('Failed to parse role from sessionStorage:', e); return '' }
    },
    userInitials() {
      return this.username.slice(0, 2).toUpperCase()
    },
    lockPinSet() {
      return !!localStorage.getItem('sc_lock_pin_hash')
    },
    unreadCount() {
      return this.recentAlerts.filter(a => !a.read).length
    },
    serverHost() {
      return window.location.hostname
    },
    mountCategories() {
      return [...new Set(this.tunnelApps.map(a => a.category))]
    },
    effectiveSshUser() {
      return this.mountSshUser || this.detectedSshUser || 'user'
    },
    effectiveSshHost() {
      return this.mountSshHost || this.detectedSshHost || this.serverHost
    },
    effectiveSshPort() {
      const p = Number.parseInt(this.mountSshPort, 10)
      if (!Number.isNaN(p) && p > 0 && p <= 65535) return p
      return this.detectedSshPort || 22
    },
    mountExampleCommand() {
      const host = this.effectiveSshHost
      const user = this.effectiveSshUser
      const port = String(this.effectiveSshPort)
      const portFlag = port !== '22' ? ` -p ${port}` : ''
      return `ssh -L 9090:localhost:9090${portFlag} ${user}@${host}`
    },
    searchResults() {
      if (!this.searchQuery || this.searchQuery.trim() === '') return []
      const query = this.searchQuery.toLowerCase()
      const results = []
      
      // Define searchable pages with icons
      const pages = [
        { path: '/dashboard', label: 'Dashboard', icon: 'mdi mdi-view-dashboard' },
        { path: '/security', label: 'Security Center', icon: 'mdi mdi-shield' },
        { path: '/firewall', label: 'Firewall', icon: 'mdi mdi-firewall' },
        { path: '/monitoring', label: 'Monitoring', icon: 'mdi mdi-chart-line' },
        { path: '/containers', label: 'Containers', icon: 'mdi mdi-box' },
        { path: '/logs', label: 'Logs', icon: 'mdi mdi-file-document' },
        { path: '/tasks', label: 'Tasks', icon: 'mdi mdi-clock-outline' },
        { path: '/alerts', label: 'Alerts', icon: 'mdi mdi-bell' },
        { path: '/users', label: 'Users', icon: 'mdi mdi-account-group' },
        { path: '/settings', label: 'Settings', icon: 'mdi mdi-cog' },
        { path: '/terminal', label: 'Terminal', icon: 'mdi mdi-terminal' },
        { path: '/audit-logs', label: 'Audit Logs', icon: 'mdi mdi-history' },
        { path: '/updates', label: 'Updates', icon: 'mdi mdi-download' },
        { path: '/security-tools', label: 'Security Tools', icon: 'mdi mdi-shield-search' },
        { path: '/services', label: 'Services', icon: 'mdi mdi-server' }
      ]
      
      // Filter pages by query
      for (const page of pages) {
        if (page.label.toLowerCase().includes(query) || page.path.toLowerCase().includes(query)) {
          results.push({ ...page, path: page.path })
        }
      }
      
      return results.slice(0, 8)
    }
  },
  methods: {
    toggleCollapse() {
      this.$store.commit('layout/TOGGLE_COLLAPSED')
    },
    setTheme(val) {
      this.$store.commit('layout/SET_THEME', val)
      this.showThemeMenu = false
    },
    lockScreen() {
      this.showUserMenu = false
      this.showThemeMenu = false
      if (!this.lockPinSet) {
        this.$router.push('/settings?tab=lock')
        return
      }
      window.dispatchEvent(new CustomEvent('sentinel:lock'))
    },
    focusSearch() {
      this.showSearchResults = true
      this.$nextTick(() => {
        const el = this.$el?.querySelector('.search-input')
        el?.focus()
      })
    },
    async toggleNotifs() {
      this.showNotifs = !this.showNotifs
      if (this.showNotifs && !this.recentAlerts.length) {
        await this.loadAlerts()
      }
    },
    async loadAlerts() {
      if (!this.$store.getters['auth/loggedIn']) return
      this.loadingAlerts = true
      try {
        const api = (await import('@/services/api')).default
        const { data } = await api.getAlerts({ signal: this.abortController?.signal })
        this.recentAlerts = (Array.isArray(data) ? data : []).slice(0, 5)
      } catch (err) {
        if (err.name !== 'AbortError' && err.response?.status !== 401) {
          console.error('Failed to load alerts:', err)
        }
      } finally {
        this.loadingAlerts = false
      }
    },
    severityIcon(s) {
      if (s === 'critical') return 'mdi mdi-shield-alert'
      if (s === 'warning') return 'mdi mdi-alert-circle-outline'
      return 'mdi mdi-information-outline'
    },
    severityColor(s) {
      if (s === 'critical') return '#f04040'
      if (s === 'warning') return '#f5a623'
      return '#4a9eff'
    },
    timeAgo(ts) {
      const diff = Math.floor(Date.now() / 1000) - ts
      if (diff < 60) return `${diff}s ago`
      if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
      return `${Math.floor(diff / 86400)}d ago`
    },
    logout() {
      sessionStorage.removeItem('sc_user')
      sessionStorage.removeItem('sc_token')
      this.$router.push('/login')
    },
    navigateSearch(direction) {
      if (this.searchResults.length === 0) return
      this.searchActiveIndex += direction
      if (this.searchActiveIndex < 0) this.searchActiveIndex = this.searchResults.length - 1
      if (this.searchActiveIndex >= this.searchResults.length) this.searchActiveIndex = 0
    },
    selectSearchResult() {
      if (this.searchActiveIndex >= 0 && this.searchResults[this.searchActiveIndex]) {
        this.navigateToSearchResult(this.searchResults[this.searchActiveIndex])
      }
    },
    navigateToSearchResult(result) {
      this.$router.push(result.path)
      this.searchQuery = ''
      this.showSearchResults = false
      this.searchActiveIndex = -1
    },
    async markAllAsRead() {
      try {
        await api.markAlertsAsRead(this.recentAlerts.map(a => a.id))
        this.recentAlerts = this.recentAlerts.map(a => ({ ...a, read: true }))
      } catch (err) {
        console.error('Failed to mark alerts as read:', err)
      }
    },
    async markAlertAsRead(id) {
      try {
        await api.markAlertAsRead(id)
        this.recentAlerts = this.recentAlerts.map(a =>
          a.id === id ? { ...a, read: true } : a
        )
      } catch (err) {
        console.error('Failed to mark alert as read:', err)
      }
    },
    async toggleQuickMount() {
      this.showQuickMount = !this.showQuickMount
      if (this.showQuickMount && !this.tunnelApps.length) {
        await this.refreshTunnelApps()
      }
    },
    openQuickMountHelp() {
      this.showQuickMountHelp = true
    },
    copyMountExample() {
      navigator.clipboard.writeText(this.mountExampleCommand).catch(() => {})
      this.mountHelpCopied = true
      setTimeout(() => { this.mountHelpCopied = false }, 1800)
    },
    async loadMountClientIp() {
      try {
        const { data } = await api.getMe()
        this.mountClientIp = data?.client_ip || ''
      } catch (_) {
        this.mountClientIp = ''
      }
    },
    async refreshTunnelApps() {
      this.loadingTunnels = true
      try {
        const { data } = await api.getTunnelableApps()
        if (Array.isArray(data)) {
          this.tunnelApps = data
          return
        }
        this.tunnelApps = Array.isArray(data?.apps) ? data.apps : []
        this.detectedSshHost = data?.connection?.host || this.detectedSshHost
        this.detectedSshPort = Number.parseInt(data?.connection?.ssh_port, 10) || this.detectedSshPort || 22
        this.detectedSshUser = data?.connection?.ssh_user || this.detectedSshUser
        this.detectedSshUserSource = data?.connection?.ssh_user_source || this.detectedSshUserSource
      } catch (_) {
        this.tunnelApps = []
      } finally {
        this.loadingTunnels = false
      }
    },
    copyMount(app) {
      const host = this.effectiveSshHost
      const user = this.effectiveSshUser
      const port = String(this.effectiveSshPort)
      const portFlag = port !== '22' ? ` -p ${port}` : ''
      const cmd = `ssh -L ${app.port}:localhost:${app.port}${portFlag} ${user}@${host}`
      navigator.clipboard.writeText(cmd).catch(() => {})
      const key = app.name + app.port
      this.mountCopied = key
      setTimeout(() => { if (this.mountCopied === key) this.mountCopied = '' }, 2000)
    },
    async grantAccess(app) {
      const ip = this.mountClientIp || 'your current IP'
      let durationHours = 3

      if (this.$swal) {
        const durationSelect = `
          <div style="margin-top:10px;text-align:left">
            <p style="margin-bottom:6px;color:#c9d8f0;font-size:0.87rem">
              Allow <strong>${app.name}</strong> (port ${app.port}) to be accessed directly from <strong>${ip}</strong>.
            </p>
            <label style="font-size:0.8rem;color:#8fa8c8;display:block;margin-bottom:4px">Duration:</label>
            <select id="sc-grant-duration" style="width:100%;padding:6px 10px;border-radius:6px;border:1px solid rgba(100,140,200,0.3);background:#0e1c30;color:#c9d8f0;font-size:0.85rem">
              <option value="1">1 hour</option>
              <option value="3" selected>3 hours</option>
              <option value="6">6 hours</option>
              <option value="12">12 hours</option>
              <option value="24">24 hours</option>
            </select>
          </div>`
        const res = await this.$swal({
          title: 'Grant Temporary Access?',
          html: durationSelect,
          icon: 'warning',
          showCancelButton: true,
          confirmButtonText: 'Grant Access',
          cancelButtonText: 'Cancel',
          confirmButtonColor: '#f5a623',
          preConfirm: () => parseInt(document.getElementById('sc-grant-duration')?.value) || 3,
        })
        if (!res.isConfirmed) return
        durationHours = res.value
      } else {
        const text = `Allow browser access to ${app.name} on port ${app.port} for IP ${ip}?`
        if (!window.confirm(text)) return
      }

      this.grantingPort = app.port
      try {
        const { data } = await api.grantTunnelAccess(app.port, durationHours)
        const grantedIp = data?.ip || this.mountClientIp || 'your IP'
        const dh = data?.duration_hours || durationHours
        const expiresAt = data?.expires_at ? new Date(data.expires_at * 1000).toLocaleString() : `in ${dh}h`
        const browseHost = this.effectiveSshHost
        const browseUrl = `http://${browseHost}:${app.port}`
        const isProxy = data?.mode === 'nat' || data?.mode === 'proxy'
        const modeNote = isProxy
          ? `<p style="margin-top:8px;padding:6px 10px;border-radius:6px;background:rgba(34,214,124,0.08);color:#22d67c;font-size:0.78rem"><i class="mdi mdi-router-network me-1"></i>Proxy mode: SentinelCore is forwarding ${browseHost}:${app.port} → 127.0.0.1:${app.port}</p>`
          : `<p style="margin-top:8px;padding:6px 10px;border-radius:6px;background:rgba(74,158,255,0.08);color:#4a9eff;font-size:0.78rem"><i class="mdi mdi-shield-check me-1"></i>UFW rule added for ${grantedIp}</p>`
        if (this.$swal) {
          await this.$swal({
            title: 'Access Granted \u2713',
            html: `<div style="text-align:left;font-size:0.87rem;color:#c9d8f0">
              <p><strong>IP:</strong> ${grantedIp} &nbsp; <strong>Port:</strong> ${app.port} &nbsp; <strong>Duration:</strong> ${dh}h</p>
              <p><strong>Expires:</strong> ${expiresAt}</p>
              ${modeNote}
              <p style="margin-top:10px">Open: <a href="${browseUrl}" target="_blank" style="color:#4a9eff">${browseUrl}</a></p>
            </div>`,
            icon: 'success',
            confirmButtonText: 'OK'
          })
        } else {
          window.alert(`Access granted for ${grantedIp} to port ${app.port} for ${dh}h. Expires: ${expiresAt}.\n\nOpen: ${browseUrl}`)
        }
      } catch (err) {
        const msg = err?.response?.data?.error || err?.response?.data?.message || 'Failed to grant temporary access'
        if (this.$swal) {
          await this.$swal({ title: 'Grant Failed', text: msg, icon: 'error', confirmButtonText: 'OK' })
        } else {
          window.alert(msg)
        }
      } finally {
        this.grantingPort = null
      }
    },
    closeAllDropdowns() {
      this.showNotifs = false
      this.showUserMenu = false
      this.showThemeMenu = false
      this.showQuickMount = false
      this.showSearchResults = false
    }
  },
  mounted() {
    this.abortController = new AbortController()
    this.loadMountClientIp()

    // Add global keyboard shortcuts
    this.handleKeyboardShortcuts = (e) => {
      // Only handle shortcuts when not typing in input fields
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.contentEditable === 'true') {
        return
      }

      // Lock screen: Space or Ctrl+L
      if ((e.key === ' ' || (e.ctrlKey && e.key === 'l')) && this.lockPinSet) {
        e.preventDefault()
        this.lockScreen()
        return
      }

      // Search: Ctrl+K or Ctrl+/
      if ((e.ctrlKey && (e.key === 'k' || e.key === '/'))) {
        e.preventDefault()
        this.focusSearch()
        return
      }

      // Close dropdowns: Escape
      if (e.key === 'Escape') {
        if (this.showQuickMountHelp) {
          this.showQuickMountHelp = false
          return
        }
        this.closeAllDropdowns()
        return
      }
    }

    document.addEventListener('keydown', this.handleKeyboardShortcuts)

    // Close dropdowns when clicking outside
    this.handleGlobalClick = (e) => {
      const clickedElement = e.target
      const isInsideDropdown = clickedElement.closest('.dropdown-menu')
      const isInsideDropdownTrigger = clickedElement.closest('[data-dropdown-trigger]')

      if (!isInsideDropdown && !isInsideDropdownTrigger) {
        this.closeAllDropdowns()
      }
    }

    document.addEventListener('click', this.handleGlobalClick, true)
  },
  beforeUnmount() {
    if (this.abortController) {
      this.abortController.abort()
    }
    document.removeEventListener('keydown', this.handleKeyboardShortcuts)
    document.removeEventListener('click', this.handleGlobalClick, true)
  }
}
</script>

<style scoped>
/* Global Search Styles */
.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--sc-text-muted);
  font-size: 0.9rem;
  pointer-events: none;
  z-index: 5;
}

.search-input {
  width: 200px;
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--sc-border);
  border-radius: 6px;
  background: var(--sc-bg-secondary);
  color: var(--sc-text);
  font-size: 0.85rem;
  transition: width 0.3s ease, box-shadow 0.3s ease;
}

.search-input:focus {
  width: 260px;
  outline: none;
  border-color: var(--sc-blue);
  box-shadow: 0 0 0 3px rgba(74, 158, 255, 0.15);
}

.search-input::placeholder {
  color: var(--sc-text-muted);
}

.search-results-dropdown {
  position: absolute;
  top: 54px;
  right: 0;
  width: 320px;
  max-height: 400px;
  overflow-y: auto;
  background: var(--sc-bg-card);
  border: 1px solid var(--sc-border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  z-index: 1050;
}

.search-results-header {
  padding: 10px 14px;
  border-bottom: 1px solid var(--sc-border);
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--sc-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.search-result-item {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  cursor: pointer;
  transition: background 0.15s ease;
  color: var(--sc-text);
  font-size: 0.85rem;
}

.search-result-item:hover,
.search-result-item.active {
  background: rgba(74, 158, 255, 0.1);
}

.search-result-item i {
  width: 20px;
  margin-right: 10px;
  color: var(--sc-blue);
  font-size: 1rem;
}

.search-result-item span:first-child {
  flex: 1;
  font-weight: 500;
}

.search-result-path {
  font-size: 0.72rem;
  color: var(--sc-text-muted);
  opacity: 0.7;
}

.search-no-results {
  padding: 20px 14px;
  text-align: center;
  color: var(--sc-text-muted);
  font-size: 0.85rem;
}

/* Alert Dropdown Styles */
.alert-dropdown {
  position: absolute;
  top: 44px;
  right: 0;
  min-width: 280px;
  max-width: 320px;
  max-height: 400px;
  overflow-y: auto;
  background: var(--sc-bg-card);
  border: 1px solid var(--sc-border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  z-index: 1050;
}

.alert-item {
  cursor: pointer;
  transition: background 0.15s ease;
  border-radius: 4px;
  margin: 2px 4px;
}

.alert-item:hover {
  background: rgba(74, 158, 255, 0.08);
}

.alert-item i.mdi-circle-small {
  color: var(--sc-blue);
}

/* Scrollbar styling for search results and alerts */
.search-results-dropdown::-webkit-scrollbar,
.alert-dropdown::-webkit-scrollbar {
  width: 6px;
}

.search-results-dropdown::-webkit-scrollbar-track,
.alert-dropdown::-webkit-scrollbar-track {
  background: var(--sc-bg-secondary);
  border-radius: 3px;
}

.search-results-dropdown::-webkit-scrollbar-thumb,
.alert-dropdown::-webkit-scrollbar-thumb {
  background: var(--sc-border);
  border-radius: 3px;
}

.search-results-dropdown::-webkit-scrollbar-thumb:hover,
.alert-dropdown::-webkit-scrollbar-thumb:hover {
  background: var(--sc-text-muted);
}

.quick-mount-panel {
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.35);
}

.quick-mount-row {
  cursor: default;
  transition: background 0.12s;
}
.quick-mount-row:hover {
  background: rgba(74, 158, 255, 0.04);
}

.quick-mount-panel::-webkit-scrollbar {
  width: 4px;
}
.quick-mount-panel::-webkit-scrollbar-track {
  background: transparent;
}
.quick-mount-panel::-webkit-scrollbar-thumb {
  background: var(--sc-border);
  border-radius: 2px;
}

.quick-mount-help-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(3, 8, 18, 0.62);
  backdrop-filter: blur(2px);
}

.quick-mount-help-modal {
  width: min(560px, 92vw);
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 12px;
  background: var(--sc-bg-card);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.38);
}

.quick-mount-help-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--sc-border, #1e2d4a);
}

.quick-mount-help-body {
  padding: 12px 14px;
}

.quick-mount-help-steps {
  margin: 0;
  padding-left: 18px;
  color: var(--sc-text, #c9d8f0);
  font-size: 0.79rem;
}

.quick-mount-help-steps li {
  margin-bottom: 7px;
}

.quick-mount-help-steps code {
  color: #4a9eff;
  background: rgba(74, 158, 255, 0.08);
  padding: 1px 5px;
  border-radius: 4px;
}

.quick-mount-help-note {
  margin-top: 10px;
  display: flex;
  gap: 8px;
  align-items: flex-start;
  font-size: 0.75rem;
  color: #8aa4c8;
  background: rgba(34, 214, 124, 0.08);
  border: 1px solid rgba(34, 214, 124, 0.2);
  border-radius: 8px;
  padding: 8px 10px;
}

.quick-mount-help-note i {
  color: #22d67c;
  font-size: 0.95rem;
  margin-top: 1px;
}

.quick-mount-help-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding: 0 14px 12px;
}

.quick-mount-help-example {
  margin-top: 10px;
  border: 1px solid rgba(74, 158, 255, 0.18);
  border-radius: 8px;
  background: rgba(74, 158, 255, 0.05);
  padding: 8px 10px;
}

.quick-mount-help-example code {
  display: block;
  font-size: 0.72rem;
  color: #4a9eff;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
