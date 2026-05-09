<script setup lang="ts">
import { ref, onMounted, inject } from 'vue'
import PageHeader from '@/components/common/PageHeader.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import {
  RefreshCw,
  Download,
  Wifi,
  WifiOff,
  Loader2,
  BarChart3,
  Tag,
  Clock,
  Database,
} from 'lucide-vue-next'
import {
  getGrafanaStatus,
  testGrafanaConnection,
  syncGrafana,
  exportGrafana,
} from '@/api/modules/grafana'
import type { GrafanaStatus } from '@/api/modules/grafana'

const { toast } = inject<any>('toast')!

const status = ref<GrafanaStatus | null>(null)
const loading = ref(false)
const testing = ref(false)
const syncing = ref(false)
const exporting = ref(false)

async function fetchStatus() {
  loading.value = true
  try {
    const res = await getGrafanaStatus()
    if (res.data.status === 'success') {
      status.value = res.data.data
    }
  } catch (e: any) {
    toast(e?.message || 'Failed to load Grafana status', 'destructive')
  } finally {
    loading.value = false
  }
}

async function handleTestConnection() {
  testing.value = true
  try {
    const res = await testGrafanaConnection()
    if (res.data.status === 'success') {
      toast(`Grafana connected successfully (v${res.data.data.version})`, 'success')
      fetchStatus()
    } else {
      toast(res.data.message || 'Connection test failed', 'destructive')
    }
  } catch (e: any) {
    toast(e?.message || 'Connection test failed', 'destructive')
  } finally {
    testing.value = false
  }
}

async function handleSyncAll() {
  syncing.value = true
  try {
    const res = await syncGrafana()
    if (res.data.status === 'success') {
      toast(`Synced ${res.data.data.synced} dashboards`, 'success')
      fetchStatus()
    } else {
      toast(res.data.message || 'Sync failed', 'destructive')
    }
  } catch (e: any) {
    toast(e?.message || 'Sync failed', 'destructive')
  } finally {
    syncing.value = false
  }
}

async function handleExport() {
  exporting.value = true
  try {
    const res = await exportGrafana()
    if (res.data.status === 'success') {
      const exportData = res.data.data
      const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'grafana-templates.json'
      a.click()
      URL.revokeObjectURL(url)
      toast('Templates exported successfully', 'success')
    } else {
      toast(res.data.message || 'Export failed', 'destructive')
    }
  } catch (e: any) {
    toast(e?.message || 'Export failed', 'destructive')
  } finally {
    exporting.value = false
  }
}

function formatTimestamp(ts?: string): string {
  if (!ts) return 'Never'
  return ts
}

onMounted(fetchStatus)
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Grafana Settings" description="Manage Grafana integration and dashboard synchronization">
      <template #actions>
        <Button variant="outline" :loading="testing" :disabled="loading" @click="handleTestConnection">
          <template #icon>
            <Wifi class="w-4 h-4" />
          </template>
          Test Connection
        </Button>
        <Button :loading="syncing" :disabled="loading" @click="handleSyncAll">
          <template #icon>
            <RefreshCw class="w-4 h-4" />
          </template>
          Sync All
        </Button>
        <Button variant="outline" :loading="exporting" :disabled="loading" @click="handleExport">
          <template #icon>
            <Download class="w-4 h-4" />
          </template>
          Export Templates
        </Button>
      </template>
    </PageHeader>

    <!-- Loading State -->
    <div v-if="loading" class="space-y-4">
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm space-y-4">
        <Skeleton class="h-6 w-1/3" />
        <Skeleton class="h-4 w-2/3" />
        <Skeleton class="h-4 w-1/2" />
      </div>
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm space-y-4">
        <Skeleton class="h-6 w-1/3" />
        <Skeleton class="h-4 w-2/3" />
      </div>
    </div>

    <template v-else-if="status">
      <!-- Connection Status Card -->
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
        <h3 class="text-base font-semibold mb-4">Connection Status</h3>
        <div class="flex items-center gap-3 mb-4">
          <span
            class="inline-flex items-center gap-2 px-3 py-1.5 text-sm rounded-full font-medium"
            :class="status.connected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'"
          >
            <component :is="status.connected ? Wifi : WifiOff" class="w-4 h-4" />
            {{ status.connected ? 'Connected' : 'Disconnected' }}
          </span>
          <span v-if="status.version" class="text-sm text-muted-foreground">
            Grafana v{{ status.version }}
          </span>
        </div>
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div>
            <label class="block text-sm font-medium mb-1">Grafana URL</label>
            <p class="text-sm font-mono text-muted-foreground truncate" :title="status.url">
              {{ status.url || 'Not configured' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">Configure in server config.yaml</p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Datasource UID</label>
            <p class="text-sm font-mono text-muted-foreground truncate" :title="status.datasource_uid">
              {{ status.datasource_uid || 'Not configured' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">Auto-populated after first sync</p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Last Sync</label>
            <p class="text-sm text-muted-foreground">
              <template v-if="status.last_sync">
                <RelativeTime :date="status.last_sync" />
              </template>
              <template v-else>
                Never
              </template>
            </p>
          </div>
        </div>
      </div>

      <!-- Annotations Status Card -->
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
        <h3 class="text-base font-semibold mb-4">Annotations</h3>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">Deployment Annotations</p>
            <p class="text-xs text-gray-500 mt-0.5">Send deployment events as annotations to Grafana dashboards for correlation with metrics</p>
          </div>
          <span
            class="inline-flex items-center gap-1.5 px-2.5 py-0.5 text-xs rounded-full font-medium"
            :class="status.annotations_enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'"
          >
            <span class="w-1.5 h-1.5 rounded-full" :class="status.annotations_enabled ? 'bg-green-500' : 'bg-gray-400'" />
            {{ status.annotations_enabled ? 'Enabled' : 'Disabled' }}
          </span>
        </div>
      </div>

      <!-- Configuration Info Card -->
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
        <h3 class="text-base font-semibold mb-4">Configuration</h3>
        <p class="text-sm text-muted-foreground mb-4">
          Grafana connection settings are configured on the server side in <code class="px-1.5 py-0.5 bg-gray-100 rounded text-xs font-mono">config.yaml</code>. Use the "Test Connection" button above to verify the configuration.
        </p>
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
            <BarChart3 class="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium">Grafana URL</p>
              <p class="text-xs text-gray-500 mt-0.5">Set <code class="font-mono">grafana.url</code> in config.yaml</p>
            </div>
          </div>
          <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
            <Tag class="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium">API Key</p>
              <p class="text-xs text-gray-500 mt-0.5">Set <code class="font-mono">grafana.api_key</code> in config.yaml</p>
            </div>
          </div>
          <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
            <Database class="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium">Datasource</p>
              <p class="text-xs text-gray-500 mt-0.5">Set <code class="font-mono">grafana.datasource_uid</code> in config.yaml</p>
            </div>
          </div>
          <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
            <Clock class="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium">Admin Credentials</p>
              <p class="text-xs text-gray-500 mt-0.5">Set <code class="font-mono">grafana.admin_user</code> and <code class="font-mono">grafana.admin_password</code> in config.yaml</p>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Error State - no status loaded -->
    <div v-else class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-base font-semibold">Unable to Load Grafana Status</h3>
          <p class="text-sm text-muted-foreground mt-1">The Grafana integration may not be configured on the server.</p>
        </div>
        <Button variant="outline" @click="fetchStatus">
          <template #icon>
            <RefreshCw class="w-4 h-4" />
          </template>
          Retry
        </Button>
      </div>
    </div>
  </div>
</template>
