const WS_URL = (() => {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const host = import.meta.env.VITE_WS_HOST || window.location.host
  return `${proto}://${host}/ws`
})()

const RECONNECT_DELAY_MS = 3000
const MAX_RECONNECT_DELAY_MS = 30000
const DISCONNECT_GRACE_MS = 1500  // Grace period before showing Offline status

class WSService {
  constructor () {
    this._ws = null
    this._handlers = {}         // type → [callback]
    this._reconnectDelay = RECONNECT_DELAY_MS
    this._intentionalClose = false
    this._statusCallbacks = []
    this._connected = false
    this._disconnectTimer = null
  }

  /** Connect and start auto-reconnect loop. */
  connect () {
    this._intentionalClose = false
    this._open()
  }

  /** Stop and prevent reconnection. */
  disconnect () {
    this._intentionalClose = true
    if (this._disconnectTimer) {
      clearTimeout(this._disconnectTimer)
      this._disconnectTimer = null
    }
    if (this._ws) {
      this._ws.close()
      this._ws = null
    }
    this._setConnected(false)
  }

  /**
   * Subscribe to a typed message.
   * @param {string} type  e.g. 'system.metrics'
   * @param {Function} cb  called with the parsed payload
   * @returns {Function}   unsubscribe function
   */
  on (type, cb) {
    if (!this._handlers[type]) this._handlers[type] = []
    this._handlers[type].push(cb)
    return () => {
      this._handlers[type] = this._handlers[type].filter(h => h !== cb)
    }
  }

  /** Subscribe to connection status changes: cb(isConnected: bool) */
  onStatus (cb) {
    this._statusCallbacks.push(cb)
    cb(this._connected)
    return () => {
      this._statusCallbacks = this._statusCallbacks.filter(h => h !== cb)
    }
  }

  get isConnected () { return this._connected }

  // ── private ────────────────────────────────────────────────────────────────

  _open () {
    if (this._ws) return

    // Clear any pending disconnect timer when attempting to reconnect
    if (this._disconnectTimer) {
      clearTimeout(this._disconnectTimer)
      this._disconnectTimer = null
    }

    const ws = new WebSocket(WS_URL)
    this._ws = ws

    ws.onopen = () => {
      this._reconnectDelay = RECONNECT_DELAY_MS
      // Clear disconnect timer and set connected
      if (this._disconnectTimer) {
        clearTimeout(this._disconnectTimer)
        this._disconnectTimer = null
      }
      this._setConnected(true)
    }

    ws.onmessage = (evt) => {
      try {
        const { type, payload } = JSON.parse(evt.data)
        const handlers = this._handlers[type]
        if (handlers) handlers.forEach(h => h(payload))
      } catch (_) { /* ignore malformed frames */ }
    }

    ws.onerror = () => { /* handled in onclose */ }

    ws.onclose = () => {
      this._ws = null
      // Set a grace period timer before marking as disconnected
      // This prevents flickering during brief interruptions (e.g., page navigation)
      this._disconnectTimer = setTimeout(() => {
        this._disconnectTimer = null
        this._setConnected(false)
      }, DISCONNECT_GRACE_MS)
      
      if (!this._intentionalClose) {
        setTimeout(() => this._open(), this._reconnectDelay)
        this._reconnectDelay = Math.min(this._reconnectDelay * 2, MAX_RECONNECT_DELAY_MS)
      }
    }
  }

  _setConnected (val) {
    this._connected = val
    this._statusCallbacks.forEach(cb => cb(val))
  }
}

export default new WSService()
