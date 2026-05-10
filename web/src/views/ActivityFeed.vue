<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useToast } from '@/composables/useToast'
import { useI18n } from 'vue-i18n'

import { Activity } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import { listEvents, getEventStats } from '@/api/modules/events'

const { t } = useI18n()
const { toast } = useToast()

interface EventItem {
  id: string
  event_type: string
  resource_type: string
  resource_id: string
  action: string
  message: string
  user_id: string
  tenant_id: string
  metadata: string
  created_at: string
}

const events = ref<EventItem[]>([])
const stats = ref<Record<string, number>>({})
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const typeFilter = ref('')

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

const typeColor = (type: string) => {
  const map: Record<string, string> = {
    deploy: 'bg-blue-500/10 text-blue-400',
    build: 'bg-purple-500/10 text-purple-400',
    server: 'bg-green-500/10 text-green-400',
    app: 'bg-amber-500/10 text-amber-400',
    user: 'bg-cyan-500/10 text-cyan-400',
    system: 'bg-gray-500/10 text-gray-400',
  }
  return map[type] || 'bg-gray-500/10 text-gray-400'
}

async function fetchEvents() {
  loading.value = true
  try {
    const res = await listEvents({ event_type: typeFilter.value || undefined, page: page.value, page_size: pageSize.value })
    if (res.data.status === 'success') {
      events.value = res.data.data.items || []
      total.value = res.data.data.total || 0
    }
  } catch { toast(t('activity.fetchFailed'), 'error') }
  finally { loading.value = false }
}

async function fetchStats() {
  try {
    const res = await getEventStats()
    if (res.data.status === 'success') stats.value = res.data.data.by_type || {}
  } catch { /* ignore */ }
}

function changePage(p: number) {
  page.value = p
  fetchEvents()
}

function filterByType(type: string) {
  typeFilter.value = type
  page.value = 1
  fetchEvents()
}

onMounted(() => { fetchEvents(); fetchStats() })
</script>

<template>
  <div>
    <PageHeader :title="t('activity.title')" :description="t('activity.description')" />

    <!-- Stats -->
    <div v-if="Object.keys(stats).length > 0" class="flex gap-3 flex-wrap mb-4">
      <button
        class="px-3 py-1.5 text-xs rounded-lg border transition-colors"
        :class="!typeFilter ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="filterByType('')"
      >
        {{ t('activity.allTypes') }} ({{ total }})
      </button>
      <button
        v-for="(count, type) in stats" :key="type"
        class="px-3 py-1.5 text-xs rounded-lg border transition-colors capitalize"
        :class="typeFilter === type ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="filterByType(type as string)"
      >
        {{ type }} ({{ count }})
      </button>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>
    <div v-else-if="events.length === 0">
      <EmptyState :icon="Activity" :title="t('activity.emptyTitle')" :description="t('activity.emptyDescription')" />
    </div>
    <div v-else class="bg-gray-900/50 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('activity.time') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('activity.type') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('activity.action') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('activity.message') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('activity.user') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in events" :key="e.id" class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors">
            <td class="px-4 py-3 text-gray-400 text-xs whitespace-nowrap">{{ new Date(e.created_at).toLocaleString() }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs capitalize" :class="typeColor(e.event_type)">{{ e.event_type }}</span>
            </td>
            <td class="px-4 py-3 text-gray-300 text-xs">{{ e.action }}</td>
            <td class="px-4 py-3 text-gray-400 text-xs max-w-md truncate">{{ e.message }}</td>
            <td class="px-4 py-3 text-gray-500 text-xs">{{ e.user_id || '-' }}</td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t border-gray-800">
        <span class="text-xs text-gray-500">{{ t('activity.pageInfo', { current: page, total: totalPages }) }}</span>
        <div class="flex gap-1">
          <Button variant="ghost" size="sm" :disabled="page <= 1" @click="changePage(page - 1)">←</Button>
          <Button variant="ghost" size="sm" :disabled="page >= totalPages" @click="changePage(page + 1)">→</Button>
        </div>
      </div>
    </div>
  </div>
</template>
