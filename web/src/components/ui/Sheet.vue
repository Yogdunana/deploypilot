<script setup lang="ts">
import { watch, onMounted, onBeforeUnmount } from 'vue'
import { cn } from '@/lib/utils'
import { X } from 'lucide-vue-next'

interface Props {
  open?: boolean
  title?: string
  side?: 'right' | 'left'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  side: 'right',
})

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

function close() {
  emit('update:open', false)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.open) {
    close()
  }
}

watch(() => props.open, (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
})

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-50 bg-black/60"
        @click="close"
      />
    </Transition>
    <Transition
      :enter-active-class="'transition duration-200 ease-out'"
      :enter-from-class="side === 'right' ? 'translate-x-full' : '-translate-x-full'"
      :enter-to-class="'translate-x-0'"
      :leave-active-class="'transition duration-150 ease-in'"
      :leave-from-class="'translate-x-0'"
      :leave-to-class="side === 'right' ? 'translate-x-full' : '-translate-x-full'"
    >
      <div
        v-if="open"
        :class="cn(
          'fixed inset-y-0 z-50 flex flex-col bg-card border-border shadow-lg',
          side === 'right' ? 'right-0 border-l' : 'left-0 border-r',
          'w-full sm:max-w-md',
          props.class
        )"
      >
        <div v-if="title" class="flex items-center justify-between border-b border-border px-6 py-4">
          <h2 class="text-lg font-semibold text-foreground">{{ title }}</h2>
          <button
            class="rounded-sm opacity-70 transition-opacity hover:opacity-100 cursor-pointer"
            @click="close"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
        <div class="flex-1 overflow-auto p-6">
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
