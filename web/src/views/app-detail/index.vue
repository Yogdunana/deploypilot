<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeft, Rocket, Hammer, RotateCcw,
  Plus, Trash2,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Table from '@/components/ui/Table.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as appsApi from '@/api/modules/apps'
import * as deploymentsApi from '@/api/modules/deployments'
import { useWebSocket } from '@/composables/useWebSocket'
import type { App, DeploymentRecord } from '@/types/models'
import type { LogEntry } from '@/components/common/LogViewer.vue'

import OverviewTab from './OverviewTab.vue'
import DeployTab from './DeployTab.vue'
import EnvVarsTab from './EnvVarsTab.vue'
import LogsTab from './LogsTab.vue'
import RollbackDialog from './RollbackDialog.vue'

interface EnvItem {
  key: string
  value: string
  visible: boolean
  isNew?: boolean
}

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = useToast()

// State
const app = ref<App | null>(null)
const loading = ref(true)
const activeTab = ref('overview')
const deploying = ref(false)
const building = ref(false)

// Rollback dialog
const rollbackDialogOpen = ref(false)
const rollingBack = ref(false)

// Deployment history
const deployments = ref<DeploymentRecord[]>([])
const deploymentsLoading = ref(false)

// ===== Logs Tab State =====
const realtimeEnabled = ref(false)
const logs = ref<LogEntry[]>([])
const loadingHistory = ref(false)

const { connected: wsConnected, send, disconnect: wsDisconnect, connect: wsConnect } = useWebSocket({
  path: `/ws/logs/${props.id}`,
  autoConnect: false,
  onMessage(data: any) {
    if (data.type === 'log') {
      logs.value.push({
        timestamp: data.timestamp || new Date().toISOString(),
        data: data.data || '',
      })
    }
  },
})

function toggleRealtime(value: boolean) {
  realtimeEnabled.value = value
  if (value) {
    wsConnect()
  } else {
    wsDisconnect()
  }
}

