<template>
  <Tooltip :label="user || 'Unknown'" :description="tooltipDescription" variant="rich" as-child>
    <button type="button" class="sc-chip sc-chip--interactive sc-focus-ring" @click="$emit('click')">
      <UserAvatar :name="user || 'Unknown'" :status="presence.status" size="sm" />
      <span>{{ user || 'Unknown' }}</span>
    </button>
  </Tooltip>
</template>

<script>
import UserAvatar from './user-avatar.vue'
import Tooltip from './tooltip.vue'
import { getUserPresence, USER_PRESENCE_EVENT } from '@/utils/user-presence'

export default {
  name: 'UserChip',
  components: { Tooltip, UserAvatar },
  props: {
    user: { type: String, default: '' },
    email: { type: String, default: '' },
    role: { type: String, default: '' },
    recentCount: { type: Number, default: 0 }
  },
  emits: ['click'],
  data () {
    return {
      presence: getUserPresence(this.user)
    }
  },
  computed: {
    tooltipDescription () {
      const lines = []
      if (this.user) lines.push(this.user)
      if (this.email) lines.push(this.email)
      if (this.role) lines.push(`Role: ${this.role}`)
      if (this.recentCount) lines.push(`Recent events: ${this.recentCount}`)
      return lines.filter(line => line !== this.user).join('\n')
    }
  },
  watch: {
    user: {
      immediate: true,
      handler (value) {
        this.presence = getUserPresence(value)
      }
    }
  },
  mounted () {
    window.addEventListener(USER_PRESENCE_EVENT, this.handlePresenceChange)
  },
  beforeUnmount () {
    window.removeEventListener(USER_PRESENCE_EVENT, this.handlePresenceChange)
  },
  methods: {
    handlePresenceChange (event) {
      if (event.detail?.username === this.user) {
        this.presence = event.detail.value
      }
    }
  }
}
</script>

<style scoped>
.sc-chip :deep(.sc-user-avatar--sm) {
  width: 18px;
  height: 18px;
}

.sc-chip :deep(.sc-user-avatar__status) {
  right: -2px;
  bottom: -2px;
  transform: scale(0.78);
}
</style>