<template>
  <div class="uptime-bar">
    <Tooltip 
      v-for="(segment, index) in segments" 
      :key="index"
      :label="segment.tooltip"
      as-child
    >
      <div 
        class="uptime-bar__segment"
        :class="getSegmentClass(segment)"
      ></div>
    </Tooltip>
  </div>
</template>

<script>
import Tooltip from '@/components/ui/tooltip.vue'

export default {
  name: 'UptimeBar',
  components: { Tooltip },
  props: {
    history: {
      type: Array,
      default: () => []
    },
    // We can display 7 segments or 24 segments based on available history or prop bounds.
    maxSegments: {
      type: Number,
      default: 24
    }
  },
  computed: {
    segments() {
      // Limit to maxSegments
      const limited = this.history.slice(-this.maxSegments)
      return limited.map((entry, index) => {
        // Assume entry can be a boolean or an object with { uptimePct, tooltip }
        // For simple boolean array support:
        const isObject = typeof entry === 'object' && entry !== null
        const ok = isObject ? (entry.ok ?? entry.status === 'ok') : !!entry
        
        let tooltip = 'Uptime slot'
        if (isObject && entry.tooltip) {
          tooltip = entry.tooltip
        } else {
          tooltip = ok ? 'Healthy' : 'Degraded/Offline'
        }

        return {
          ok,
          tooltip
        }
      })
    }
  },
  methods: {
    getSegmentClass(segment) {
      if (segment.ok === undefined || segment.ok === null) return 'is-unknown'
      return segment.ok ? 'is-ok' : 'is-error'
    }
  }
}
</script>

<style scoped>
.uptime-bar {
  display: flex;
  align-items: stretch;
  gap: 2px;
  height: 18px;
  width: 100%;
}

.uptime-bar__segment {
  flex: 1;
  min-width: 3px;
  border-radius: 4px;
  background: var(--surface-1);
  transition: opacity 0.2s, background 0.2s, transform 0.2s;
  cursor: default;
}

.uptime-bar__segment:hover {
  opacity: 0.8;
  transform: scaleY(1.15);
}

.uptime-bar__segment.is-ok {
  background: var(--state-ok, #22d67c);
}

.uptime-bar__segment.is-error {
  background: var(--state-error, #f04040);
}

.uptime-bar__segment.is-unknown {
  background: var(--border-subtle, #3b4252);
}
</style>