<script setup lang="ts">
import { cn } from '@/lib/utils'

interface TabItem {
  key: string
  label: string
  icon?: any
}

interface Props {
  modelValue?: string
  tabs?: TabItem[]
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  tabs: () => [],
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function selectTab(key: string) {
  emit('update:modelValue', key)
}
</script>

<template>
  <div :class="cn('w-full', props.class)">
    <div class="inline-flex h-10 items-center justify-center rounded-md bg-accent/50 p-1 text-muted-foreground">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        :class="cn(
          'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium transition-all duration-150 cursor-pointer',
          modelValue === tab.key
            ? 'bg-card text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground'
        )"
        @click="selectTab(tab.key)"
      >
        <component v-if="tab.icon" :is="tab.icon" class="w-4 h-4" />
        {{ tab.label }}
      </button>
    </div>
  </div>
</template>
