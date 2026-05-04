<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Radio, History, Trash2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Switch from '@/components/ui/Switch.vue'
import LogViewer from '@/components/common/LogViewer.vue'
import type { LogEntry } from '@/components/common/LogViewer.vue'

const { t } = useI18n()

const props = defineProps<{
  logs: LogEntry[]
  realtimeEnabled: boolean
  wsConnected: boolean
  loadingHistory: boolean
}>()

const emit = defineEmits<{
  'update:realtimeEnabled': [value: boolean]
  loadHistory: []
  clearLogs: []
}>()

function toggleRealtime(value: boolean) {
  emit('update:realtimeEnabled', value)
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-2">
        <Radio class="w-4 h-4 text-muted-foreground" />
        <span class="text-sm text-muted-foreground">{{ t('appDetail.realtimeLogs') }}</span>
        <Switch :model-value="realtimeEnabled" @update:model-value="toggleRealtime" />
      </div>
      <div class="flex-1" />
      <Button variant="outline" size="sm" :loading="loadingHistory" @click="emit('loadHistory')">
        <template #icon><History class="w-4 h-4" /></template>
        {{ t('appDetail.loadHistoryLogs') }}
      </Button>
      <Button variant="outline" size="sm" @click="emit('clearLogs')">
        <template #icon><Trash2 class="w-4 h-4" /></template>
        {{ t('appDetail.clear') }}
      </Button>
    </div>

    <div class="h-[calc(100vh-280px)] min-h-[400px]">
      <LogViewer
        :logs="logs"
        :connected="wsConnected"
        :auto-scroll="true"
        @clear="emit('clearLogs')"
      />
    </div>
  </div>
</template>
