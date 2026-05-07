<template>
  <footer>
    <div class="footer-left">
      <i class="mdi mdi-shield-half-full" style="color:#4a9eff"></i>
      <span>SentinelCore v1.0.0</span>
      <span style="color:var(--sc-border)">|</span>
      <a href="https://github.com/ehsanR91" target="_blank">github.com/ehsanR91</a>
    </div>
    <div class="footer-right">
      <span v-if="clientIp" style="color:var(--sc-text-muted);font-size:0.72rem">
        <i class="mdi mdi-ip-network-outline" style="margin-right:3px;opacity:0.6"></i>Your IP: <span style="color:var(--sc-text-secondary);font-family:monospace">{{ clientIp }}</span>
      </span>
      <span style="color:var(--sc-border);margin:0 6px" v-if="clientIp">|</span>
      <span>{{ now }}</span>
    </div>
  </footer>
</template>

<script>
import api from '@/services/api'

export default {
  name: 'Footer',
  data() {
    return { now: '', clientIp: '' }
  },
  created() {
    this.tick()
    setInterval(this.tick, 1000)
    this.loadIp()
  },
  methods: {
    tick() {
      this.now = new Date().toLocaleString('en-GB', { hour12: false })
    },
    async loadIp() {
      try {
        const res = await api.getMe()
        this.clientIp = res.data?.client_ip || ''
      } catch (_) {}
    }
  }
}
</script>
