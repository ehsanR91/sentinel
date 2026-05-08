import ws from '@/services/ws'

const HISTORY_LEN = 30
const NUMERIC_SNAP_KEYS = [
  'cpu_pct', 'ram_pct', 'swap_pct', 'disk_pct',
  'ram_used', 'ram_total', 'swap_used', 'swap_total',
  'disk_used', 'disk_total', 'disk_free',
  'net_rx_rate', 'net_tx_rate', 'net_rx_total', 'net_tx_total',
  'load1', 'load5', 'load15', 'uptime'
]

function emptySnap () {
  return {
    cpu_pct: 0, cpu_cores: [],
    ram_pct: 0, ram_used: 0, ram_total: 0,
    swap_pct: 0, swap_used: 0, swap_total: 0,
    disk_pct: 0, disk_used: 0, disk_total: 0, disk_free: 0,
    partitions: [],
    net_rx_rate: 0, net_tx_rate: 0, net_rx_total: 0, net_tx_total: 0,
    load1: 0, load5: 0, load15: 0,
    uptime: 0, hostname: '', os: '', kernel: '', platform: '',
    ts: 0
  }
}

function sanitizeNumber (value) {
  const num = Number(value)
  return Number.isFinite(num) ? num : null
}

function sanitizeSnap (snap) {
  const sanitized = { ...snap }
  NUMERIC_SNAP_KEYS.forEach((key) => {
    if (key in sanitized) {
      const num = sanitizeNumber(sanitized[key])
      sanitized[key] = num === null ? 0 : num
    }
  })
  return sanitized
}

function assertFiniteHistory (history, name) {
  if (!import.meta.env.DEV) return
  history.forEach((value, index) => {
    if (value !== null && !Number.isFinite(value)) {
      console.error('Non-finite chart history value', { history: name, index, value })
    }
  })
}

function initHistory () {
  return Array(HISTORY_LEN).fill(0)
}

export default {
  namespaced: true,

  state: () => ({
    snap: emptySnap(),
    cpuHistory: initHistory(),
    ramHistory: initHistory(),
    netRxHistory: initHistory(),
    netTxHistory: initHistory(),
    wsConnected: false,
    processes: [],
    services: []
  }),

  mutations: {
    SET_SNAP (state, snap) {
      const sanitizedSnap = sanitizeSnap(snap)
      state.snap = sanitizedSnap

      const cpuPoint = sanitizeNumber(snap.cpu_pct)
      const ramPoint = sanitizeNumber(snap.ram_pct)
      const rxPoint = sanitizeNumber(snap.net_rx_rate)
      const txPoint = sanitizeNumber(snap.net_tx_rate)

      state.cpuHistory = [...state.cpuHistory.slice(1), cpuPoint]
      state.ramHistory = [...state.ramHistory.slice(1), ramPoint]
      state.netRxHistory = [...state.netRxHistory.slice(1), rxPoint]
      state.netTxHistory = [...state.netTxHistory.slice(1), txPoint]

      if (import.meta.env.DEV) {
        assertFiniteHistory(state.cpuHistory, 'cpuHistory')
        assertFiniteHistory(state.ramHistory, 'ramHistory')
        assertFiniteHistory(state.netRxHistory, 'netRxHistory')
        assertFiniteHistory(state.netTxHistory, 'netTxHistory')
      }
    },
    SET_WS_CONNECTED (state, val) { state.wsConnected = val },
    SET_PROCESSES (state, list) { state.processes = list },
    SET_SERVICES (state, list) { state.services = list }
  },

  actions: {
    startLive ({ commit }) {
      ws.connect()

      ws.on('system.metrics', payload => {
        commit('SET_SNAP', payload)
      })

      ws.onStatus(connected => {
        commit('SET_WS_CONNECTED', connected)
      })
    },

    stopLive () {
      ws.disconnect()
    },

    async fetchProcesses ({ commit }) {
      try {
        const { default: api } = await import('@/services/api')
        const { data } = await api.getProcesses(50)
        commit('SET_PROCESSES', data)
      } catch (_) {}
    },

    async fetchServices ({ commit }) {
      try {
        const { default: api } = await import('@/services/api')
        const { data } = await api.getServices()
        commit('SET_SERVICES', data)
      } catch (_) {}
    }
  },

  getters: {
    snap: s => s.snap,
    cpuHistory: s => s.cpuHistory,
    ramHistory: s => s.ramHistory,
    netRxHistory: s => s.netRxHistory,
    netTxHistory: s => s.netTxHistory,
    wsConnected: s => s.wsConnected,
    processes: s => s.processes,
    services: s => s.services
  }
}
