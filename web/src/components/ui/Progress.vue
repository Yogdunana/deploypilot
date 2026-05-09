<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  value?: number
  variant?: 'default' | 'success' | 'warning' | 'destructive'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  value: 0,
  variant: 'default',
})

const clampedValue = computed(() => Math.min(100, Math.max(0, props.value)))

const colorClasses = computed(() => {
  switch (props.variant) {
    case 'success': return 'bg-success'
    case 'warning': return 'bg-warning'
    case 'destructive': return 'bg-destructive'
    default: return 'bg-primary'
  }
})
</script>

<template>
  <div :class="cn('relative h-2 w-full overflow-hidden rounded-full bg-accent', props.class)">
    <div
      :class="cn('h-full rounded-full transition-all duration-300 ease-out', colorClasses)"
      :style="{ width: `${clampedValue}%` }"
    />
  </div>
</template>
