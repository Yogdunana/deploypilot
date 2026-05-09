<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  modelValue?: string
  placeholder?: string
  rows?: number
  disabled?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  rows: 3,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const classes = computed(() =>
  cn(
    'flex min-h-[80px] w-full rounded-md border border-border bg-card px-3 py-2 text-sm text-foreground shadow-sm transition-colors duration-150',
    'placeholder:text-muted-foreground',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary',
    'disabled:cursor-not-allowed disabled:opacity-50',
    'resize-none',
    props.class
  )
)

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <textarea
    :class="classes"
    :value="modelValue"
    :placeholder="placeholder"
    :rows="rows"
    :disabled="disabled"
    @input="onInput"
  />
</template>
