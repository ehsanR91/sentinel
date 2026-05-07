const state = () => ({
  enabled: false,
  pinSet: false
})

const mutations = {
  SET_LOCK(state, { enabled, pinSet }) {
    state.enabled = !!enabled
    state.pinSet = !!pinSet
  },
  CLEAR_LOCK(state) {
    state.enabled = false
    state.pinSet = false
  }
}

const actions = {
  setLockState({ commit }, payload) {
    commit('SET_LOCK', payload || { enabled: false, pinSet: false })
  },
  clearLock({ commit }) {
    commit('CLEAR_LOCK')
  }
}

const getters = {
  lockEnabled: state => state.enabled,
  lockPinSet:  state => state.pinSet
}

export default { namespaced: true, state, mutations, actions, getters }
