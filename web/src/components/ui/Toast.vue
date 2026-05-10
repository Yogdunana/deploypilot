<script setup lang="ts">
import { ref, provide, readonly } from 'vue'
import { cn } from '@/lib/utils'
import { X, CheckCircle, AlertCircle } from 'lucide-vue-next'

interface ToastItem {
  id: number
  message: string
  variant: 'default' | 'success' | 'destructive'
}

const toasts = ref<ToastItem[]>([])
let nextId = 0

function toast(message: string, variant: ToastItem['variant'] = 'default') {
  const id = nextId++
  toasts.value.push({ id, message, variant })
  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }, 3000)
}

function removeToast(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

provide('toast', { toast })

const variantClasses: Record<string, string> = {
  default: 'border-border bg-card text-foreground',
  success: 'border-success/30 bg-success/10 text-success',
  destructive: 'border-destructive/30 bg-destructive/10 text-destructive',
  error: 'border-destructive/30 bg-destructive/10 text-destructive',
  warning: 'bg-yellow-50 text-yellow-900 border-yellow-200 dark:bg-yellow-950 dark:text-yellow-200 dark:border-yellow-800',
  info: 'bg-blue-50 text-blue-900 border-blue-200 dark:bg-blue-950 dark:text-blue-200 dark:border-blue-800',
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2">
      <TransitionGroup
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="opacity-0 translate-x-4"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-4"
      >
        <div
          v-for="item in toasts"
          :key="item.id"
          :class="cn(
            'flex items-center gap-3 rounded-lg border px-4 py-3 shadow-lg min-w-[280px] max-w-[420px]',
            variantClasses[item.variant]
          )"
        >
          <CheckCircle v-if="item.variant === 'success'" class="w-4 h-4 shrink-0" />
          <AlertCircle v-else-if="item.variant === 'destructive'" class="w-4 h-4 shrink-0" />
          <span class="text-sm flex-1">{{ item.message }}</span>
          <button
            class="shrink-0 rounded-sm opacity-70 hover:opacity-100 transition-opacity cursor-pointer"
            @click="removeToast(item.id)"
          >
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
