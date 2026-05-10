import { defineStore } from 'pinia'

export const useLockStore = defineStore('lock', {
  state: () => ({
    enabled: false,
    pinSet: false
  }),

  getters: {
    lockEnabled: state => state.enabled,
    lockPinSet: state => state.pinSet
  },

  actions: {
    setLockState(payload) {
      this.enabled = !!payload?.enabled
      this.pinSet = !!payload?.pinSet
    },
    clearLock() {
      this.enabled = false
      this.pinSet = false
    }
  }
})