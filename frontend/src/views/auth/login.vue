<template>
  <div class="auth-wrapper">
    <div class="auth-card" :class="{ shake: nudge }">
      <!-- Logo -->
      <div class="auth-logo">
        <div class="logo-icon">
          <i class="mdi mdi-shield-half-full"></i>
        </div>
        <div class="logo-name">Sentinel<span>Core</span></div>
      </div>

      <h5 class="auth-title">Welcome back</h5>
      <p class="auth-subtitle">Sign in to access your security dashboard</p>

      <!-- Alert -->
      <div v-if="error" class="alert alert-danger d-flex align-items-center gap-2 mb-3 py-2" style="font-size:0.8rem">
        <i class="mdi mdi-alert-circle-outline"></i>
        {{ error }}
      </div>

      <div v-if="step === 'credentials'" class="alert alert-warning d-flex align-items-center gap-2 mb-3 py-2" style="font-size:0.78rem;background:rgba(245,166,35,.12);border:none;color:#f5a623">
        <i class="mdi mdi-shield-alert-outline"></i>
        5 failed attempts will lock your account for 15 minutes.
      </div>

      <form @submit.prevent="handleLogin">
        <!-- Step 1: Credentials -->
        <template v-if="step === 'credentials'">
          <div class="mb-3">
            <label class="form-label">Username</label>
            <div class="input-group input-group-icon">
              <span class="input-group-text"><i class="mdi mdi-account-outline"></i></span>
              <input v-model="form.username" type="text" class="form-control" placeholder="admin" autocomplete="username" required />
            </div>
          </div>
          <div class="mb-3">
            <label class="form-label">Password</label>
            <div class="input-group input-group-icon">
              <span class="input-group-text"><i class="mdi mdi-lock-outline"></i></span>
              <input v-model="form.password" :type="showPw ? 'text' : 'password'" class="form-control" placeholder="••••••••" autocomplete="current-password" required />
              <button type="button" class="input-group-text" @click="showPw = !showPw" style="cursor:pointer;border-left:none">
                <i :class="`mdi ${showPw ? 'mdi-eye-off-outline' : 'mdi-eye-outline'}`"></i>
              </button>
            </div>
          </div>
        </template>

        <!-- Step 2: TOTP -->
        <template v-else>
          <div class="mb-3 text-center" style="color:#8aa4c8;font-size:0.82rem">
            <i class="mdi mdi-two-factor-authentication me-1" style="color:#22d67c"></i>
            Enter the 6-digit code from your authenticator app
          </div>
          <div class="mb-3">
            <label class="form-label">Authenticator Code</label>
            <div class="input-group input-group-icon">
              <span class="input-group-text"><i class="mdi mdi-cellphone-key"></i></span>
              <input
                v-model="form.totp"
                type="text"
                inputmode="numeric"
                pattern="[0-9]{6}"
                maxlength="6"
                class="form-control"
                :class="{ 'is-invalid': totpInvalid }"
                placeholder="000 000"
                style="font-family:monospace;letter-spacing:0.4em;font-size:1.1rem;text-align:center"
                autofocus
                required
                @input="onTotpInput"
              />
            </div>
          </div>
          <button type="button" class="btn btn-sm btn-link p-0 mb-3" style="color:#5a7499;font-size:0.75rem" @click="backToCredentials">
            <i class="mdi mdi-arrow-left me-1"></i>Back to login
          </button>
        </template>

        <!-- Submit -->
        <button type="submit" class="btn btn-login mt-1" :disabled="loading">
          <span v-if="loading" class="spinner-border spinner-border-sm me-2" role="status"></span>
          <i v-else class="mdi mdi-login-variant me-2"></i>
          {{ loading ? 'Verifying…' : (step === '2fa' ? 'Verify Code' : 'Sign In') }}
        </button>
      </form>

      <div class="auth-footer-note">
        <i class="mdi mdi-lock-outline me-1"></i>
        Localhost-only mode • All connections are logged
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'LoginPage',
  data() {
    return {
      form: { username: '', password: '', totp: '' },
      showPw: false,
      loading: false,
      error: null,
      step: 'credentials', // 'credentials' | '2fa'
      pendingToken: null,
      isDev: import.meta.env.DEV,
      nudge: false,
      totpInvalid: false
    }
  },
  methods: {
    triggerNudge() {
      this.nudge = false
      requestAnimationFrame(() => {
        this.nudge = true
        setTimeout(() => { this.nudge = false }, 450)
      })
    },
    onTotpInput() {
      this.totpInvalid = false
      const raw = (this.form.totp || '').replace(/\s+/g, '')
      this.form.totp = raw
      if (raw.length === 6 && !this.loading) {
        this.handleLogin()
      }
    },
    async handleLogin() {
      this.loading = true
      this.error = null

      try {
        if (this.step === '2fa') {
          await this.handleVerify2FA()
          return
        }

        const result = await this.$store.dispatch('auth/login', {
          username: this.form.username,
          password: this.form.password
        })

        if (result.success) {
          sessionStorage.setItem('sc_preload_dashboard', '1')
          this.$router.push('/dashboard')
        } else if (result.requires_2fa) {
          this.pendingToken = result.pending_token
          this.step = '2fa'
          this.error = null
        } else {
          this.error = result.message || 'Invalid credentials'
          this.triggerNudge()
        }
      } catch (e) {
        console.error('Login error:', e)
        this.error = e.response?.status === 429
          ? 'Too many login attempts — please wait a few minutes before trying again.'
          : 'Connection error — is SentinelCore daemon running?'
        this.triggerNudge()
      } finally {
        this.loading = false
      }
    },

    async handleVerify2FA() {
      try {
        const result = await this.$store.dispatch('auth/verify2fa', {
          pending_token: this.pendingToken,
          code: this.form.totp
        })
        if (result.success) {
          sessionStorage.setItem('sc_preload_dashboard', '1')
          this.$router.push('/dashboard')
        } else {
          this.error = result.message || 'Invalid 2FA code'
          this.totpInvalid = true
          this.form.totp = ''
          this.triggerNudge()
        }
      } finally {
        this.loading = false
      }
    },

    backToCredentials() {
      this.step = 'credentials'
      this.pendingToken = null
      this.form.totp = ''
      this.error = null
      this.totpInvalid = false
    }
  }
}
</script>

<style scoped>
.shake {
  animation: sc-shake 0.42s ease;
}

@keyframes sc-shake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-10px); }
  40% { transform: translateX(10px); }
  60% { transform: translateX(-8px); }
  80% { transform: translateX(8px); }
}

.is-invalid {
  border-color: #f04040 !important;
  box-shadow: 0 0 0 0.2rem rgba(240, 64, 64, 0.15) !important;
}
</style>
