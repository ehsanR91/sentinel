<template>
  <span v-if="tokens.length" class="sc-tooltip__shortcut" aria-hidden="true">
    <span v-for="token in tokens" :key="token" class="sc-tooltip__keycap">{{ token }}</span>
  </span>
</template>

<script>
import { normalizeShortcut } from '@/utils/tooltip-shortcut'

export default {
  name: 'TooltipShortcut',
  props: {
    shortcut: {
      type: [String, Array],
      default: ''
    }
  },
  computed: {
    tokens () {
      return normalizeShortcut(this.shortcut)
    }
  }
}
</script>