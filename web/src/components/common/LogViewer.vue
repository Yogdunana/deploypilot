<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { Search, Lock, Unlock, Trash2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'

export interface LogEntry {
  timestamp: string
  data: string
}

interface Props {
  logs: LogEntry[]
  autoScroll?: boolean
  connected?: boolean
  maxLogs?: number
}

const props = withDefaults(defineProps<Props>(), {
  autoScroll: true,
  connected: false,
  maxLogs: 1000,
})

const emit = defineEmits<{
  clear: []
}>()

const searchQuery = ref('')
const scrollLocked = ref(false)
const containerRef = ref<HTMLElement | null>(null)

// 限制日志数量
const limitedLogs = computed(() => {
  if (props.logs.length <= props.maxLogs) return props.logs
  return props.logs.slice(-props.maxLogs)
})

// 过滤日志
const filteredLogs = computed(() => {
  if (!searchQuery.value) return limitedLogs.value
  const query = searchQuery.value.toLowerCase()
  return limitedLogs.value.filter((log) => log.data.toLowerCase().includes(query))
})

// 高亮搜索匹配
function highlightMatch(text: string): string {
  if (!searchQuery.value) return escapeHtml(text)
  const escaped = escapeHtml(text)
  const query = escapeHtml(searchQuery.value)
  const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  return escaped.replace(regex, '<mark class="bg-yellow-500/30 text-yellow-200 rounded px-0.5">$1</mark>')
}

function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

// 自动滚动
watch(
  () => filteredLogs.value.length,
  async () => {
    if (props.autoScroll && !scrollLocked.value) {
      await nextTick()
      scrollToBottom()
    }
  }
)

function scrollToBottom() {
  if (containerRef.value) {
    containerRef.value.scrollTop = containerRef.value.scrollHeight
  }
}

function handleScroll() {
  if (!containerRef.value) return
  const { scrollTop, scrollHeight, clientHeight } = containerRef.value
  // 如果用户向上滚动超过 50px，则锁定滚动
  scrollLocked.value = scrollHeight - scrollTop - clientHeight > 50
}

function toggleScrollLock() {
  scrollLocked.value = !scrollLocked.value
  if (!scrollLocked.value) {
    nextTick(() => scrollToBottom())
  }
}

function formatTimestamp(ts: string): string {
  if (!ts) return ''
  try {
    const date = new Date(ts)
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    } as Intl.DateTimeFormatOptions & { fractionalSecondDigits: number })
  } catch {
    return ts
  }
}

onMounted(() => {
  if (props.autoScroll) {
    nextTick(() => scrollToBottom())
  }
})
</script>

<template>
  <div class="flex flex-col h-full rounded-lg border border-border overflow-hidden">
    <!-- 工具栏 -->
    <div class="flex items-center gap-2 px-3 py-2 bg-[#111111] border-b border-[#222222]">
      <!-- 连接状态指示灯 -->
      <div class="flex items-center gap-1.5">
        <span
          class="inline-block w-2 h-2 rounded-full"
          :class="connected ? 'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.5)]' : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.5)]'"
        />
        <span class="text-xs text-[#888888]">{{ connected ? '已连接' : '已断开' }}</span>
      </div>

      <div class="flex-1" />

      <!-- 搜索框 -->
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#666666]" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索日志..."
          class="h-7 w-48 pl-7 pr-2 rounded-md border border-[#333333] bg-[#1a1a1a] text-xs text-[#d4d4d4] placeholder:text-[#555555] focus:outline-none focus:border-[#444444] focus:ring-1 focus:ring-[#444444]"
        />
      </div>

      <!-- 滚动锁定 -->
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7 text-[#888888] hover:text-[#d4d4d4] hover:bg-[#222222]"
        :title="scrollLocked ? '解锁滚动' : '锁定滚动'"
        @click="toggleScrollLock"
      >
        <Lock v-if="scrollLocked" class="w-3.5 h-3.5" />
        <Unlock v-else class="w-3.5 h-3.5" />
      </Button>

      <!-- 清空 -->
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7 text-[#888888] hover:text-red-400 hover:bg-[#222222]"
        title="清空日志"
        @click="emit('clear')"
      >
        <Trash2 class="w-3.5 h-3.5" />
      </Button>

      <!-- 日志计数 -->
      <span class="text-xs text-[#555555]">
        {{ filteredLogs.length }} 条
      </span>
    </div>

    <!-- 日志内容 -->
    <div
      ref="containerRef"
      class="flex-1 overflow-auto bg-[#0a0a0a] p-3"
      @scroll="handleScroll"
    >
      <!-- 空状态 -->
      <div v-if="filteredLogs.length === 0" class="flex items-center justify-center h-full">
        <p class="text-sm text-[#555555]">暂无日志</p>
      </div>

      <!-- 日志行 -->
      <div
        v-for="(log, index) in filteredLogs"
        :key="index"
        class="flex gap-3 py-0.5 hover:bg-[#111111] rounded px-1 -mx-1 font-mono text-[13px] leading-5"
      >
        <span class="text-[#555555] shrink-0 select-none w-20 text-right">
          {{ formatTimestamp(log.timestamp) }}
        </span>
        <span class="text-[#d4d4d4] whitespace-pre-wrap break-all" v-html="highlightMatch(log.data)" />
      </div>
    </div>
  </div>
</template>
