<template>
  <div class="duration-picker">
    <NumberField :input-id="`${inputId}-value`" :model-value="modelValue.value" @update:modelValue="update('value', $event)" />
    <SelectField :input-id="`${inputId}-unit`" :model-value="modelValue.unit" :options="units" @update:modelValue="update('unit', $event)" />
  </div>
</template>

<script>
import NumberField from './number-field.vue'
import SelectField from './select-field.vue'

export default {
  name: 'DurationPicker',
  components: { NumberField, SelectField },
  props: {
    modelValue: {
      type: Object,
      default: () => ({ value: 30, unit: 'days' })
    },
    inputId: { type: String, required: true }
  },
  emits: ['update:modelValue'],
  data() {
    return {
      units: [
        { label: 'minutes', value: 'minutes' },
        { label: 'hours', value: 'hours' },
        { label: 'days', value: 'days' }
      ]
    }
  },
  methods: {
    update(key, value) {
      this.$emit('update:modelValue', { ...this.modelValue, [key]: value })
    }
  }
}
</script>

<style scoped>
.duration-picker {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px;
  gap: var(--space-8);
}

@media (max-width: 768px) {
  .duration-picker {
    grid-template-columns: 1fr;
  }
}
</style>