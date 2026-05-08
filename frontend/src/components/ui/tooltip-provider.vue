<template>
  <RekaTooltipProvider
    :delay-duration="delayDuration"
    :skip-delay-duration="skipDelayDuration"
    :disable-hoverable-content="disableHoverableContent"
    :ignore-non-keyboard-focus="ignoreNonKeyboardFocus"
    :content="contentDefaults"
  >
    <slot />
  </RekaTooltipProvider>
</template>

<script>
import { TooltipProvider as RekaTooltipProvider } from 'reka-ui'

export default {
  name: 'TooltipProvider',
  components: { RekaTooltipProvider },
  props: {
    delayDuration: { type: Number, default: 500 },
    skipDelayDuration: { type: Number, default: 300 },
    disableHoverableContent: { type: Boolean, default: true },
    ignoreNonKeyboardFocus: { type: Boolean, default: true }
  },
  computed: {
    contentDefaults () {
      return {
        sideOffset: 7,
        collisionPadding: 12,
        avoidCollisions: true,
        sticky: 'partial',
        hideWhenDetached: true,
        updatePositionStrategy: 'always'
      }
    }
  }
}
</script>