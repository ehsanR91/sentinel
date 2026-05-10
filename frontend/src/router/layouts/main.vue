<template>
  <div class="wrapper" :class="wrapperClasses">
    <Sidebar />
    <!-- Mobile backdrop: shows during open AND during edge-swipe drag -->
    <div
      class="sidebar-backdrop"
      :class="{ active: (sidebarOpen || edgeSwipe.active) && isMobileViewport }"
      :style="backdropStyle"
      @click="closeSidebar"
    ></div>
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
      viewportWidth: typeof window === 'undefined' ? 1440 : window.innerWidth,
      // Edge-swipe-to-open gesture state
      edgeSwipe: { active: false, startX: 0, currentX: 0, startTime: 0 }
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
    // Backdrop opacity during edge-swipe drag (0 → 1 as sidebar opens)
    backdropStyle() {
      const drag = this.layoutStore.swipeOpenDrag
      if (!this.sidebarOpen && drag !== null && this.isMobileViewport) {
        return { display: 'block', opacity: drag }
      }
      return {}
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
    document.addEventListener('touchstart', this._onDocTouchStart, { passive: true })
    document.addEventListener('touchmove',  this._onDocTouchMove,  { passive: true })
    document.addEventListener('touchend',   this._onDocTouchEnd,   { passive: true })
    document.addEventListener('touchcancel',this._onDocTouchCancel,{ passive: true })
    this.handleResize()
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.handleResize)
    document.removeEventListener('touchstart', this._onDocTouchStart)
    document.removeEventListener('touchmove',  this._onDocTouchMove)
    document.removeEventListener('touchend',   this._onDocTouchEnd)
    document.removeEventListener('touchcancel',this._onDocTouchCancel)
  },
  methods: {
    handleResize() {
      this.viewportWidth = window.innerWidth
      if (!this.isMobileViewport) {
        this._cancelEdgeSwipe()
        this.closeSidebar()
      }
    },
    toggleSidebar() {
      this.layoutStore.toggleSidebar()
    },
    closeSidebar() {
      this.layoutStore.closeSidebar()
    },

    // ── Edge swipe-to-open ────────────────────────────────────────────────────
    // Only fires when sidebar is CLOSED and touch starts from the correct edge.
    // The sidebar's own handlers cover swipe-to-close.
    _onDocTouchStart(e) {
      if (!this.isMobileViewport || this.sidebarOpen) return
      const touch = e.changedTouches[0]
      if (!touch) return
      const x = touch.clientX
      const isRight = this.sidebarPosition === 'right'
      // Only begin gesture if touch originates within EDGE_ZONE px of the sidebar edge
      const EDGE_ZONE = 28
      const fromCorrectEdge = isRight
        ? (window.innerWidth - x) <= EDGE_ZONE
        : x <= EDGE_ZONE
      if (!fromCorrectEdge) return
      this.edgeSwipe = { active: true, startX: x, currentX: x, startTime: Date.now() }
      this.layoutStore.setSwipeOpenDrag(0)
    },

    _onDocTouchMove(e) {
      if (!this.edgeSwipe.active) return
      const touch = e.changedTouches[0]
      if (!touch) return
      const x = touch.clientX
      this.edgeSwipe.currentX = x
      const isRight = this.sidebarPosition === 'right'
      const sidebarW = Math.min(window.innerWidth * 0.86, 320)
      // Inward drag distance: positive = moving toward center from the edge
      const dx = isRight
        ? this.edgeSwipe.startX - x   // right edge: drag left
        : x - this.edgeSwipe.startX   // left  edge: drag right
      // Cancel if user immediately drags the wrong way (scrolling)
      if (dx < -10) {
        this._cancelEdgeSwipe()
        return
      }
      const progress = Math.min(1, Math.max(0, dx / sidebarW))
      this.layoutStore.setSwipeOpenDrag(progress)
    },

    _onDocTouchEnd(e) {
      if (!this.edgeSwipe.active) return
      const touch = e.changedTouches[0]
      if (!touch) return
      const x = touch.clientX
      const isRight = this.sidebarPosition === 'right'
      const sidebarW = Math.min(window.innerWidth * 0.86, 320)
      const dx = isRight
        ? this.edgeSwipe.startX - x
        : x - this.edgeSwipe.startX
      const dt = Math.max(1, Date.now() - this.edgeSwipe.startTime)
      const velocity = dx / dt  // px/ms; positive = inward
      const progress = dx / sidebarW
      // Open if dragged > 30% OR flung fast inward
      if (progress > 0.3 || velocity > 0.45) {
        this.layoutStore.openSidebar()
      }
      // Clear drag progress; CSS transition will animate to final state
      this.layoutStore.setSwipeOpenDrag(null)
      this.edgeSwipe = { active: false, startX: 0, currentX: 0, startTime: 0 }
    },

    _onDocTouchCancel() {
      this._cancelEdgeSwipe()
    },

    _cancelEdgeSwipe() {
      this.layoutStore.setSwipeOpenDrag(null)
      this.edgeSwipe = { active: false, startX: 0, currentX: 0, startTime: 0 }
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
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  /* Smooth fade when opening/closing via button; instant override when dragging (inline style) */
  transition: opacity 0.28s ease;
}

.sidebar-backdrop.active {
  display: block;
  opacity: 1;
}

@media (min-width: 993px) {
  .sidebar-backdrop { display: none !important; }
}
</style>
