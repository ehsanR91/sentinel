import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import store from './state/store'
import VueApexCharts from 'vue3-apexcharts'
import VueSweetalert2 from 'vue-sweetalert2'
import 'sweetalert2/dist/sweetalert2.min.css'
import vClickOutside from 'v-click-outside'
import 'nprogress/nprogress.css'
import idleTimer from '@/plugins/idle-timer'
import api from '@/services/api'
import { pwaState, promptInstall } from '@/plugins/pwa'
import '@/utils/console-guard'

import '@/design/index.scss'

// Online/offline state helper (used for offline PWA behavior)
window.__sc_is_online__ = navigator.onLine
window.addEventListener('online', () => {
  window.__sc_is_online__ = true
  window.dispatchEvent(new CustomEvent('sentinel:online'))
})
window.addEventListener('offline', () => {
  window.__sc_is_online__ = false
  window.dispatchEvent(new CustomEvent('sentinel:offline'))
})

// Apply theme before mount to avoid flash
const savedTheme = localStorage.getItem('sc_theme') || 'system'
const resolved = savedTheme === 'system'
  ? (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark')
  : savedTheme
document.documentElement.setAttribute('data-theme', resolved)

const app = createApp(App)

app.config.globalProperties.$isOnline = () => window.__sc_is_online__
app.config.globalProperties.$pwaState = pwaState
app.config.globalProperties.$promptInstall = promptInstall

app.use(router)
app.use(store)
app.use(VueApexCharts)
app.use(VueSweetalert2, {
  target: 'body',
  backdrop: true,
  heightAuto: false,
  allowOutsideClick: true,
  customClass: {
    popup: 'sc-swal-popup'
  }
})
app.use(vClickOutside)
app.use(idleTimer)

app.mount('#app')

// Start idle timer after app is mounted and auth state may be available
app.config.globalProperties.$idleTimer.start({
  router,
  api,
  onIdle: () => {
    console.warn('Idle timeout: redirecting to login')
  }
})
