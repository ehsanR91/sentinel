import { defineStore } from 'pinia'

function resolveTheme(theme) {
  return theme === 'system'
    ? (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark')
    : theme
}

export const useLayoutStore = defineStore('layout', {
  state: () => ({
    sidebarOpen: false,
    swipeOpenDrag: null,        // null | number 0-1 (open gesture progress)
    swipeCloseDrag: null,       // null | number px (close gesture live offset)
    sidebarCollapsed: JSON.parse(localStorage.getItem('sc_sidebar_collapsed') || 'false'),
    sidebarHidden: JSON.parse(localStorage.getItem('sc_sidebar_hidden') || 'false'),
    sidebarDensity: localStorage.getItem('sc_sidebar_density') || 'comfortable',
    sidebarPosition: localStorage.getItem('sc_sidebar_position') || 'left',
    theme: localStorage.getItem('sc_theme') || 'system'
  }),

  actions: {
    toggleSidebar() {
      this.sidebarOpen = !this.sidebarOpen
    },
    openSidebar() {
      this.sidebarOpen = true
    },
    closeSidebar() {
      this.sidebarOpen = false
    },
    setSwipeOpenDrag(v) {
      this.swipeOpenDrag = v
    },
    setSwipeCloseDrag(v) {
      this.swipeCloseDrag = v
    },
    toggleCollapsed() {
      this.setCollapsed(!this.sidebarCollapsed)
    },
    setCollapsed(value) {
      this.sidebarCollapsed = !!value
      localStorage.setItem('sc_sidebar_collapsed', JSON.stringify(this.sidebarCollapsed))
    },
    toggleVisibility() {
      this.setVisibility(!this.sidebarHidden)
    },
    setVisibility(value) {
      this.sidebarHidden = !!value
      localStorage.setItem('sc_sidebar_hidden', JSON.stringify(this.sidebarHidden))
    },
    setSidebarDensity(density) {
      this.sidebarDensity = density === 'compact' ? 'compact' : 'comfortable'
      localStorage.setItem('sc_sidebar_density', this.sidebarDensity)
    },
    setSidebarPosition(position) {
      this.sidebarPosition = position === 'right' ? 'right' : 'left'
      localStorage.setItem('sc_sidebar_position', this.sidebarPosition)
    },
    setTheme(theme) {
      this.theme = theme
      localStorage.setItem('sc_theme', theme)
      const resolved = resolveTheme(theme)
      document.documentElement.setAttribute('data-theme', resolved)
      window.dispatchEvent(new CustomEvent('sc:theme-change', { detail: resolved }))
    }
  }
})