<template>
  <div class="sc-user-avatar" :class="sizeClass" :title="name">
    <img
      v-if="!imageFailed && src"
      :src="src"
      :alt="name ? `${name} avatar` : 'User avatar'"
      class="sc-user-avatar__image"
      @error="imageFailed = true"
    >
    <span v-else class="sc-user-avatar__fallback">{{ initials }}</span>
    <span class="sc-user-avatar__status">
      <StatusDot :status="status" />
    </span>
  </div>
</template>

<script>
import StatusDot from './status-dot.vue'

export default {
  name: 'UserAvatar',
  components: { StatusDot },
  props: {
    name: { type: String, default: '' },
    src: { type: String, default: '' },
    status: { type: String, default: 'offline' },
    size: { type: String, default: 'md' }
  },
  data() {
    return {
      imageFailed: false
    }
  },
  computed: {
    initials() {
      const parts = String(this.name || '')
        .trim()
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
      if (!parts.length) return 'SC'
      return parts.map(part => part[0]).join('').toUpperCase()
    },
    sizeClass() {
      return `sc-user-avatar--${this.size}`
    }
  },
  watch: {
    src() {
      this.imageFailed = false
    }
  }
}
</script>

<style scoped>
.sc-user-avatar {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 30%, transparent), color-mix(in srgb, var(--surface-2) 85%, transparent));
  color: var(--text-primary);
  overflow: visible;
}

.sc-user-avatar--sm {
  width: 32px;
  height: 32px;
}

.sc-user-avatar--md {
  width: 40px;
  height: 40px;
}

.sc-user-avatar--lg {
  width: 56px;
  height: 56px;
}

.sc-user-avatar__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: inherit;
}

.sc-user-avatar__fallback {
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.03em;
}

.sc-user-avatar--lg .sc-user-avatar__fallback {
  font-size: 18px;
}

.sc-user-avatar__status {
  position: absolute;
  right: -1px;
  bottom: -1px;
  display: inline-flex;
}
</style>