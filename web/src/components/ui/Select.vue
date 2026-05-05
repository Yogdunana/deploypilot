<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { cn } from '@/lib/utils'
import { ChevronDown } from 'lucide-vue-next'

const { t } = useI18n()

interface SelectOption {
  label: string
  value: string | number
}

interface Props {
  modelValue?: string | number
  options?: SelectOption[]
  placeholder?: string
  disabled?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  options: () => [],
  placeholder: 'Please select',
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)

const selectedLabel = computed(() => {
  const opt = props.options.find((o) => o.value === props.modelValue)
  return opt ? opt.label : ''
})

function toggle() {
  if (!props.disabled) {
    isOpen.value = !isOpen.value
  }
}

function select(option: SelectOption) {
  emit('update:modelValue', option.value)
  isOpen.value = false
}

function onClickOutside(event: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', onClickOutside))
</script>

<template>
  <div ref="containerRef" :class="cn('relative', props.class)">
    <button
      type="button"
      :class="cn(
        'flex h-9 w-full items-center justify-between rounded-md border border-border bg-card px-3 py-2 text-sm text-foreground shadow-sm transition-colors duration-150',
        'hover:bg-card-hover',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary',
        'disabled:cursor-not-allowed disabled:opacity-50',
        !selectedLabel && 'text-muted-foreground'
      )"
      :disabled="disabled"
      @click="toggle"
    >
      <span>{{ selectedLabel || placeholder }}</span>
      <ChevronDown class="w-4 h-4 opacity-50 shrink-0" />
    </button>
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
        class="absolute z-50 mt-1 w-full min-w-[8rem] overflow-hidden rounded-md border border-border bg-card shadow-lg"
      >
        <div class="p-1 max-h-60 overflow-auto">
          <button
            v-for="option in options"
            :key="option.value"
            type="button"
            :class="cn(
              'flex w-full items-center rounded-sm px-2 py-1.5 text-sm cursor-pointer transition-colors duration-100',
              option.value === modelValue
                ? 'bg-accent text-foreground'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'
            )"
            @click="select(option)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>
