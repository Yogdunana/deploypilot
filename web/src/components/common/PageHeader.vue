<script setup lang="ts">
import { cn } from '@/lib/utils'

interface Props {
  title?: string
  description?: string
  class?: string
}

const props = defineProps<Props>()
</script>

<template>
  <div :class="cn('flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between py-4', props.class)">
    <div class="min-w-0">
      <h1 v-if="title" class="text-lg sm:text-xl font-semibold text-foreground truncate">{{ title }}</h1>
      <p v-if="description" class="mt-0.5 sm:mt-1 text-sm text-muted-foreground">{{ description }}</p>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <slot name="actions" />
    </div>
  </div>
</template>
