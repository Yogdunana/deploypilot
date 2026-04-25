<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Unplug } from 'lucide-vue-next'
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
const isConnected = ref(false)
const terminalRef = ref<InstanceType<typeof TerminalEmulator> | null>(null)

// 获取服务器信息
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

function handleDisconnect() {
  terminalRef.value?.disconnect()
  isConnected.value = false
  toast(t('serverTerminal.disconnectedMsg'), 'default')
}

function handleReconnect() {
  terminalRef.value?.connect()
  isConnected.value = true
}

function goBack() {
  router.push(`/servers/${props.id}`)
}

onMounted(() => {
  fetchServer()
})
</script>

<template>
  <div class="flex flex-col h-screen bg-[#0a0a0a]">
    <!-- 顶部工具栏 -->
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
          <span class="text-sm font-medium text-[#d4d4d4] font-mono">
            {{ serverName || t('serverTerminal.title') }}
          </span>
          <span
            class="inline-block w-2 h-2 rounded-full"
            :style="isConnected ? { backgroundColor: '#22c55e', boxShadow: '0 0 6px rgba(34,197,94,0.5)' } : { backgroundColor: '#ef4444' }"
          />
        </div>
      </div>

      <div class="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          class="h-7 text-xs text-[#888888] hover:text-[#d4d4d4] hover:bg-[#222222]"
          @click="handleReconnect"
        >
          {{ t('serverTerminal.reconnect') }}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          class="h-7 text-xs text-[#888888] hover:text-red-400 hover:bg-[#222222]"
          @click="handleDisconnect"
        >
          <template #icon><Unplug class="w-3.5 h-3.5" /></template>
          {{ t('serverTerminal.disconnect') }}
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

    <!-- 终端区域 -->
    <div class="flex-1 overflow-hidden">
      <TerminalEmulator
        v-if="!loading"
        ref="terminalRef"
        :server-id="props.id"
      />
      <div v-else class="flex items-center justify-center h-full">
        <span class="text-sm text-[#555555]">{{ t('serverTerminal.connecting') }}</span>
      </div>
    </div>
  </div>
</template>