async function loadHistory() {
  loadingHistory.value = true
  try {
    const res = await appsApi.getLogs(Number(props.id), 500)
    if (res.data.status === 'success') {
      const logText = res.data.data
      if (logText && typeof logText === 'string') {
        const lines = logText.split('\n').filter((line) => line.trim())
        const historyLogs: LogEntry[] = lines.map((line) => ({
          timestamp: new Date().toISOString(),
          data: line,
        }))
        logs.value = [...historyLogs, ...logs.value]
        toast(t('appDetail.historyLogsLoaded', { count: historyLogs.length }), 'success')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.loadHistoryFailed'), 'destructive')
  } finally {
    loadingHistory.value = false
  }
}

function clearLogs() {
  logs.value = []
}

// ===== Env Tab State =====
const envList = ref<EnvItem[]>([])
const envLoading = ref(true)
const envSaving = ref(false)

async function fetchEnv() {
  envLoading.value = true
  try {
    const res = await appsApi.getEnv(Number(props.id))
    if (res.data.status === 'success') {
      const envData = res.data.data || {}
      envList.value = Object.entries(envData).map(([key, value]) => ({
        key,
        value: value as string,
        visible: false,
      }))
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.envLoadFailed'), 'destructive')
  } finally {
    envLoading.value = false
  }
}

function addVariable() {
  envList.value.push({ key: '', value: '', visible: true, isNew: true })
}

function removeVariable(index: number) {
  envList.value.splice(index, 1)
}

function toggleVisibility(index: number) {
  envList.value[index].visible = !envList.value[index].visible
}

async function saveEnv() {
  const emptyKeys = envList.value.filter((item) => !item.key.trim())
  if (emptyKeys.length > 0) {
    toast(t('appDetail.emptyVarName'), 'destructive')
    return
  }
  const keys = envList.value.map((item) => item.key.trim())
  const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index)
  if (duplicates.length > 0) {
    toast(t('appDetail.duplicateVarName', { names: duplicates.join(', ') }), 'destructive')
    return
  }

  envSaving.value = true
  try {
    const envObject: Record<string, string> = {}
    envList.value.forEach((item) => {
      if (item.key.trim()) {
        envObject[item.key.trim()] = item.value
      }
    })
    await appsApi.updateEnv(Number(props.id), { env_vars: JSON.stringify(envObject) })
    toast(t('appDetail.envSaved'), 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.envSaveFailed'), 'destructive')
  } finally {
    envSaving.value = false
  }
}

// ===== Backups Tab State =====
interface Backup {
  id: string
  app_id: number
  created_at: string
  size?: string
}

const backups = ref<Backup[]>([])
const backupsLoading = ref(true)
const creating = ref(false)
const restoring = ref(false)
const deleting = ref(false)

const restoreDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const selectedBackup = ref<Backup | null>(null)

const backupColumns = computed(() => [
  { key: 'id', label: t('appDetail.backupId') },
  { key: 'app_id', label: t('appDetail.appId') },
  { key: 'created_at', label: t('appDetail.createdAt') },
  { key: 'actions', label: t('appDetail.actions'), width: '160px' },
])

async function fetchBackups() {
  backupsLoading.value = true
  try {
    const res = await appsApi.listBackups(Number(props.id))
    if (res.data.status === 'success') {
      backups.value = res.data.data || []
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.backupFetchFailed'), 'destructive')
  } finally {
    backupsLoading.value = false
  }
}

async function createBackup() {
  creating.value = true
  try {
    await appsApi.backup(Number(props.id))
    toast(t('appDetail.backupCreated'), 'success')
    fetchBackups()
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.backupCreateFailed'), 'destructive')
  } finally {
    creating.value = false
  }
}

function openRestoreDialog(backup: any) {
  selectedBackup.value = backup
  restoreDialogOpen.value = true
}

async function confirmRestore() {
  if (!selectedBackup.value) return
  restoring.value = true
  try {
    await appsApi.restore(Number(props.id), { backup_id: selectedBackup.value.id })
    toast(t('appDetail.backupRestored'), 'success')
    restoreDialogOpen.value = false
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.backupRestoreFailed'), 'destructive')
  } finally {
    restoring.value = false
  }
}

function openDeleteDialog(backup: any) {
  selectedBackup.value = backup
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!selectedBackup.value) return
  deleting.value = true
  try {
    await appsApi.deleteBackup(Number(props.id), selectedBackup.value.id)
    toast(t('appDetail.backupDeleted'), 'success')
    deleteDialogOpen.value = false
    fetchBackups()
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.backupDeleteFailed'), 'destructive')
  } finally {
    deleting.value = false
  }
}

// Tabs
const detailTabs = computed(() => [
  { key: 'overview', label: t('appDetail.overview') },
  { key: 'logs', label: t('appDetail.logs') },
  { key: 'env', label: t('appDetail.envVars') },
  { key: 'backups', label: t('appDetail.backups') },
  { key: 'deployments', label: t('appDetail.deploymentHistory') },
])

// Fetch app detail
async function fetchApp() {
  loading.value = true
  try {
    const res = await appsApi.get(Number(props.id))
    if (res.data.status === 'success') {
      app.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Fetch deployment history
async function fetchDeployments() {
  deploymentsLoading.value = true
  try {
    const res = await deploymentsApi.list(Number(props.id))
    if (res.data.status === 'success') {
      deployments.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.fetchDeployFailed'), 'destructive')
  } finally {
    deploymentsLoading.value = false
  }
}

// Deploy
async function handleDeploy() {
  if (!app.value) return
  deploying.value = true
  try {
    await appsApi.deploy(app.value.id)
    toast(t('appDetail.deployTriggered'), 'success')
    fetchApp()
    if (activeTab.value === 'deployments') fetchDeployments()
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.deployFailed'), 'destructive')
  } finally {
    deploying.value = false
  }
}

// Build
async function handleBuild() {
  if (!app.value) return
  building.value = true
  try {
    await appsApi.build(app.value.id, {})
    toast(t('appDetail.buildTriggered'), 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.buildFailed'), 'destructive')
  } finally {
    building.value = false
  }
}

// Rollback
function openRollbackDialog() {
  rollbackDialogOpen.value = true
}

async function confirmRollback() {
  if (!app.value) return
  rollingBack.value = true
  try {
    await appsApi.rollback(app.value.id, { version: app.value.current_version })
    toast(t('appDetail.rollbackTriggered'), 'success')
    fetchApp()
    if (activeTab.value === 'deployments') fetchDeployments()
  } catch (err: any) {
    toast(err.response?.data?.message || t('appDetail.rollbackFailed'), 'destructive')
  } finally {
    rollingBack.value = false
  }
}

// Watch tab changes to load data lazily
watch(activeTab, (val) => {
  if (val === 'deployments' && deployments.value.length === 0) {
    fetchDeployments()
  }
  if (val === 'env' && envList.value.length === 0 && !envLoading.value) {
    fetchEnv()
  }
  if (val === 'backups' && backups.value.length === 0 && !backupsLoading.value) {
    fetchBackups()
  }
})

// Cleanup WebSocket on unmount
onBeforeUnmount(() => {
  wsDisconnect()
})

onMounted(() => {
  fetchApp()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader>
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push('/apps')">
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <div>
            <div class="flex items-center gap-2">
              <h1 class="text-xl font-semibold text-foreground">
                {{ app?.name || t('appDetail.loading') }}
              </h1>
              <StatusBadge v-if="app" :status="app.status" />
            </div>
            <p v-if="app?.repo_url" class="mt-0.5 text-sm text-muted-foreground">
              {{ app.repo_url }}
            </p>
          </div>
        </div>
      </template>
      <template #actions>
        <Button :loading="deploying" @click="handleDeploy">
          <template #icon><Rocket class="w-4 h-4" /></template>
          {{ t('appDetail.deploy') }}
        </Button>
        <Button variant="outline" :loading="building" @click="handleBuild">
          <template #icon><Hammer class="w-4 h-4" /></template>
          {{ t('appDetail.build') }}
        </Button>
        <Button variant="outline" @click="openRollbackDialog">
          <template #icon><RotateCcw class="w-4 h-4" /></template>
          {{ t('appDetail.rollback') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-4">
      <Skeleton class="h-10 w-96" />
      <div class="grid grid-cols-2 gap-4">
        <Skeleton v-for="i in 6" :key="i" class="h-24" />
      </div>
    </div>

    <!-- Tabs -->
    <template v-else-if="app">
      <Tabs v-model="activeTab" :tabs="detailTabs" />

      <!-- Overview Tab -->
      <OverviewTab v-if="activeTab === 'overview'" :app="app" />

      <!-- Logs Tab -->
      <LogsTab
        v-else-if="activeTab === 'logs'"
        :logs="logs"
        :realtime-enabled="realtimeEnabled"
        :ws-connected="wsConnected"
        :loading-history="loadingHistory"
        @update:realtime-enabled="toggleRealtime"
        @load-history="loadHistory"
        @clear-logs="clearLogs"
      />

      <!-- Environment Variables Tab -->
      <EnvVarsTab
        v-else-if="activeTab === 'env'"
        :env-list="envList"
        :loading="envLoading"
        :saving="envSaving"
        @add="addVariable"
        @remove="removeVariable"
        @toggle-visibility="toggleVisibility"
        @save="saveEnv"
      />

      <!-- Backups Tab -->
      <div v-else-if="activeTab === 'backups'" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">
            {{ t('appDetail.totalBackups', { count: backups.length }) }}
          </p>
          <Button :loading="creating" size="sm" @click="createBackup">
            <template #icon><Plus class="w-4 h-4" /></template>
            {{ t('appDetail.createBackup') }}
          </Button>
        </div>

        <Table
          :columns="backupColumns"
          :data="backups"
          :loading="backupsLoading"
        >
          <template #cell-id="{ row }">
            <span class="font-mono text-sm text-foreground">{{ row.id }}</span>
          </template>
          <template #cell-app_id="{ row }">
            <span class="text-sm text-muted-foreground">{{ row.app_id }}</span>
          </template>
          <template #cell-created_at="{ row }">
            <RelativeTime :date="row.created_at" />
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                class="h-7 text-xs"
                @click="openRestoreDialog(row)"
              >
                <template #icon><RotateCcw class="w-3.5 h-3.5" /></template>
                {{ t('appDetail.restore') }}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                class="h-7 text-xs text-muted-foreground hover:text-destructive"
                @click="openDeleteDialog(row)"
              >
                <template #icon><Trash2 class="w-3.5 h-3.5" /></template>
                {{ t('appDetail.delete') }}
              </Button>
            </div>
          </template>
        </Table>
      </div>

      <!-- Deployment History Tab -->
      <DeployTab
        v-else-if="activeTab === 'deployments'"
        :deployments="deployments"
        :loading="deploymentsLoading"
      />
    </template>

    <!-- Rollback confirmation dialog -->
    <RollbackDialog
      v-model:open="rollbackDialogOpen"
      :app-name="app?.name || ''"
      :current-version="app?.current_version || ''"
      @confirm="confirmRollback"
    />

    <!-- Restore backup dialog -->
    <AlertDialog
      v-model:open="restoreDialogOpen"
      :title="t('appDetail.restoreConfirm')"
      :description="t('appDetail.restoreConfirmDesc', { id: selectedBackup?.id || '' })"
      :confirm-text="t('appDetail.confirmRestore')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmRestore"
    />

    <!-- Delete backup dialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('appDetail.deleteBackupConfirm')"
      :description="t('appDetail.deleteBackupConfirmDesc', { id: selectedBackup?.id || '' })"
      :confirm-text="t('appDetail.confirmDelete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
