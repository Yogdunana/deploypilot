<script setup lang="ts">
import { ref, computed, inject, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
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
        toast(`已加载 ${historyLogs.length} 条历史日志`, 'success')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '加载历史日志失败', 'destructive')
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
    toast(err.response?.data?.message || '加载环境变量失败', 'destructive')
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
    toast('存在空的变量名', 'destructive')
    return
  }
  const keys = envList.value.map((item) => item.key.trim())
  const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index)
  if (duplicates.length > 0) {
    toast(`存在重复的变量名: ${duplicates.join(', ')}`, 'destructive')
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
    toast('环境变量保存成功', 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || '保存环境变量失败', 'destructive')
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

const backupColumns = [
  { key: 'id', label: '备份 ID' },
  { key: 'app_id', label: '应用 ID' },
  { key: 'created_at', label: '创建时间' },
  { key: 'actions', label: '操作', width: '160px' },
]

async function fetchBackups() {
  backupsLoading.value = true
  try {
    const res = await appsApi.listBackups(Number(props.id))
    if (res.data.status === 'success') {
      backups.value = res.data.data || []
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取备份列表失败', 'destructive')
  } finally {
    backupsLoading.value = false
  }
}

async function createBackup() {
  creating.value = true
  try {
    await appsApi.backup(Number(props.id))
    toast('备份创建成功', 'success')
    fetchBackups()
  } catch (err: any) {
    toast(err.response?.data?.message || '创建备份失败', 'destructive')
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
    toast('备份恢复成功', 'success')
    restoreDialogOpen.value = false
  } catch (err: any) {
    toast(err.response?.data?.message || '恢复备份失败', 'destructive')
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
    toast('备份删除成功', 'success')
    deleteDialogOpen.value = false
    fetchBackups()
  } catch (err: any) {
    toast(err.response?.data?.message || '删除备份失败', 'destructive')
  } finally {
    deleting.value = false
  }
}

// Tabs
const detailTabs = [
  { key: 'overview', label: '概览' },
  { key: 'logs', label: '日志' },
  { key: 'env', label: '环境变量' },
  { key: 'backups', label: '备份' },
  { key: 'deployments', label: '部署历史' },
]

// Deployment history columns
const deploymentColumns = [
  { key: 'container_name', label: '容器名' },
  { key: 'image', label: '镜像' },
  { key: 'status', label: '状态' },
  { key: 'error_message', label: '错误信息' },
  { key: 'created_at', label: '时间' },
]

// Fetch app detail
async function fetchApp() {
  loading.value = true
  try {
    const res = await appsApi.get(Number(props.id))
    if (res.data.status === 'success') {
      app.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取应用详情失败', 'destructive')
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
    toast(err.response?.data?.message || '获取部署记录失败', 'destructive')
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
    toast('部署已触发', 'success')
    fetchApp()
    if (activeTab.value === 'deployments') fetchDeployments()
  } catch (err: any) {
    toast(err.response?.data?.message || '部署失败', 'destructive')
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
    toast('构建已触发', 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || '构建失败', 'destructive')
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
    toast('回滚已触发', 'success')
    fetchApp()
    if (activeTab.value === 'deployments') fetchDeployments()
  } catch (err: any) {
    toast(err.response?.data?.message || '回滚失败', 'destructive')
  } finally {
    rollingBack.value = false
  }
}

// Info items for overview
const infoItems = computed(() => {
  if (!app.value) return []
  return [
    { label: '应用名称', value: app.value.name, icon: Layers },
    { label: '仓库地址', value: app.value.repo_url, icon: GitBranch },
    { label: '分支', value: app.value.branch, icon: GitBranch },
    { label: '技术栈', value: app.value.tech_stack, icon: Layers },
    { label: '域名', value: app.value.domain, icon: Globe },
    { label: '服务器', value: String(app.value.server_id), icon: Server },
    { label: '状态', value: app.value.status, icon: Shield },
    { label: '当前版本', value: app.value.current_version || '-', icon: Layers },
    { label: '容器名', value: app.value.container_name || '-', icon: Server },
    { label: '创建时间', value: app.value.created_at, icon: Clock, isTime: true },
    { label: '更新时间', value: app.value.updated_at, icon: Clock, isTime: true },
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
                {{ app?.name || '加载中...' }}
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
          部署
        </Button>
        <Button variant="outline" :loading="building" @click="handleBuild">
          <template #icon><Hammer class="w-4 h-4" /></template>
          构建
        </Button>
        <Button variant="outline" @click="openRollbackDialog">
          <template #icon><RotateCcw class="w-4 h-4" /></template>
          回滚
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
            <h3 class="text-sm font-medium text-foreground">基本信息</h3>
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
                <p v-if="item.label === '状态'" class="text-sm text-foreground mt-0.5">
                  <StatusBadge :status="item.value" />
                </p>
                <p v-else-if="item.label === '技术栈'" class="text-sm text-foreground mt-0.5">
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
            <h3 class="text-sm font-medium text-foreground">资源限制</h3>
          </template>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <p class="text-xs text-muted-foreground">内存</p>
              <p class="text-sm text-foreground mt-0.5">{{ app.resource_limits?.memory || '-' }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">CPU</p>
              <p class="text-sm text-foreground mt-0.5">{{ app.resource_limits?.cpu || '-' }}</p>
            </div>
          </div>
        </Card>

        <!-- Deploy Progress -->
        <Card>
          <template #header>
            <h3 class="text-sm font-medium text-foreground">部署进度</h3>
          </template>
          <DeployProgress :app-id="String(app.id)" />
        </Card>
      </div>

      <!-- Logs Tab -->
      <div v-else-if="activeTab === 'logs'" class="space-y-3">
        <!-- 工具栏 -->
        <div class="flex items-center gap-2">
          <div class="flex items-center gap-2">
            <Radio class="w-4 h-4 text-muted-foreground" />
            <span class="text-sm text-muted-foreground">实时日志</span>
            <Switch v-model="realtimeEnabled" @update:model-value="toggleRealtime" />
          </div>
          <div class="flex-1" />
          <Button variant="outline" size="sm" :loading="loadingHistory" @click="loadHistory">
            <template #icon><History class="w-4 h-4" /></template>
            加载历史日志
          </Button>
          <Button variant="outline" size="sm" @click="clearLogs">
            <template #icon><Trash2 class="w-4 h-4" /></template>
            清空
          </Button>
        </div>

        <!-- 日志查看器 -->
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
        <!-- 操作栏 -->
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">
            共 {{ envList.length }} 个变量
          </p>
          <div class="flex items-center gap-2">
            <Button :loading="envSaving" size="sm" @click="saveEnv">
              <template #icon><Save class="w-4 h-4" /></template>
              保存
            </Button>
          </div>
        </div>

        <!-- 加载状态 -->
        <div v-if="envLoading" class="space-y-3">
          <Skeleton v-for="i in 5" :key="i" class="h-10 w-full" />
        </div>

        <!-- 环境变量编辑器 -->
        <div v-else class="space-y-2">
          <!-- 表头 -->
          <div class="grid grid-cols-[1fr_1fr_80px] gap-3 px-1">
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Key</span>
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Value</span>
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider text-center">操作</span>
          </div>

          <!-- 变量行 -->
          <div
            v-for="(item, index) in envList"
            :key="index"
            class="grid grid-cols-[1fr_1fr_80px] gap-3 items-center group"
          >
            <Input
              v-model="item.key"
              placeholder="变量名"
              :class="item.isNew ? 'border-primary/50' : ''"
            />
            <div class="relative">
              <Input
                v-model="item.value"
                :type="item.visible ? 'text' : 'password'"
                placeholder="变量值"
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

          <!-- 添加按钮 -->
          <Button variant="outline" size="sm" class="mt-2" @click="addVariable">
            <template #icon><Plus class="w-4 h-4" /></template>
            添加变量
          </Button>
        </div>
      </div>

      <!-- Backups Tab -->
      <div v-else-if="activeTab === 'backups'" class="space-y-3">
        <!-- 操作栏 -->
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">
            共 {{ backups.length }} 个备份
          </p>
          <Button :loading="creating" size="sm" @click="createBackup">
            <template #icon><Plus class="w-4 h-4" /></template>
            创建备份
          </Button>
        </div>

        <!-- 备份表格 -->
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
                恢复
              </Button>
              <Button
                variant="ghost"
                size="sm"
                class="h-7 text-xs text-muted-foreground hover:text-destructive"
                @click="openDeleteDialog(row)"
              >
                <template #icon><Trash2 class="w-3.5 h-3.5" /></template>
                删除
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
      title="回滚应用"
      :description="`确定要将应用「${app?.name}」回滚到上一个版本吗？当前版本：${app?.current_version || '未知'}`"
      confirm-text="确认回滚"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmRollback"
    />

    <!-- Restore backup dialog -->
    <AlertDialog
      v-model:open="restoreDialogOpen"
      title="恢复备份"
      :description="`确定要恢复备份「${selectedBackup?.id}」吗？此操作将覆盖当前应用状态。`"
      confirm-text="确认恢复"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmRestore"
    />

    <!-- Delete backup dialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      title="删除备份"
      :description="`确定要删除备份「${selectedBackup?.id}」吗？此操作不可撤销。`"
      confirm-text="确认删除"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
