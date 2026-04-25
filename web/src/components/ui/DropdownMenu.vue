<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { cn } from '@/lib/utils'

interface DropdownItem {
  label: string
  icon?: any
  action?: () => void
  danger?: boolean
}

interface Props {
  items?: DropdownItem[]
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  items: () => [],
})

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}

function onItemClick(item: DropdownItem) {
  item.action?.()
  close()
}

function onClickOutside(event: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    close()
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', onClickOutside))
</script>

<template>
  <div ref="containerRef" :class="cn('relative inline-block', props.class)">
    <div @click.stop="toggle">
      <slot name="trigger" />
    </div>
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="isOpen"
        class="absolute right-0 z-50 mt-1 min-w-[8rem] overflow-hidden rounded-md border border-border bg-card shadow-lg"
      >
        <div class="p-1">
          <button
            v-for="(item, index) in items"
            :key="index"
            type="button"
            :class="cn(
              'flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm cursor-pointer transition-colors duration-100',
              item.danger
                ? 'text-destructive hover:bg-destructive/10'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'
            )"
            @click="onItemClick(item)"
          >
            <component v-if="item.icon" :is="item.icon" class="w-4 h-4" />
            {{ item.label }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>
