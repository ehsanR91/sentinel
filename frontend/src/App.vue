<template>
  <TooltipProvider>
    <RouterView v-slot="{ Component, route }">
      <Transition :name="route.meta?.transition || 'app-shell'" mode="out-in">
        <component :is="Component" />
      </Transition>
    </RouterView>

    <CommandPalette />
    <Teleport to="body">
      <div id="sc-tooltip-root"></div>
    </Teleport>

    <Transition name="dash-preload">
      <div
        v-if="showDashboardPreload"
        class="dash-preload"
        role="status"
        aria-live="polite"
        aria-label="Loading dashboard"
      >
        <div class="dash-preload-orb"></div>
        <div class="dash-preload-card">
          <div class="dash-preload-logo">
            <i class="mdi mdi-shield-half-full"></i>
          </div>
          <div class="dash-preload-title">Preparing SentinelCore</div>
          <div class="dash-preload-subtitle">Syncing dashboard widgets and live telemetry</div>
          <div class="dash-preload-bar" aria-hidden="true">
            <span class="dash-preload-progress"></span>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Lock Screen Overlay -->
    <Transition name="lock-fade">
      <div v-if="locked" class="lock-screen" role="dialog" aria-modal="true" aria-label="Screen locked" @click.self="focusActiveDigit">
        <div class="lock-card" :class="{ shake: pinError }" @click="focusActiveDigit">
          <div class="lock-avatar">{{ userInitials }}</div>
          <div class="lock-username">{{ username }}</div>
          <div class="lock-subtitle">Screen locked — enter your 6-digit PIN</div>
          <div class="lock-pin-boxes">
            <input
              v-for="i in 6"
              :key="i"
              :ref="el => { if (el) pinBoxRefs[i-1] = el }"
              v-model="pinDigits[i-1]"
              type="password"
              inputmode="numeric"
              pattern="[0-9]*"
              maxlength="1"
              class="lock-pin-box"
              :class="{ filled: pinDigits[i-1], error: pinError }"
              autocomplete="off"
              @input="onDigitInput(i-1)"
              @keydown="onDigitKeydown($event, i-1)"
              @paste.prevent="onPinPaste"
              @focus="pinFocusIndex = i-1"
            />
          </div>
          <div v-if="pinError" class="lock-error">
            <i class="mdi mdi-alert-circle-outline"></i> Incorrect PIN — try again
          </div>
          <div v-else class="lock-hint">Enter digits to unlock</div>
        </div>
      </div>
    </Transition>
  </TooltipProvider>
</template>

<script>
import CommandPalette from '@/components/command-palette.vue'
import api from '@/services/api'
import { useMetricsStore } from '@/stores/metrics'
import TooltipProvider from '@/components/ui/tooltip-provider.vue'

