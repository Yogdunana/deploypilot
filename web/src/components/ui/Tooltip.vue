<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  content?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  content: '',
  side: 'top',
})

const isVisible = ref(false)
const triggerRef = ref<HTMLElement | null>(null)

const positionClasses = computed(() => {
  switch (props.side) {
    case 'top': return 'bottom-full left-1/2 -translate-x-1/2 mb-2'
    case 'bottom': return 'top-full left-1/2 -translate-x-1/2 mt-2'
    case 'left': return 'right-full top-1/2 -translate-y-1/2 mr-2'
    case 'right': return 'left-full top-1/2 -translate-y-1/2 ml-2'
    default: return 'bottom-full left-1/2 -translate-x-1/2 mb-2'
  }
})

function show() {
  isVisible.value = true
}

function hide() {
  isVisible.value = false
}
</script>

<template>
  <div ref="triggerRef" class="relative inline-flex" @mouseenter="show" @mouseleave="hide">
    <slot />
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="isVisible && content"
        :class="cn(
          'absolute z-50 overflow-hidden rounded-md border border-border bg-card px-3 py-1.5 text-xs text-foreground shadow-md whitespace-nowrap',
          positionClasses
        )"
      >
        {{ content }}
      </div>
    </Transition>
  </div>
</template>
