<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
  variant?: 'text' | 'circular' | 'rectangular'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'text',
})

const classes = computed(() => {
  switch (props.variant) {
    case 'circular':
      return 'rounded-full h-10 w-10'
    case 'rectangular':
      return 'rounded-md'
    case 'text':
    default:
      return 'rounded-md h-4 w-full'
  }
})
</script>

<template>
  <div
    :class="cn('animate-pulse bg-accent', classes, props.class)"
  />
</template>
