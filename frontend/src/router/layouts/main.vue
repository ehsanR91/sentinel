<template>
  <div class="wrapper" :class="wrapperClasses">
    <Sidebar />
    <!-- Mobile backdrop -->
    <div class="sidebar-backdrop" :class="{ active: sidebarOpen && isMobileViewport }" @click="closeSidebar"></div>
    <div class="page-content">
      <Topbar @toggle-sidebar="toggleSidebar" />
      <div class="content-page">
        <RouterView v-slot="{ Component }">
          <Transition name="page" mode="out-in">
            <KeepAlive :include="['DashboardPage', 'MonitoringPage', 'SecurityPage', 'FirewallPage']" :max="4">
              <component :is="Component" />
            </KeepAlive>
          </Transition>
        </RouterView>
      </div>
      <Footer />
    </div>
  </div>
</template>

<script>
import Sidebar from '@/components/sidebar.vue'
import Topbar  from '@/components/topbar.vue'
import Footer  from '@/components/footer.vue'
import { useLayoutStore } from '@/stores/layout'

export default {
  name: 'MainLayout',
  setup() {
    return {
      layoutStore: useLayoutStore()
    }
  },
  components: { Sidebar, Topbar, Footer },
  data() {
    return {
      viewportWidth: typeof window === 'undefined' ? 1440 : window.innerWidth
    }
  },
  computed: {
    sidebarCollapsed() {
      return this.layoutStore.sidebarCollapsed
    },
    sidebarHidden() {
      return this.layoutStore.sidebarHidden
    },
    sidebarOpen() {
      return this.layoutStore.sidebarOpen
    },
    sidebarPosition() {
      return this.layoutStore.sidebarPosition
    },
    isMobileViewport() {
      return this.viewportWidth < 768
    },
    isCompactViewport() {
      return this.viewportWidth < 1100 && !this.isMobileViewport
    },
    effectiveSidebarCollapsed() {
      return !this.isMobileViewport && (this.sidebarCollapsed || this.isCompactViewport)
    },
    wrapperClasses() {
      return {
        'sidebar-collapsed': this.effectiveSidebarCollapsed,
        'sidebar-open': this.sidebarOpen && this.isMobileViewport,
        'sidebar-hidden': this.sidebarHidden,
        'sidebar-right': this.sidebarPosition === 'right'
      }
    }
  },
  watch: {
    $route() {
      if (this.isMobileViewport) {
        this.closeSidebar()
      }
    }
  },
  mounted() {
    window.addEventListener('resize', this.handleResize, { passive: true })
    this.handleResize()
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.handleResize)
  },
  methods: {
    handleResize() {
      this.viewportWidth = window.innerWidth
      if (!this.isMobileViewport) {
        this.closeSidebar()
      }
    },
    toggleSidebar() {
      this.layoutStore.toggleSidebar()
    },
    closeSidebar() {
      this.layoutStore.closeSidebar()
    }
  }
}
</script>

<style scoped>
.sidebar-backdrop {
  display: none;
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.sidebar-backdrop.active {
  display: block;
}

@media (min-width: 993px) {
  .sidebar-backdrop { display: none !important; }
}
</style>
