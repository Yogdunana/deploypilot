<script setup lang="ts">
import { ref } from 'vue'
import { useToast } from '@/composables/useToast'
import { Rocket, Search, CheckCircle, XCircle, Loader2, Clock } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import * as cicdApi from '@/api/modules/cicd'
import type { CICDBuild } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

// 触发构建表单
const triggerForm = ref({
  provider: 'github_actions',
  repo_url: '',
  branch: 'main',
})

// 查询构建状态表单
const queryForm = ref({
  run_id: '',
  provider: 'github_actions',
})

// 构建状态
const triggering = ref(false)
const querying = ref(false)
const buildResult = ref<CICDBuild | null>(null)
const queryResult = ref<CICDBuild | null>(null)

// Provider 选项
const providerOptions = [
  { label: 'GitHub Actions', value: 'github_actions' },
]

// 构建状态样式
function getBuildStatusVariant(status: string): 'success' | 'destructive' | 'warning' | 'secondary' {
  const s = (status || '').toLowerCase()
  switch (s) {
    case 'success':
    case 'completed':
      return 'success'
    case 'failed':
    case 'failure':
    case 'cancelled':
      return 'destructive'
    case 'running':
    case 'in_progress':
    case 'queued':
      return 'warning'
    default:
      return 'secondary'
  }
}

function getBuildStatusLabel(status: string): string {
  const s = (status || '').toLowerCase()
  const map: Record<string, string> = {
    success: t('cicd.buildStatusLabel.success'),
    completed: t('cicd.buildStatusLabel.completed'),
    failed: t('cicd.buildStatusLabel.failed'),
    failure: t('cicd.buildStatusLabel.failed'),
    cancelled: t('cicd.buildStatusLabel.cancelled'),
    running: t('cicd.buildStatusLabel.running'),
    in_progress: t('cicd.buildStatusLabel.in_progress'),
    queued: t('cicd.buildStatusLabel.queued'),
    pending: t('cicd.buildStatusLabel.pending'),
  }
  return map[s] || s
}

// 触发构建
async function handleTrigger() {
  if (!triggerForm.value.repo_url.trim()) {
    toast(t('cicd.repoUrlRequired'), 'destructive')
    return
  }
  if (!triggerForm.value.branch.trim()) {
    toast(t('cicd.branchRequired'), 'destructive')
    return
  }

  triggering.value = true
  buildResult.value = null
  try {
    const res = await cicdApi.triggerBuild({
      provider: triggerForm.value.provider,
      repo_url: triggerForm.value.repo_url.trim(),
      branch: triggerForm.value.branch.trim(),
    })
    if (res.data.status === 'success') {
      buildResult.value = res.data.data
      toast(t('cicd.buildTriggered'), 'success')
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('cicd.triggerFailed'), 'destructive')
  } finally {
    triggering.value = false
  }
}

// 查询构建状态
async function handleQuery() {
  if (!queryForm.value.run_id.trim()) {
    toast(t('cicd.runIdRequired'), 'destructive')
    return
  }

  querying.value = true
  queryResult.value = null
  try {
    const res = await cicdApi.getBuildStatus(queryForm.value.run_id.trim(), queryForm.value.provider)
    if (res.data.status === 'success') {
      queryResult.value = res.data.data
      toast(t('cicd.querySuccess'), 'success')
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('cicd.queryFailed'), 'destructive')
  } finally {
    querying.value = false
  }
}

