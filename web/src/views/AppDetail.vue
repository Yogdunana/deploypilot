<script setup lang="ts">
import { ref, computed, inject, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeft, Rocket, Hammer, RotateCcw, Shield,
  Clock, GitBranch, Globe, Server, Layers,
  Radio, History, Trash2, Plus, Eye, EyeOff, Save,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import LogViewer from '@/components/common/LogViewer.vue'
import DeployProgress from '@/components/common/DeployProgress.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Card from '@/components/ui/Card.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Table from '@/components/ui/Table.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Input from '@/components/ui/Input.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Switch from '@/components/ui/Switch.vue'
import * as appsApi from '@/api/modules/apps'
import * as deploymentsApi from '@/api/modules/deployments'
import { useWebSocket } from '@/composables/useWebSocket'
import type { App, DeploymentRecord } from '@/types/models'
import type { LogEntry } from '@/components/common/LogViewer.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = inject<any>('toast')!

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

function toggleRealtime() {
  if (realtimeEnabled.value) {
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
interface EnvItem {
  key: string
  value: string
  visible: boolean
  isNew?: boolean
}

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

// Deployment history columns
const deploymentColumns = computed(() => [
  { key: 'container_name', label: t('appDetail.deployContainerName') },
  { key: 'image', label: t('appDetail.image') },
  { key: 'status', label: t('appDetail.status') },
  { key: 'error_message', label: t('appDetail.errorMessage') },
  { key: 'created_at', label: t('appDetail.time') },
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

// Info items for overview
const infoItems = computed(() => {
  if (!app.value) return []
  return [
    { label: t('appDetail.appName'), value: app.value.name, icon: Layers },
    { label: t('appDetail.repoUrl'), value: app.value.repo_url, icon: GitBranch },
    { label: t('appDetail.branch'), value: app.value.branch, icon: GitBranch },
    { label: t('appDetail.stack'), value: app.value.tech_stack, icon: Layers },
    { label: t('appDetail.domain'), value: app.value.domain, icon: Globe },
    { label: t('appDetail.server'), value: String(app.value.server_id), icon: Server },
    { label: t('appDetail.status'), value: app.value.status, icon: Shield, isStatus: true },
    { label: t('appDetail.currentVersion'), value: app.value.current_version || '-', icon: Layers },
    { label: t('appDetail.containerName'), value: app.value.container_name || '-', icon: Server },
    { label: t('appDetail.createdAt'), value: app.value.created_at, icon: Clock, isTime: true },
    { label: t('appDetail.updatedAt'), value: app.value.updated_at, icon: Clock, isTime: true },
  ]
})

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
      <div v-if="activeTab === 'overview'" class="space-y-4">
        <Card>
          <template #header>
            <h3 class="text-sm font-medium text-foreground">{{ t('appDetail.basicInfo') }}</h3>
          </template>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-0">
            <div
              v-for="(item, index) in infoItems"
              :key="item.label"
              class="flex items-start gap-3 py-3"
              :class="index % 2 === 0 ? 'pr-4' : 'pr-4 md:border-l md:border-border md:pl-4'"
            >
              <component :is="item.icon" class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div class="min-w-0">
                <p class="text-xs text-muted-foreground">{{ item.label }}</p>
                <p v-if="item.isStatus" class="text-sm text-foreground mt-0.5">
                  <StatusBadge :status="item.value" />
                </p>
                <p v-else-if="item.label === t('appDetail.stack')" class="text-sm text-foreground mt-0.5">
                  <Badge v-if="item.value" variant="secondary">{{ item.value }}</Badge>
                  <span v-else>-</span>
                </p>
                <p v-else-if="item.isTime" class="text-sm text-foreground mt-0.5">
                  <RelativeTime :date="item.value" />
                </p>
                <p v-else class="text-sm text-foreground mt-0.5 truncate">{{ item.value || '-' }}</p>
              </div>
            </div>
          </div>
        </Card>

        <!-- Resource limits -->
        <Card>
          <template #header>
            <h3 class="text-sm font-medium text-foreground">{{ t('appDetail.resourceLimits') }}</h3>
          </template>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <p class="text-xs text-muted-foreground">{{ t('appDetail.memory') }}</p>
              <p class="text-sm text-foreground mt-0.5">{{ app.resource_limits?.memory || '-' }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">{{ t('appDetail.cpu') }}</p>
              <p class="text-sm text-foreground mt-0.5">{{ app.resource_limits?.cpu || '-' }}</p>
            </div>
          </div>
        </Card>

        <!-- Deploy Progress -->
        <Card>
          <template #header>
            <h3 class="text-sm font-medium text-foreground">{{ t('appDetail.deployProgress') }}</h3>
          </template>
          <DeployProgress :app-id="String(app.id)" />
        </Card>
      </div>

      <!-- Logs Tab -->
      <div v-else-if="activeTab === 'logs'" class="space-y-3">
        <div class="flex items-center gap-2">
          <div class="flex items-center gap-2">
            <Radio class="w-4 h-4 text-muted-foreground" />
            <span class="text-sm text-muted-foreground">{{ t('appDetail.realtimeLogs') }}</span>
            <Switch v-model="realtimeEnabled" @update:model-value="toggleRealtime" />
          </div>
          <div class="flex-1" />
          <Button variant="outline" size="sm" :loading="loadingHistory" @click="loadHistory">
            <template #icon><History class="w-4 h-4" /></template>
            {{ t('appDetail.loadHistoryLogs') }}
          </Button>
          <Button variant="outline" size="sm" @click="clearLogs">
            <template #icon><Trash2 class="w-4 h-4" /></template>
            {{ t('appDetail.clear') }}
          </Button>
        </div>

        <div class="h-[calc(100vh-280px)] min-h-[400px]">
          <LogViewer
            :logs="logs"
            :connected="wsConnected"
            :auto-scroll="true"
            @clear="clearLogs"
          />
        </div>
      </div>

      <!-- Environment Variables Tab -->
      <div v-else-if="activeTab === 'env'" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">
            {{ t('appDetail.totalVars', { count: envList.length }) }}
          </p>
          <div class="flex items-center gap-2">
            <Button :loading="envSaving" size="sm" @click="saveEnv">
              <template #icon><Save class="w-4 h-4" /></template>
              {{ t('appDetail.save') }}
            </Button>
          </div>
        </div>

        <div v-if="envLoading" class="space-y-3">
          <Skeleton v-for="i in 5" :key="i" class="h-10 w-full" />
        </div>

        <div v-else class="space-y-2">
          <div class="grid grid-cols-[1fr_1fr_80px] gap-3 px-1">
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('appDetail.key') }}</span>
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('appDetail.value') }}</span>
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider text-center">{{ t('appDetail.actions') }}</span>
          </div>

          <div
            v-for="(item, index) in envList"
            :key="index"
            class="grid grid-cols-[1fr_1fr_80px] gap-3 items-center group"
          >
            <Input
              v-model="item.key"
              :placeholder="t('appDetail.varNamePlaceholder')"
              :class="item.isNew ? 'border-primary/50' : ''"
            />
            <div class="relative">
              <Input
                v-model="item.value"
                :type="item.visible ? 'text' : 'password'"
                :placeholder="t('appDetail.varValuePlaceholder')"
              />
              <button
                class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                @click="toggleVisibility(index)"
              >
                <Eye v-if="!item.visible" class="w-4 h-4" />
                <EyeOff v-else class="w-4 h-4" />
              </button>
            </div>
            <div class="flex justify-center">
              <Button
                variant="ghost"
                size="icon"
                class="h-8 w-8 text-muted-foreground hover:text-destructive"
                @click="removeVariable(index)"
              >
                <Trash2 class="w-4 h-4" />
              </Button>
            </div>
          </div>

          <Button variant="outline" size="sm" class="mt-2" @click="addVariable">
            <template #icon><Plus class="w-4 h-4" /></template>
            {{ t('appDetail.addVar') }}
          </Button>
        </div>
      </div>

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
      <div v-else-if="activeTab === 'deployments'">
        <Table
          :columns="deploymentColumns"
          :data="deployments"
          :loading="deploymentsLoading"
        >
          <template #cell-image="{ row }">
            <span class="text-sm text-muted-foreground truncate max-w-[200px] inline-block">
              {{ row.image || '-' }}
            </span>
          </template>
          <template #cell-status="{ row }">
            <StatusBadge :status="row.status" />
          </template>
          <template #cell-error_message="{ row }">
            <span
              v-if="row.error_message"
              class="text-sm text-destructive truncate max-w-[200px] inline-block"
              :title="row.error_message"
            >
              {{ row.error_message }}
            </span>
            <span v-else class="text-sm text-muted-foreground">-</span>
          </template>
          <template #cell-created_at="{ row }">
            <RelativeTime :date="row.created_at" />
          </template>
        </Table>
      </div>
    </template>

    <!-- Rollback confirmation dialog -->
    <AlertDialog
      v-model:open="rollbackDialogOpen"
      :title="t('appDetail.rollbackConfirm')"
      :description="t('appDetail.rollbackConfirmDesc', { name: app?.name || '', version: app?.current_version || '' })"
      :confirm-text="t('appDetail.confirmRollback')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
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
