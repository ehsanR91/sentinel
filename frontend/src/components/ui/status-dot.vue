<template>
  <span class="sc-status-dot" :class="`sc-status-dot--${resolvedState}`" :aria-label="label" role="img"></span>
</template>

<script>
export default {
  name: 'StatusDot',
  props: {
    status: { type: String, default: 'offline' }
  },
  computed: {
    resolvedState() {
      const allowed = ['online', 'away', 'dnd', 'offline']
      return allowed.includes(this.status) ? this.status : 'offline'
    },
    label() {
      const labels = {
        online: 'Online',
        away: 'Away',
        dnd: 'Do not disturb',
        offline: 'Offline'
      }
      return labels[this.resolvedState]
    }
  }
}
</script>

<style scoped>
.sc-status-dot {
  display: inline-flex;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  border: 2px solid var(--surface-1);
  box-shadow: 0 0 0 1px color-mix(in srgb, #000 10%, transparent);
}

.sc-status-dot--online {
  background: #22d67c;
}

.sc-status-dot--away {
  background: #f5a623;
}

.sc-status-dot--dnd {
  background: #f04040;
}

.sc-status-dot--offline {
  background: #7d8ea8;
}
</style>