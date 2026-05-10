import { defineStore } from 'pinia'

import api from '@/services/api'

export function getStoredUser() {
  try {
    return JSON.parse(sessionStorage.getItem('sc_user') || 'null')
  } catch {
    return null
  }
}

function persistUser(user) {
  if (user) {
    sessionStorage.setItem('sc_user', JSON.stringify(user))
  } else {
    sessionStorage.removeItem('sc_user')
  }
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: getStoredUser()
  }),

  getters: {
    loggedIn: state => !!state.user
  },

  actions: {
    setUser(user) {
      const nextUser = user ? {
        ...user,
        loginAt: user.loginAt || Date.now()
      } : null
      this.user = nextUser
      persistUser(nextUser)
    },

    clearUser() {
      this.user = null
      persistUser(null)
    },

    async login({ username, password }) {
      try {
        const { data } = await api.login(username, password)
        if (data.requires_2fa) {
          return { success: false, requires_2fa: true, pending_token: data.pending_token }
        }

        this.setUser({ username: data.username, role: data.role })
        return { success: true }
      } catch (err) {
        const msg = err.response?.data?.error || 'Invalid credentials'
        return { success: false, message: msg }
      }
    },

    async verify2fa({ pending_token, code }) {
      try {
        const { data } = await api.verify2fa(pending_token, code)
        this.setUser({ username: data.username, role: data.role })
        return { success: true }
      } catch (err) {
        const msg = err.response?.data?.error || 'Invalid 2FA code'
        return { success: false, message: msg }
      }
    },

    logout() {
      this.clearUser()
    }
  }
})