<script setup lang="ts">
import { cn } from '@/lib/utils'
import Button from '@/components/ui/Button.vue'

interface Props {
  icon?: any
  title?: string
  description?: string
  actionText?: string
  class?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  action: []
}>()

function onAction() {
  emit('action')
}
</script>

<template>
  <div :class="cn('flex flex-col items-center justify-center py-12 text-center', props.class)">
    <div v-if="$slots.icon || icon" class="mb-4 text-muted-foreground">
      <slot name="icon">
        <component v-if="icon" :is="icon" class="w-12 h-12 mx-auto" />
      </slot>
    </div>
    <h3 v-if="title" class="text-sm font-medium text-foreground">{{ title }}</h3>
    <p v-if="description" class="mt-1 text-sm text-muted-foreground max-w-sm">{{ description }}</p>
    <div v-if="actionText" class="mt-4">
      <slot name="action">
        <Button size="sm" @click="onAction">{{ actionText }}</Button>
      </slot>
    </div>
  </div>
</template>
