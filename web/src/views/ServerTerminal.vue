<script setup lang="ts">
import { ref, inject, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Unplug, Plus, Maximize2, Minimize2, Trash2, Terminal as TerminalIcon } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import TerminalEmulator from '@/components/common/TerminalEmulator.vue'
import * as serversApi from '@/api/modules/servers'
import type { Server } from '@/types/models'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = inject<any>('toast')!

const serverName = ref('')
const loading = ref(true)
const isFullscreen = ref(false)

// Multi-tab support
interface Tab {
  id: string
  name: string
  status: 'connecting' | 'connected' | 'disconnected' | 'error'
}

const tabs = ref<Tab[]>([])
const activeTabId = ref('')
let tabCounter = 0

// Quick commands
const quickCommands = [
  { label: 'top', command: 'top' },
  { label: 'htop', command: 'htop' },
  { label: 'df -h', command: 'df -h' },
  { label: 'free -h', command: 'free -h' },
  { label: 'ps aux', command: 'ps aux' },
  { label: 'docker ps', command: 'docker ps' },
  { label: 'netstat -tlnp', command: 'netstat -tlnp' },
  { label: 'uptime', command: 'uptime' },
]

function createTab() {
  tabCounter++
  const tab: Tab = {
    id: `tab-${tabCounter}`,
    name: `Terminal ${tabCounter}`,
    status: 'connecting',
  }
  tabs.value.push(tab)
  activeTabId.value = tab.id
}

function closeTab(tabId: string) {
  if (tabs.value.length <= 1) return
  const idx = tabs.value.findIndex((t) => t.id === tabId)
  tabs.value.splice(idx, 1)
  if (activeTabId.value === tabId) {
    activeTabId.value = tabs.value[Math.max(0, idx - 1)].id
  }
}

