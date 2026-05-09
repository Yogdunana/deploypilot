<script setup lang="ts">
import { watch, onMounted, onBeforeUnmount } from 'vue'
import { cn } from '@/lib/utils'
import { X } from 'lucide-vue-next'

interface Props {
  open?: boolean
  title?: string
  description?: string
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
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

function onOverlayClick(event: MouseEvent) {
  if ((event.target as HTMLElement).classList.contains('dialog-overlay')) {
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
        class="dialog-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/60"
        @click="onOverlayClick"
      >
        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-if="open"
            :class="cn(
              'relative w-full max-w-lg rounded-lg border border-border bg-card p-6 shadow-lg',
              props.class
            )"
          >
            <button
              v-if="title"
              class="absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 cursor-pointer"
              @click="close"
            >
              <X class="w-4 h-4" />
            </button>
            <div v-if="title" class="mb-4">
              <h2 class="text-lg font-semibold text-foreground">{{ title }}</h2>
              <p v-if="description" class="mt-1 text-sm text-muted-foreground">{{ description }}</p>
            </div>
            <slot />
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