// 格式化时间
function formatTime(dateStr: string): string {
  if (!dateStr) return '-'
  try {
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch {
    return dateStr
  }
}
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader :title="t('cicd.title')" :description="t('cicd.description')" />

    <!-- 两列布局 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- 左侧 - 触发构建 -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Rocket class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-medium text-foreground">{{ t('cicd.triggerBuild') }}</h3>
          </div>
        </template>
        <div class="space-y-4">
          <!-- Provider -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">Provider</label>
            <Select
              v-model="triggerForm.provider"
              :options="providerOptions"
              :placeholder="t('cicd.selectProvider')"
            />
          </div>

          <!-- 仓库地址 -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">{{ t('cicd.repoUrl') }}</label>
            <Input
              v-model="triggerForm.repo_url"
              :placeholder="t('cicd.repoUrlPlaceholder')"
            />
          </div>

          <!-- 分支 -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">{{ t('cicd.branch') }}</label>
            <Input
              v-model="triggerForm.branch"
              :placeholder="t('cicd.branchPlaceholder')"
            />
          </div>

          <!-- 触发按钮 -->
          <Button :loading="triggering" class="w-full" @click="handleTrigger">
            <template #icon><Rocket class="w-4 h-4" /></template>
            {{ t('cicd.triggerBuild') }}
          </Button>

          <!-- 构建结果 -->
          <div v-if="buildResult" class="rounded-md border border-border p-4 space-y-2">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-foreground">{{ t('cicd.buildResult') }}</span>
              <Badge :variant="getBuildStatusVariant(buildResult.status)">
                {{ getBuildStatusLabel(buildResult.status) }}
              </Badge>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
              <div>
                <span class="text-muted-foreground">Run ID:</span>
                <span class="text-foreground ml-1 font-mono break-all">{{ buildResult.run_id }}</span>
              </div>
              <div>
                <span class="text-muted-foreground">Provider:</span>
                <span class="text-foreground ml-1">{{ buildResult.provider }}</span>
              </div>
              <div>
                <span class="text-muted-foreground">{{ t('cicd.triggerType') }}:</span>
                <span class="text-foreground ml-1">{{ buildResult.trigger_type || '-' }}</span>
              </div>
              <div>
                <span class="text-muted-foreground">{{ t('cicd.startTime') }}:</span>
                <span class="text-foreground ml-1">{{ formatTime(buildResult.started_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- 右侧 - 查询构建状态 -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Search class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-medium text-foreground">{{ t('cicd.queryBuildStatus') }}</h3>
          </div>
        </template>
        <div class="space-y-4">
          <!-- Run ID -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">Run ID</label>
            <Input
              v-model="queryForm.run_id"
              :placeholder="t('cicd.runIdPlaceholder')"
            />
          </div>

          <!-- Provider -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">Provider</label>
            <Select
              v-model="queryForm.provider"
              :options="providerOptions"
              :placeholder="t('cicd.selectProvider')"
            />
          </div>

          <!-- 查询按钮 -->
          <Button :loading="querying" variant="outline" class="w-full" @click="handleQuery">
            <template #icon><Search class="w-4 h-4" /></template>
            {{ t('cicd.queryStatus') }}
          </Button>

          <!-- 查询结果 -->
          <div v-if="queryResult" class="rounded-md border border-border p-4 space-y-3">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-foreground">{{ t('cicd.buildStatus') }}</span>
              <Badge :variant="getBuildStatusVariant(queryResult.status)">
                {{ getBuildStatusLabel(queryResult.status) }}
              </Badge>
            </div>

            <!-- 状态图标 -->
            <div class="flex items-center gap-3 py-2">
              <CheckCircle v-if="queryResult.status === 'success' || queryResult.status === 'completed'" class="w-8 h-8 text-success" />
              <XCircle v-else-if="queryResult.status === 'failed' || queryResult.status === 'failure'" class="w-8 h-8 text-destructive" />
              <Loader2 v-else-if="queryResult.status === 'running' || queryResult.status === 'in_progress'" class="w-8 h-8 text-warning animate-spin" />
              <Clock v-else class="w-8 h-8 text-muted-foreground" />
              <div>
                <p class="text-sm font-medium text-foreground">
                  {{ getBuildStatusLabel(queryResult.status) }}
                </p>
                <p class="text-xs text-muted-foreground">
                  Run ID: {{ queryResult.run_id }}
                </p>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
              <div>
                <span class="text-muted-foreground">Provider:</span>
                <span class="text-foreground ml-1">{{ queryResult.provider }}</span>
              </div>
              <div>
                <span class="text-muted-foreground">{{ t('cicd.triggerType') }}:</span>
                <span class="text-foreground ml-1">{{ queryResult.trigger_type || '-' }}</span>
              </div>
              <div>
                <span class="text-muted-foreground">{{ t('cicd.startTime') }}:</span>
                <span class="text-foreground ml-1">{{ formatTime(queryResult.started_at) }}</span>
              </div>
              <div>
                <span class="text-muted-foreground">{{ t('cicd.endTime') }}:</span>
                <span class="text-foreground ml-1">{{ formatTime(queryResult.finished_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