function handleStatusChange(tabId: string, status: Tab['status']) {
  const tab = tabs.value.find((t) => t.id === tabId)
  if (tab) tab.status = status
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

// Fetch server info
async function fetchServer() {
  loading.value = true
  try {
    const res = await serversApi.list({ page: 1, page_size: 1000 })
    if (res.data.status === 'success') {
      const found = (res.data.data as Server[]).find((s) => s.id === Number(props.id))
      if (found) {
        serverName.value = found.name
      } else {
        toast(t('serverTerminal.serverNotFound'), 'destructive')
        router.push('/servers')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('serverTerminal.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push(`/servers/${props.id}`)
}

// Keyboard shortcuts
function handleKeyDown(e: KeyboardEvent) {
  // Ctrl+Shift+T: new tab
  if (e.ctrlKey && e.shiftKey && e.key === 'T') {
    e.preventDefault()
    createTab()
  }
  // Ctrl+W: close tab
  if (e.ctrlKey && e.key === 'w') {
    e.preventDefault()
    closeTab(activeTabId.value)
  }
  // F11: toggle fullscreen
  if (e.key === 'F11') {
    e.preventDefault()
    toggleFullscreen()
  }
}

onMounted(() => {
  fetchServer()
  createTab()
  document.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <div
    class="flex flex-col bg-[#0a0a0a]"
    :class="isFullscreen ? 'fixed inset-0 z-50' : 'h-screen'"
  >
    <!-- Top toolbar -->
    <div class="flex items-center justify-between h-11 px-3 bg-[#111111] border-b border-[#222222] shrink-0">
      <div class="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          class="h-7 w-7 text-[#888888] hover:text-[#d4d4d4] hover:bg-[#222222]"
          @click="goBack"
        >
          <ArrowLeft class="w-4 h-4" />
        </Button>
        <div class="flex items-center gap-2">
          <TerminalIcon class="w-4 h-4 text-[#888888]" />
          <span class="text-sm font-medium text-[#d4d4d4] font-mono">
            {{ serverName || t('serverTerminal.title') }}
          </span>
        </div>
      </div>

      <div class="flex items-center gap-1">
        <!-- Quick commands dropdown -->
        <div class="relative group">
          <Button
            variant="ghost"
            size="sm"
            class="h-7 text-xs text-[#888888] hover:text-[#d4d4d4] hover:bg-[#222222]"
          >
            ⚡ {{ t('serverTerminal.quickCommands') }}
          </Button>
          <div class="absolute right-0 top-full mt-1 hidden group-hover:block z-50 bg-[#1a1a1a] border border-[#333333] rounded-lg shadow-xl py-1 min-w-[160px]">
            <button
              v-for="cmd in quickCommands"
              :key="cmd.label"
              class="w-full text-left px-3 py-1.5 text-xs text-[#d4d4d4] hover:bg-[#264f78] font-mono"
              @click="toast(cmd.label + ' — use terminal directly', 'default')"
            >
              {{ cmd.label }}
            </button>
          </div>
        </div>

        <Button
          variant="ghost"
          size="sm"
          class="h-7 text-xs text-[#888888] hover:text-[#d4d4d4] hover:bg-[#222222]"
          @click="toggleFullscreen"
        >
          <template #icon>
            <Minimize2 v-if="isFullscreen" class="w-3.5 h-3.5" />
            <Maximize2 v-else class="w-3.5 h-3.5" />
          </template>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          class="h-7 text-xs text-[#888888] hover:text-[#d4d4d4] hover:bg-[#222222]"
          @click="goBack"
        >
          {{ t('serverTerminal.back') }}
        </Button>
      </div>
    </div>

    <!-- Tab bar -->
    <div class="flex items-center h-9 bg-[#0d0d0d] border-b border-[#222222] shrink-0 overflow-x-auto">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="flex items-center gap-1.5 h-full px-3 text-xs cursor-pointer border-r border-[#222222] transition-colors"
        :class="tab.id === activeTabId
          ? 'bg-[#0a0a0a] text-[#d4d4d4] border-b-2 border-b-[#3b82f6]'
          : 'text-[#666666] hover:text-[#999999] hover:bg-[#151515]'"
        @click="activeTabId = tab.id"
      >
        <span
          class="inline-block w-1.5 h-1.5 rounded-full shrink-0"
          :style="{
            backgroundColor: tab.status === 'connected' ? '#22c55e' : tab.status === 'error' ? '#ef4444' : '#eab308'
          }"
        />
        <span class="font-mono">{{ tab.name }}</span>
        <button
          v-if="tabs.length > 1"
          class="ml-1 p-0.5 rounded hover:bg-[#333333] text-[#666666] hover:text-[#ef4444]"
          @click.stop="closeTab(tab.id)"
        >
          <Trash2 class="w-3 h-3" />
        </button>
      </div>
      <button
        class="flex items-center justify-center w-8 h-full text-[#666666] hover:text-[#d4d4d4] hover:bg-[#151515] transition-colors"
        title="New Tab (Ctrl+Shift+T)"
        @click="createTab"
      >
        <Plus class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- Terminal area -->
    <div class="flex-1 overflow-hidden relative">
      <template v-if="!loading">
        <div
          v-for="tab in tabs"
          :key="tab.id"
          class="absolute inset-0"
          :class="tab.id === activeTabId ? 'visible' : 'invisible'"
        >
          <TerminalEmulator
            :server-id="props.id"
            :on-status-change="(s) => handleStatusChange(tab.id, s)"
          />
        </div>
      </template>
      <div v-else class="flex items-center justify-center h-full">
        <span class="text-sm text-[#555555]">{{ t('serverTerminal.connecting') }}</span>
      </div>
    </div>

    <!-- Status bar -->
    <div class="flex items-center justify-between h-6 px-3 bg-[#111111] border-t border-[#222222] shrink-0 text-[10px] text-[#555555] font-mono">
      <div class="flex items-center gap-3">
        <span>{{ serverName }}</span>
        <span>SSH</span>
      </div>
      <div class="flex items-center gap-3">
        <span>{{ tabs.length }} tab(s)</span>
        <span>Ctrl+Shift+T: new tab</span>
        <span>Ctrl+W: close tab</span>
        <span>F11: fullscreen</span>
      </div>
    </div>
  </div>
</template>
