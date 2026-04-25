<script setup lang="ts">
import { cn } from '@/lib/utils'

interface Props {
  class?: string
}

const props = defineProps<Props>()
</script>

<template>
  <div :class="cn('relative overflow-auto', props.class)">
    <slot />
  </div>
</template>

<style scoped>
:deep(::-webkit-scrollbar) {
  width: 6px;
  height: 6px;
}
:deep(::-webkit-scrollbar-track) {
  background: transparent;
}
:deep(::-webkit-scrollbar-thumb) {
  background: oklch(0.3 0 0);
  border-radius: 3px;
}
:deep(::-webkit-scrollbar-thumb:hover) {
  background: oklch(0.4 0 0);
}
</style>
