<template>
  <button type="button" class="sc-chip sc-chip--interactive sc-focus-ring" :title="title" @click="$emit('click')">
    <span class="user-avatar">{{ initials }}</span>
    <span>{{ user || 'Unknown' }}</span>
  </button>
</template>

<script>
export default {
  name: 'UserChip',
  props: {
    user: { type: String, default: '' },
    email: { type: String, default: '' },
    role: { type: String, default: '' },
    recentCount: { type: Number, default: 0 }
  },
  emits: ['click'],
  computed: {
    initials () {
      const parts = (this.user || '?').split(/[._\s-]+/).filter(Boolean)
      return parts.slice(0, 2).map(part => part[0]?.toUpperCase()).join('') || '?'
    },
    title () {
      const lines = []
      if (this.user) lines.push(this.user)
      if (this.email) lines.push(this.email)
      if (this.role) lines.push(`Role: ${this.role}`)
      if (this.recentCount) lines.push(`Recent events: ${this.recentCount}`)
      return lines.join('\n')
    }
  }
}
</script>

<style scoped>
.user-avatar {
  width: 18px;
  height: 18px;
  display: inline-grid;
  place-items: center;
  border-radius: 50%;
  background: var(--accent-muted);
  color: var(--accent);
  font-size: 10px;
  font-weight: 700;
}
</style>