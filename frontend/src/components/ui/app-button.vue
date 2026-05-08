<template>
  <button
    :type="type"
    class="sc-button sc-focus-ring"
    :class="buttonClasses"
    :disabled="disabled || loading"
    :aria-label="computedAriaLabel"
    @click="$emit('click', $event)"
  >
    <i v-if="loading" class="mdi mdi-loading mdi-spin" aria-hidden="true"></i>
    <i v-else-if="icon" :class="icon" aria-hidden="true"></i>
    <span v-if="!iconOnly"><slot>{{ label }}</slot></span>
  </button>
</template>

<script>
export default {
  name: 'AppButton',
  props: {
    type: { type: String, default: 'button' },
    variant: { type: String, default: 'secondary' },
    size: { type: String, default: 'md' },
    icon: { type: String, default: '' },
    loading: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
    label: { type: String, default: '' },
    ariaLabel: { type: String, default: '' },
    iconOnly: { type: Boolean, default: false }
  },
  emits: ['click'],
  computed: {
    buttonClasses () {
      return [
        `sc-button--${this.variant}`,
        `sc-button--${this.size}`,
        { 'sc-button--icon-only': this.iconOnly }
      ]
    },
    computedAriaLabel () {
      return this.ariaLabel || this.label || undefined
    }
  }
}
</script>