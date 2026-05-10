<script setup lang="ts">
import { watch, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { cn } from '@/lib/utils'
import { X } from 'lucide-vue-next'
import Button from './Button.vue'

const { t } = useI18n()

interface Props {
  open?: boolean
  title?: string
  description?: string
  confirmText?: string
  cancelText?: string
  variant?: 'default' | 'destructive'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  confirmText: undefined,
  cancelText: undefined,
  variant: 'default',
})

const emit = defineEmits<{
  confirm: []
  cancel: []
  'update:open': [value: boolean]
}>()

function close() {
  emit('update:open', false)
}

function confirm() {
  emit('confirm')
  close()
}

function cancel() {
  emit('cancel')
  close()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.open) {
    cancel()
  }
}

function onOverlayClick(event: MouseEvent) {
  if ((event.target as HTMLElement).classList.contains('alert-dialog-overlay')) {
    cancel()
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
        class="alert-dialog-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/60"
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
              'relative w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-lg',
              props.class
            )"
          >
            <div class="mb-4">
              <h2 class="text-lg font-semibold text-foreground">{{ title }}</h2>
              <p v-if="description" class="mt-2 text-sm text-muted-foreground">{{ description }}</p>
            </div>
            <div class="flex justify-end gap-2">
              <Button variant="outline" size="sm" @click="cancel">
                {{ cancelText || t('common.cancel') }}
              </Button>
              <Button
                :variant="variant === 'destructive' ? 'destructive' : 'default'"
                size="sm"
                @click="confirm"
              >
                {{ confirmText || t('common.confirm') }}
              </Button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
