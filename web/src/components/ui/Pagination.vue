<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

interface Props {
  page?: number
  pageSize?: number
  total?: number
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  page: 1,
  pageSize: 10,
  total: 0,
})

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))

const pages = computed(() => {
  const total = totalPages.value
  const current = props.page
  const items: (number | string)[] = []

  if (total <= 7) {
    for (let i = 1; i <= total; i++) items.push(i)
  } else {
    items.push(1)
    if (current > 3) items.push('...')
    const start = Math.max(2, current - 1)
    const end = Math.min(total - 1, current + 1)
    for (let i = start; i <= end; i++) items.push(i)
    if (current < total - 2) items.push('...')
    items.push(total)
  }

  return items
})

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    emit('update:page', page)
  }
}
</script>

<template>
  <div :class="cn('flex items-center gap-1', props.class)">
    <button
      class="inline-flex h-8 w-8 items-center justify-center rounded-md text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors disabled:opacity-50 disabled:pointer-events-none cursor-pointer"
      :disabled="page <= 1"
      @click="goToPage(page - 1)"
    >
      <ChevronLeft class="w-4 h-4" />
    </button>
    <template v-for="p in pages" :key="p">
      <span v-if="p === '...'" class="inline-flex h-8 w-8 items-center justify-center text-sm text-muted-foreground">
        ...
      </span>
      <button
        v-else
        :class="cn(
          'inline-flex h-8 w-8 items-center justify-center rounded-md text-sm transition-colors cursor-pointer',
          p === page
            ? 'bg-primary text-primary-foreground'
            : 'text-muted-foreground hover:bg-accent hover:text-foreground'
        )"
        @click="goToPage(p as number)"
      >
        {{ p }}
      </button>
    </template>
    <button
      class="inline-flex h-8 w-8 items-center justify-center rounded-md text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors disabled:opacity-50 disabled:pointer-events-none cursor-pointer"
      :disabled="page >= totalPages"
      @click="goToPage(page + 1)"
    >
      <ChevronRight class="w-4 h-4" />
    </button>
  </div>
</template>
