import { defineStore } from 'pinia'

import ws from '@/services/ws'
import api from '@/services/api'

const HISTORY_LEN = 1800
const MAX_CHART_POINTS = 240
const NUMERIC_SNAP_KEYS = [
  'cpu_pct', 'ram_pct', 'swap_pct', 'disk_pct',
  'ram_used', 'ram_total', 'swap_used', 'swap_total',
  'disk_used', 'disk_total', 'disk_free',
  'net_rx_rate', 'net_tx_rate', 'net_rx_total', 'net_tx_total',
  'load1', 'load5', 'load15', 'uptime', 'unread_alerts', 'active_bans'
]

let subscriptionsRegistered = false

function emptySnap () {
  return {
    cpu_pct: 0, cpu_cores: [],
    ram_pct: 0, ram_used: 0, ram_total: 0,
    swap_pct: 0, swap_used: 0, swap_total: 0,
    disk_pct: 0, disk_used: 0, disk_total: 0, disk_free: 0,
    partitions: [],
    net_rx_rate: 0, net_tx_rate: 0, net_rx_total: 0, net_tx_total: 0,
    load1: 0, load5: 0, load15: 0,
    unread_alerts: 0, active_bans: 0,
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

function clampPercent (value) {
  const num = sanitizeNumber(value)
  if (num === null) return null
  return Math.max(0, Math.min(100, num))
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
  return Array(HISTORY_LEN).fill(null)
}

function initTimestampHistory () {
  const now = Date.now()
  return Array.from({ length: HISTORY_LEN }, (_, index) => now - (HISTORY_LEN - 1 - index) * 1000)
}

function rangeToSeconds (range) {
  return { '1m': 60, '5m': 300, '15m': 900, '1h': 3600 }[range] || 60
}

function sliceHistory (values, timestamps, range) {
  const seconds = rangeToSeconds(range)
  const cutoff = Date.now() - seconds * 1000
  const start = timestamps.findIndex(ts => ts >= cutoff)
  const from = start === -1 ? 0 : start
  const ts = timestamps.slice(from)
  const vals = values.slice(from)
  const points = ts.map((t, index) => ({ x: t, y: vals[index] }))
  if (points.length <= MAX_CHART_POINTS) {
    return points
  }

  const step = Math.ceil(points.length / MAX_CHART_POINTS)
  const sampled = []
  for (let index = 0; index < points.length; index += step) {
    sampled.push(points[index])
  }
  const lastPoint = points[points.length - 1]
  if (sampled[sampled.length - 1] !== lastPoint) {
    sampled.push(lastPoint)
  }
  return sampled
}

function pushHistoryPoint (history, value) {
  history.push(value)
  if (history.length > HISTORY_LEN) {
    history.shift()
  }
}

export const useMetricsStore = defineStore('metrics', {
  state: () => ({
    snap: emptySnap(),
    cpuHistory: initHistory(),
    ramHistory: initHistory(),
    swapHistory: initHistory(),
    diskHistory: initHistory(),
    netRxHistory: initHistory(),
    netTxHistory: initHistory(),
    metricTimestamps: initTimestampHistory(),
    lastMetricTs: 0,
    wsConnected: false,
    processes: [],
    networkProcesses: [],
    services: [],
    liveSummary: {
      unreadAlerts: 0,
      activeBans: 0
    }
  }),

  getters: {
    historySlice: (state) => (key, range) => {
      const histMap = {
        cpu: state.cpuHistory,
        ram: state.ramHistory,
        swap: state.swapHistory,
        disk: state.diskHistory,
        netRx: state.netRxHistory,
        netTx: state.netTxHistory
      }
      return sliceHistory(histMap[key] || [], state.metricTimestamps, range)
    }
  },

  actions: {
    applySnapshot (snap) {
      const sanitizedSnap = sanitizeSnap(snap)
      this.snap = sanitizedSnap

      const metricTs = (sanitizeNumber(sanitizedSnap.ts) || Math.floor(Date.now() / 1000)) * 1000
      const cpuPoint = clampPercent(sanitizedSnap.cpu_pct)
      const ramPoint = clampPercent(sanitizedSnap.ram_pct)
      const swapPoint = clampPercent(sanitizedSnap.swap_pct)
      const diskPoint = clampPercent(sanitizedSnap.disk_pct)
      const rxPoint = sanitizeNumber(sanitizedSnap.net_rx_rate)
      const txPoint = sanitizeNumber(sanitizedSnap.net_tx_rate)

      pushHistoryPoint(this.cpuHistory, cpuPoint)
      pushHistoryPoint(this.ramHistory, ramPoint)
      pushHistoryPoint(this.swapHistory, swapPoint)
      pushHistoryPoint(this.diskHistory, diskPoint)
      pushHistoryPoint(this.netRxHistory, rxPoint)
      pushHistoryPoint(this.netTxHistory, txPoint)
      pushHistoryPoint(this.metricTimestamps, metricTs)
      this.lastMetricTs = Math.floor(metricTs / 1000)
      this.liveSummary = {
        unreadAlerts: sanitizedSnap.unread_alerts || 0,
        activeBans: sanitizedSnap.active_bans || 0
      }

      if (import.meta.env.DEV) {
        assertFiniteHistory(this.cpuHistory, 'cpuHistory')
        assertFiniteHistory(this.ramHistory, 'ramHistory')
        assertFiniteHistory(this.swapHistory, 'swapHistory')
        assertFiniteHistory(this.diskHistory, 'diskHistory')
        assertFiniteHistory(this.netRxHistory, 'netRxHistory')
        assertFiniteHistory(this.netTxHistory, 'netTxHistory')
      }
    },

    setWsConnected (value) {
      this.wsConnected = value
    },

    setProcesses (list) {
      this.processes = Array.isArray(list) ? list : []
    },

    setNetworkProcesses (list) {
      this.networkProcesses = Array.isArray(list) ? list : []
    },

    setServices (list) {
      this.services = Array.isArray(list) ? list : []
    },

    resetLiveSummary () {
      this.liveSummary = {
        unreadAlerts: 0,
        activeBans: 0
      }
    },

    startLive () {
      if (!subscriptionsRegistered) {
        ws.on('system.metrics', payload => {
          this.applySnapshot(payload)
        })

        ws.onStatus(connected => {
          this.setWsConnected(connected)
          if (!connected) {
            this.resetLiveSummary()
          }
        })
        subscriptionsRegistered = true
      }

      ws.connect()
    },

    stopLive () {
      ws.disconnect()
    },

    async fetchProcesses () {
      try {
        const { data } = await api.getProcesses(50)
        this.setProcesses(data)
      } catch (_) {}
    },

    async fetchNetworkProcesses () {
      try {
        const { data } = await api.getNetworkProcesses(50)
        this.setNetworkProcesses(data)
      } catch (_) {
        this.setNetworkProcesses([])
      }
    },

    async fetchServices () {
      try {
        const { data } = await api.getServices()
        this.setServices(data)
      } catch (_) {}
    }
  }
})