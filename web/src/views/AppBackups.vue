<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useRouter } from 'vue-router'
import { ArrowLeft, Plus, RotateCcw, Trash2, Loader2 } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Table from '@/components/ui/Table.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as appsApi from '@/api/modules/apps'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = useToast()

const appName = ref('')
const loading = ref(true)
const creating = ref(false)
const restoring = ref(false)
const deleting = ref(false)

interface Backup {
  id: string
  app_id: number
  created_at: string
  size?: string
}

const backups = ref<Backup[]>([])

// 对话框状态
const restoreDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const selectedBackup = ref<Backup | null>(null)

// 表格列
const columns = computed(() => [
  { key: 'id', label: t('appBackups.appName') },
  { key: 'app_id', label: t('appBackups.image') },
  { key: 'created_at', label: t('appBackups.createdAt') },
  { key: 'actions', label: t('appBackups.restore'), width: '160px' },
])

// 获取备份列表
async function fetchBackups() {
  loading.value = true
  try {
    const res = await appsApi.listBackups(props.id)
    if (res.data.status === 'success') {
      backups.value = res.data.data || []
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appBackups.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// 创建备份
async function createBackup() {
  creating.value = true
  try {
    await appsApi.backup(props.id)
    toast(t('appBackups.backupCreated'), 'success')
    fetchBackups()
  } catch (err: any) {
    toast(err.response?.data?.message || t('appBackups.backupCreateFailed'), 'destructive')
  } finally {
    creating.value = false
  }
}

// 恢复备份
function openRestoreDialog(backup: any) {
  selectedBackup.value = backup
  restoreDialogOpen.value = true
}

async function confirmRestore() {
  if (!selectedBackup.value) return
  restoring.value = true
  try {
    await appsApi.restore(props.id, { backup_id: selectedBackup.value.id })
    toast(t('appBackups.restored'), 'success')
    restoreDialogOpen.value = false
  } catch (err: any) {
    toast(err.response?.data?.message || t('appBackups.restoreFailed'), 'destructive')
  } finally {
    restoring.value = false
  }
}

// 删除备份
function openDeleteDialog(backup: any) {
  selectedBackup.value = backup
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!selectedBackup.value) return
  deleting.value = true
  try {
    await appsApi.deleteBackup(props.id, selectedBackup.value.id)
    toast(t('appBackups.deleted'), 'success')
    deleteDialogOpen.value = false
    fetchBackups()
  } catch (err: any) {
    toast(err.response?.data?.message || t('appBackups.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
  }
}

// 获取应用信息
async function fetchApp() {
  try {
    const res = await appsApi.get(props.id)
    if (res.data.status === 'success') {
      appName.value = res.data.data.name
    }
  } catch {
    // 静默处理
  }
}

onMounted(() => {
  fetchApp()
  fetchBackups()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader>
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push('/apps')">
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <div>
            <h1 class="text-xl font-semibold text-foreground">
              {{ t('appBackups.title', { name: appName }) }}
            </h1>
            <p class="mt-0.5 text-sm text-muted-foreground">
              {{ t('appBackups.totalBackups', { count: backups.length }) }}
            </p>
          </div>
        </div>
      </template>
      <template #actions>
        <Button :loading="creating" @click="createBackup">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('appBackups.createBackup') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 备份表格 -->
    <Table
      :columns="columns"
      :data="backups"
      :loading="loading"
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
            {{ t('appBackups.restore') }}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="h-7 text-xs text-muted-foreground hover:text-destructive"
            @click="openDeleteDialog(row)"
          >
            <template #icon><Trash2 class="w-3.5 h-3.5" /></template>
            {{ t('appBackups.delete') }}
          </Button>
        </div>
      </template>
    </Table>

    <!-- 恢复确认对话框 -->
    <AlertDialog
      v-model:open="restoreDialogOpen"
      :title="t('appBackups.restoreConfirm')"
      :description="t('appBackups.restoreConfirmDesc', { id: selectedBackup?.id || '' })"
      :confirm-text="t('appBackups.confirmRestore')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmRestore"
    />

    <!-- 删除确认对话框 -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('appBackups.deleteConfirm')"
      :description="t('appBackups.deleteConfirmDesc', { id: selectedBackup?.id || '' })"
      :confirm-text="t('appBackups.confirmDelete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
