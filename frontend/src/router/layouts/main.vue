<template>
  <div class="wrapper" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <Sidebar />
    <!-- Mobile backdrop -->
    <div class="sidebar-backdrop" @click="closeSidebar"></div>
    <div class="page-content">
      <Topbar @toggle-sidebar="toggleSidebar" />
      <div class="content-page">
        <RouterView v-slot="{ Component }">
          <Transition name="page" mode="out-in">
            <component :is="Component" />
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

export default {
  name: 'MainLayout',
  components: { Sidebar, Topbar, Footer },
  computed: {
    sidebarCollapsed() {
      return this.$store.state.layout.sidebarCollapsed
    }
  },
  watch: {
    $route() {
      this.closeSidebar()
    }
  },
  methods: {
    toggleSidebar() {
      const el = document.getElementById('sidebar-menu')
      if (!el) return
      el.classList.toggle('open')
      document.querySelector('.sidebar-backdrop')?.classList.toggle('active')
    },
    closeSidebar() {
      document.getElementById('sidebar-menu')?.classList.remove('open')
      document.querySelector('.sidebar-backdrop')?.classList.remove('active')
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
