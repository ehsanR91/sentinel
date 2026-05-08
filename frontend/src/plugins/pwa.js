import { reactive } from 'vue'
import { registerSW } from 'virtual:pwa-register'

const isLocalhost = ['localhost', '127.0.0.1', '[::1]'].includes(window.location.hostname)
const isHttps = window.location.protocol === 'https:'
const isPwaSupported = 'serviceWorker' in navigator && (isHttps || isLocalhost)
const isStandalone = window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone === true
const isIos = /iphone|ipad|ipod/i.test(window.navigator.userAgent) && !window.matchMedia('(display-mode: browser)').matches

const state = reactive({
  isSupported: isPwaSupported,
  isStandalone,
  isIos,
  deferredPrompt: null,
  installRequested: false,
  installed: false,
  updateAvailable: false,
  offlineReady: false,
  needsRefresh: false,
  swRegistration: null,
  lastError: null
})

const registration = isPwaSupported ? registerSW({
  immediate: true,
  onNeedRefresh() {
    state.updateAvailable = true
    state.needsRefresh = true
  },
  onOfflineReady() {
    state.offlineReady = true
  }
}) : null

window.addEventListener('beforeinstallprompt', (event) => {
  event.preventDefault()
  state.deferredPrompt = event
  state.installRequested = true
})

window.addEventListener('appinstalled', () => {
  state.installed = true
  state.deferredPrompt = null
})

window.addEventListener('controllerchange', () => {
  if (state.needsRefresh) {
    window.location.reload()
  }
})

window.addEventListener('load', () => {
  if (!isHttps && !isLocalhost) {
    console.info('PWA features disabled: serve over HTTPS to enable install, service worker, and offline caching.')
  }
})

const promptInstall = async () => {
  if (!state.deferredPrompt) {
    return false
  }
  try {
    const choiceResult = await state.deferredPrompt.prompt()
    state.installed = choiceResult.outcome === 'accepted'
    state.deferredPrompt = null
    return choiceResult
  } catch (err) {
    console.error('PWA install prompt failed', err)
    state.lastError = err
    return null
  }
}

const reloadApp = async () => {
  if (state.needsRefresh && registration) {
    try {
      await registration.update()
      state.updateAvailable = false
      state.needsRefresh = false
      window.location.reload()
    } catch (err) {
      console.error('Failed to refresh PWA', err)
      state.lastError = err
    }
  }
}

const skipWaiting = async () => {
  if (registration && registration()) {
    const reg = await registration()
    if (reg?.update) {
      reg.update()
    }
  }
}

export { state as pwaState, promptInstall, registration, reloadApp }
