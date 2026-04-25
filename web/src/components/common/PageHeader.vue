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
  <div :class="cn('flex items-center justify-between py-4', props.class)">
    <div>
      <h1 v-if="title" class="text-xl font-semibold text-foreground">{{ title }}</h1>
      <p v-if="description" class="mt-1 text-sm text-muted-foreground">{{ description }}</p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
