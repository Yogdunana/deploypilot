<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { formatRelativeTime } from '@/lib/utils'
import { cn } from '@/lib/utils'

interface Props {
  date?: string | Date
  class?: string
}

const props = defineProps<Props>()

const formatted = ref('')
let timer: ReturnType<typeof setInterval> | null = null

function update() {
  if (props.date) {
    formatted.value = formatRelativeTime(props.date)
  }
}

onMounted(() => {
  update()
  timer = setInterval(update, 60000)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <time :class="cn('text-sm text-muted-foreground', props.class)" :datetime="date ? new Date(date).toISOString() : undefined">
    {{ formatted }}
  </time>
</template>
