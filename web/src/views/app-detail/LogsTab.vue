<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Radio, History, Trash2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Switch from '@/components/ui/Switch.vue'
import LogViewer from '@/components/common/LogViewer.vue'
import type { LogEntry } from '@/components/common/LogViewer.vue'

const props = defineProps<{
  logs: LogEntry[]
  realtimeEnabled: boolean
  wsConnected: boolean
  loadingHistory: boolean
}>()

const emit = defineEmits<{
  (e: 'update:realtime-enabled', value: boolean): void
  (e: 'load-history'): void
  (e: 'clear-logs'): void
}>()

const { t } = useI18n()
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-2">
        <Radio class="w-4 h-4 text-muted-foreground" />
        <span class="text-sm text-muted-foreground">{{ t('appDetail.realtimeLogs') }}</span>
        <Switch :model-value="realtimeEnabled" @update:model-value="emit('update:realtime-enabled', $event)" />
      </div>
      <div class="flex-1" />
      <Button variant="outline" size="sm" :loading="loadingHistory" @click="emit('load-history')">
        <template #icon><History class="w-4 h-4" /></template>
        {{ t('appDetail.loadHistoryLogs') }}
      </Button>
      <Button variant="outline" size="sm" @click="emit('clear-logs')">
        <template #icon><Trash2 class="w-4 h-4" /></template>
        {{ t('appDetail.clear') }}
      </Button>
    </div>

    <div class="h-[calc(100vh-280px)] min-h-[400px]">
      <LogViewer
        :logs="logs"
        :connected="wsConnected"
        :auto-scroll="true"
        @clear="emit('clear-logs')"
      />
    </div>
  </div>
</template>
