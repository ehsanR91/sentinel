/**
 * Idle session timeout and security event logging plugin
 */
export default {
  install(app, options) {
    const IDLE_TIMEOUT = 15 * 60 * 1000 // 15 minutes
    const WARNING_TIMEOUT = 2 * 60 * 1000 // 2 minutes before timeout
    let idleTimer = null
    let warningTimer = null
    let lastActivity = Date.now()
    let onIdleCallback = null
    let router = null
    let api = null

    function resetTimers() {
      lastActivity = Date.now()
      clearTimeout(idleTimer)
      clearTimeout(warningTimer)
      warningTimer = setTimeout(() => {
        // Show warning (optional UI)
        console.warn('Session will expire soon due to inactivity')
      }, IDLE_TIMEOUT - WARNING_TIMEOUT)
      idleTimer = setTimeout(() => {
        logSecurityEvent('session_timeout_idle')
        onIdleCallback?.()
        router?.push('/login')
      }, IDLE_TIMEOUT)
    }

    function logSecurityEvent(event) {
      if (!api) return
      try {
        api.post('/api/v1/security/log', { event, ts: Date.now() }).catch(() => {})
      } catch {}
    }

    function activityHandler() {
      resetTimers()
    }

    const events = ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart', 'click']

    function start(opts) {
      if (!opts || !opts.router || !opts.api) return
      router = opts.router
      api = opts.api
      onIdleCallback = opts.onIdle
      events.forEach(e => document.addEventListener(e, activityHandler, true))
      resetTimers()
    }

    function stop() {
      events.forEach(e => document.removeEventListener(e, activityHandler, true))
      clearTimeout(idleTimer)
      clearTimeout(warningTimer)
    }

    app.config.globalProperties.$idleTimer = { start, stop, reset: resetTimers, logSecurityEvent }
    app.provide('idleTimer', { start, stop, reset: resetTimers, logSecurityEvent })
  }
}
