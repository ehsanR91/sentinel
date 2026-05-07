export const namespaced = true

const state = () => ({
  sidebarOpen: true,
  sidebarCollapsed: JSON.parse(localStorage.getItem('sc_sidebar_collapsed') || 'false'),
  theme: localStorage.getItem('sc_theme') || 'system'
})

const mutations = {
  TOGGLE_SIDEBAR(state) {
    state.sidebarOpen = !state.sidebarOpen
  },
  TOGGLE_COLLAPSED(state) {
    state.sidebarCollapsed = !state.sidebarCollapsed
    localStorage.setItem('sc_sidebar_collapsed', JSON.stringify(state.sidebarCollapsed))
  },
  SET_THEME(state, theme) {
    state.theme = theme
    localStorage.setItem('sc_theme', theme)
    const resolved = theme === 'system'
      ? (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark')
      : theme
    document.documentElement.setAttribute('data-theme', resolved)
  }
}

export default { namespaced: true, state, mutations }
