<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  src?: string
  name?: string
  size?: 'sm' | 'md' | 'lg'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  name: '',
  size: 'md',
})

const sizeClasses = computed(() => {
  switch (props.size) {
    case 'sm': return 'h-7 w-7 text-xs'
    case 'lg': return 'h-12 w-12 text-lg'
    case 'md':
    default: return 'h-9 w-9 text-sm'
  }
})

const initials = computed(() => {
  if (!props.name) return '?'
  const parts = props.name.trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  }
  return props.name.slice(0, 2).toUpperCase()
})
</script>

<template>
  <span :class="cn('relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full', sizeClasses, props.class)">
    <img
      v-if="src"
      :src="src"
      :alt="name"
      class="aspect-square h-full w-full object-cover"
    />
    <span
      v-else
      class="flex h-full w-full items-center justify-center rounded-full bg-accent font-medium text-muted-foreground"
    >
      {{ initials }}
    </span>
  </span>
</template>
