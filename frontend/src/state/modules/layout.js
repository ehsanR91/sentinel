export const namespaced = true

const state = () => ({
  sidebarOpen: false,
  sidebarCollapsed: JSON.parse(localStorage.getItem('sc_sidebar_collapsed') || 'false'),
  sidebarHidden: JSON.parse(localStorage.getItem('sc_sidebar_hidden') || 'false'),
  sidebarDensity: localStorage.getItem('sc_sidebar_density') || 'comfortable',
  sidebarPosition: localStorage.getItem('sc_sidebar_position') || 'left',
  theme: localStorage.getItem('sc_theme') || 'system'
})

const mutations = {
  TOGGLE_SIDEBAR(state) {
    state.sidebarOpen = !state.sidebarOpen
  },
  OPEN_SIDEBAR(state) {
    state.sidebarOpen = true
  },
  CLOSE_SIDEBAR(state) {
    state.sidebarOpen = false
  },
  TOGGLE_COLLAPSED(state) {
    state.sidebarCollapsed = !state.sidebarCollapsed
    localStorage.setItem('sc_sidebar_collapsed', JSON.stringify(state.sidebarCollapsed))
  },
  SET_COLLAPSED(state, value) {
    state.sidebarCollapsed = !!value
    localStorage.setItem('sc_sidebar_collapsed', JSON.stringify(state.sidebarCollapsed))
  },
  TOGGLE_VISIBILITY(state) {
    state.sidebarHidden = !state.sidebarHidden
    localStorage.setItem('sc_sidebar_hidden', JSON.stringify(state.sidebarHidden))
  },
  SET_VISIBILITY(state, value) {
    state.sidebarHidden = !!value
    localStorage.setItem('sc_sidebar_hidden', JSON.stringify(state.sidebarHidden))
  },
  SET_SIDEBAR_DENSITY(state, density) {
    state.sidebarDensity = density === 'compact' ? 'compact' : 'comfortable'
    localStorage.setItem('sc_sidebar_density', state.sidebarDensity)
  },
  SET_SIDEBAR_POSITION(state, position) {
    state.sidebarPosition = position === 'right' ? 'right' : 'left'
    localStorage.setItem('sc_sidebar_position', state.sidebarPosition)
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
