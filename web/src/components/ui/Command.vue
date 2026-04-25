<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { cn } from '@/lib/utils'
import { Search } from 'lucide-vue-next'

interface CommandItem {
  label: string
  icon?: any
  action?: () => void
  shortcut?: string
  group?: string
}

interface Props {
  open?: boolean
  items?: CommandItem[]
  placeholder?: string
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  items: () => [],
  placeholder: '输入命令或搜索...',
})

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const searchQuery = ref('')
const activeIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

const filteredItems = ref<CommandItem[]>([...props.items])

watch(() => props.items, (val) => {
  filteredItems.value = [...val]
})

watch(searchQuery, (query) => {
  if (!query.trim()) {
    filteredItems.value = [...props.items]
  } else {
    const lower = query.toLowerCase()
    filteredItems.value = props.items.filter(
      (item) => item.label.toLowerCase().includes(lower)
    )
  }
  activeIndex.value = 0
})

watch(() => props.open, async (val) => {
  if (val) {
    searchQuery.value = ''
    activeIndex.value = 0
    await nextTick()
    inputRef.value?.focus()
  }
})

function close() {
  emit('update:open', false)
}

function selectItem(item: CommandItem) {
  item.action?.()
  close()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    close()
    return
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    activeIndex.value = Math.min(activeIndex.value + 1, filteredItems.value.length - 1)
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
    return
  }
  if (event.key === 'Enter' && filteredItems.value[activeIndex.value]) {
    selectItem(filteredItems.value[activeIndex.value])
  }
}

function onGlobalKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
    event.preventDefault()
    emit('update:open', !props.open)
  }
}

onMounted(() => document.addEventListener('keydown', onGlobalKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', onGlobalKeydown))
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
        class="fixed inset-0 z-50 flex items-start justify-center pt-[20vh] bg-black/60"
        @click.self="close"
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
              'w-full max-w-lg rounded-lg border border-border bg-card shadow-2xl overflow-hidden',
              props.class
            )"
          >
            <div class="flex items-center border-b border-border px-3">
              <Search class="w-4 h-4 shrink-0 text-muted-foreground" />
              <input
                ref="inputRef"
                v-model="searchQuery"
                type="text"
                :placeholder="placeholder"
                class="flex-1 h-11 bg-transparent px-3 text-sm text-foreground outline-none placeholder:text-muted-foreground"
                @keydown="onKeydown"
              />
            </div>
            <div class="max-h-72 overflow-auto p-1">
              <div v-if="filteredItems.length === 0" class="py-6 text-center text-sm text-muted-foreground">
                未找到结果
              </div>
              <button
                v-for="(item, index) in filteredItems"
                :key="index"
                type="button"
                :class="cn(
                  'flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm cursor-pointer transition-colors duration-100',
                  index === activeIndex
                    ? 'bg-accent text-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                )"
                @click="selectItem(item)"
                @mouseenter="activeIndex = index"
              >
                <component v-if="item.icon" :is="item.icon" class="w-4 h-4 shrink-0" />
                <span class="flex-1 text-left">{{ item.label }}</span>
                <span v-if="item.shortcut" class="text-xs text-muted-foreground">{{ item.shortcut }}</span>
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
