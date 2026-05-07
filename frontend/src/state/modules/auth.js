const state = () => ({
  user: (() => {
    try { return JSON.parse(sessionStorage.getItem('sc_user') || 'null') } catch { return null }
  })()
})

const mutations = {
  SET_USER(state, user) {
    state.user = user
    if (user) {
      sessionStorage.setItem('sc_user', JSON.stringify(user))
    } else {
      sessionStorage.removeItem('sc_user')
    }
  },
  LOGOUT(state) {
    state.user = null
    sessionStorage.removeItem('sc_user')
  }
}

const actions = {
  async login({ commit }, { username, password }) {
    try {
      const api = (await import('@/services/api')).default
      const { data } = await api.login(username, password)

      // Two-step login: backend returned 2FA challenge
      if (data.requires_2fa) {
        return { success: false, requires_2fa: true, pending_token: data.pending_token }
      }

      const user = { username: data.username, role: data.role }
      commit('SET_USER', user)
      return { success: true }
    } catch (err) {
      const msg = err.response?.data?.error || 'Invalid credentials'
      return { success: false, message: msg }
    }
  },

  async verify2fa({ commit }, { pending_token, code }) {
    try {
      const api = (await import('@/services/api')).default
      const { data } = await api.verify2fa(pending_token, code)
      const user = { username: data.username, role: data.role }
      commit('SET_USER', user)
      return { success: true }
    } catch (err) {
      const msg = err.response?.data?.error || 'Invalid 2FA code'
      return { success: false, message: msg }
    }
  },

  logout({ commit }) {
    commit('LOGOUT')
  }
}

const getters = {
  loggedIn: state => !!state.user,
  user:     state => state.user
}

export default { namespaced: true, state, mutations, actions, getters }