export default {
  name: 'App',
  components: { CommandPalette, TooltipProvider },
  setup() {
    return {
      metricsStore: useMetricsStore()
    }
  },
  data() {
    return {
      locked: false,
      pinDigits: ['', '', '', '', '', ''],
      pinBoxRefs: [],
      pinFocusIndex: 0,
      pinError: false,
      lockEnabled: false,
      lockPinSet: false,
      showDashboardPreload: false,
      preloadTimer: null
    }
  },
  computed: {
    username() {
      try { return JSON.parse(sessionStorage.getItem('sc_user') || '{}').username || 'user' } catch { return 'user' }
    },
    userInitials() {
      return this.username.slice(0, 2).toUpperCase()
    },
    authToken() {
      return this.$store.state.auth?.token || null
    }
  },
  watch: {
    // When a user logs in, the token goes from null → value. Load lock settings then.
    authToken(newVal, oldVal) {
      if (newVal && !oldVal) {
        this.metricsStore.startLive()
        this.loadLockSettings()
      }
      if (!newVal && oldVal) {
        this.metricsStore.stopLive()
      }
    },
    '$route.path'(path) {
      if (path === '/dashboard') {
        this.maybeRunDashboardPreload()
      }
    }
  },
  created() {
    // Apply theme before any rendering
    const theme = localStorage.getItem('sc_theme') || 'system'
    const resolved = theme === 'system'
      ? (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark')
      : theme
    document.documentElement.setAttribute('data-theme', resolved)
  },
  mounted() {
    // Start WebSocket service globally when authenticated
    if (this.$store.getters['auth/loggedIn']) {
      this.metricsStore.startLive()
      this.loadLockSettings()
    }
    document.addEventListener('keydown', this.handleKeyDown)
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', this.onSystemThemeChange)
    // Listen for lock requests dispatched via CustomEvent (replaces window.$sentinelLock).
    window.addEventListener('sentinel:lock', this.lock)
    window.addEventListener('sentinel:dashboard-ready', this.stopDashboardPreload)
    this.maybeRunDashboardPreload()
  },
  beforeUnmount() {
    document.removeEventListener('keydown', this.handleKeyDown)
    window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', this.onSystemThemeChange)
    window.removeEventListener('sentinel:lock', this.lock)
    window.removeEventListener('sentinel:dashboard-ready', this.stopDashboardPreload)
    this.clearPreloadTimer()
    document.body.style.overflow = ''
  },
  methods: {
    clearPreloadTimer() {
      if (this.preloadTimer) {
        clearTimeout(this.preloadTimer)
        this.preloadTimer = null
      }
    },
    maybeRunDashboardPreload() {
      if (this.$route.path !== '/dashboard') return
      if (sessionStorage.getItem('sc_preload_dashboard') !== '1') return

      this.showDashboardPreload = true
      this.clearPreloadTimer()
      this.preloadTimer = setTimeout(() => {
        this.stopDashboardPreload()
      }, 1150)
    },
    stopDashboardPreload() {
      this.clearPreloadTimer()
      this.showDashboardPreload = false
      sessionStorage.removeItem('sc_preload_dashboard')
    },
    async loadLockSettings() {
      if (!this.$store.getters['auth/loggedIn']) return
      try {
        const { data } = await api.getLockSettings()
        this.lockEnabled = data.enabled || false
        this.lockPinSet = data.pinSet || false
        this.$store.dispatch('lock/setLockState', {
          enabled: this.lockEnabled,
          pinSet: this.lockPinSet
        })
      } catch (err) {
        if (err.response?.status !== 401) {
          console.error('Failed to load lock settings:', err)
        }
      }
    },
    handleKeyDown(e) {
      if (this.locked) {
        // Escape does nothing, keep locked
        if (e.key === 'Escape') { e.preventDefault(); return }
        // Tab cycles through boxes only
        if (e.key === 'Tab') {
          e.preventDefault()
          const next = (this.pinFocusIndex + (e.shiftKey ? -1 : 1) + 6) % 6
          this.pinBoxRefs[next]?.focus()
        }
        return
      }
      if (e.code === 'Space' && !['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName) && this.lockEnabled && this.lockPinSet) {
        e.preventDefault()
        this.lock()
      }
    },
    onSystemThemeChange() {
      const theme = localStorage.getItem('sc_theme') || 'system'
      if (theme === 'system') {
        const resolved = window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
        document.documentElement.setAttribute('data-theme', resolved)
      }
    },
    lock() {
      try {
        if (!this.lockPinSet) {
          console.warn('Lock requested but no PIN set')
          return
        }
        this.locked = true
        this.pinDigits = ['', '', '', '', '', '']
        this.pinError = false
        document.body.style.overflow = 'hidden'
        this.$nextTick(() => {
          this.pinBoxRefs[0]?.focus()
        })
      } catch (err) {
        console.error('Lock failed:', err)
      }
    },
    focusActiveDigit() {
      const idx = this.pinDigits.findIndex(d => d === '')
      const target = idx === -1 ? 5 : idx
      this.pinBoxRefs[target]?.focus()
    },
    onDigitInput(index) {
      // Strip non-numeric characters
      const val = this.pinDigits[index]
      if (val && !/^[0-9]$/.test(val)) {
        this.pinDigits[index] = ''
        return
      }
      if (val && index < 5) {
        this.$nextTick(() => this.pinBoxRefs[index + 1]?.focus())
      }
      // Auto-submit when all 6 filled
      if (this.pinDigits.every(d => d !== '')) {
        this.$nextTick(() => this.unlockAttempt())
      }
    },
    onDigitKeydown(e, index) {
      if (e.key === 'Backspace') {
        if (this.pinDigits[index] === '' && index > 0) {
          this.pinDigits[index - 1] = ''
          this.$nextTick(() => this.pinBoxRefs[index - 1]?.focus())
        } else {
          this.pinDigits[index] = ''
        }
        e.preventDefault()
      } else if (e.key === 'ArrowLeft' && index > 0) {
        this.pinBoxRefs[index - 1]?.focus()
      } else if (e.key === 'ArrowRight' && index < 5) {
        this.pinBoxRefs[index + 1]?.focus()
      } else if (e.key === 'Enter') {
        this.unlockAttempt()
      }
    },
    onPinPaste(e) {
      const text = (e.clipboardData || window.clipboardData).getData('text').replace(/\D/g, '').slice(0, 6)
      if (!text) return
      for (let i = 0; i < 6; i++) {
        this.pinDigits[i] = text[i] || ''
      }
      this.$nextTick(() => {
        const focusIdx = Math.min(text.length, 5)
        this.pinBoxRefs[focusIdx]?.focus()
        if (text.length === 6) this.unlockAttempt()
      })
    },
    async unlockAttempt() {
      const pin = this.pinDigits.join('')
      if (pin.length !== 6) return
      try {
        await api.verifyLockPin(pin)
        this.locked = false
        this.pinDigits = ['', '', '', '', '', '']
        this.pinError = false
        document.body.style.overflow = ''
      } catch (err) {
        console.error('Unlock failed:', err)
        this.pinError = true
        this.pinDigits = ['', '', '', '', '', '']
        this.$nextTick(() => this.pinBoxRefs[0]?.focus())
        setTimeout(() => { this.pinError = false }, 2000)
      }
    }
  }
}
</script>
