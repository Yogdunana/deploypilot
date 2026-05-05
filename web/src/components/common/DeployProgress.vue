<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSSE } from '@/composables/useSSE'
import { CheckCircle, XCircle, Loader2, Circle, Clock } from 'lucide-vue-next'
import Progress from '@/components/ui/Progress.vue'

const { t } = useI18n()

interface Props {
  appId: string
}

const props = defineProps<Props>()

// 步骤定义
interface Step {
  key: string
  label: string
  status: 'waiting' | 'running' | 'done' | 'error'
  message?: string
}

const steps = ref<Step[]>([
  { key: 'pulling', label: t('deploy.stepPulling'), status: 'waiting' },
  { key: 'building', label: t('deploy.stepBuilding'), status: 'waiting' },
  { key: 'deploying', label: t('deploy.stepDeploying'), status: 'waiting' },
  { key: 'health_check', label: t('deploy.stepHealthCheck'), status: 'waiting' },
  { key: 'done', label: t('deploy.stepDone'), status: 'waiting' },
])

const overallStatus = ref<'running' | 'success' | 'error' | 'idle'>('idle')
const errorMessage = ref('')

// 步骤映射
const stepKeyMap: Record<string, string> = {
  pulling: 'pulling',
  building: 'building',
  deploying: 'deploying',
  health_check: 'health_check',
  done: 'done',
}

// 计算进度
const progressValue = computed(() => {
  const doneSteps = steps.value.filter((s) => s.status === 'done').length
  return Math.round((doneSteps / steps.value.length) * 100)
})

// 当前步骤索引
const currentStepIndex = computed(() => {
  return steps.value.findIndex((s) => s.status === 'running')
})

// 使用 SSE 连接
const { status: sseStatus } = useSSE({
  url: `/api/v1/sse/deploy/${props.appId}`,
  autoConnect: false,
  onEvent(event: string, data: any) {
    const stepKey = data.step || data.type || event

    if (stepKey === 'done') {
      // 标记所有步骤为完成
      steps.value.forEach((s) => {
        if (s.status !== 'error') s.status = 'done'
      })
      overallStatus.value = data.success === false ? 'error' : 'success'
      if (data.error) errorMessage.value = data.error
      return
    }

    if (stepKey === 'error') {
      overallStatus.value = 'error'
      errorMessage.value = data.message || data.error || t('deploy.failed')
      // 标记当前运行中的步骤为错误
      const runningStep = steps.value.find((s) => s.status === 'running')
      if (runningStep) {
        runningStep.status = 'error'
        runningStep.message = errorMessage.value
      }
      return
    }

    // 映射步骤
    const mappedKey = stepKeyMap[stepKey]
    if (mappedKey) {
      // 将之前的 running 步骤标记为 done
      steps.value.forEach((s) => {
        if (s.status === 'running') s.status = 'done'
      })

      // 设置当前步骤为 running
      const step = steps.value.find((s) => s.key === mappedKey)
      if (step) {
        step.status = 'running'
        step.message = data.message || ''
      }

      overallStatus.value = 'running'
    }
  },
  onError(err) {
    overallStatus.value = 'error'
    errorMessage.value = err.message || t('deploy.connectionFailed')
  },
})

function startWatching() {
  // 重置状态
  steps.value.forEach((s) => {
    s.status = 'waiting'
    s.message = ''
  })
  overallStatus.value = 'running'
  errorMessage.value = ''
}

// 暴露方法供外部调用
defineExpose({ startWatching })
</script>

<template>
  <div class="space-y-4">
    <!-- 整体进度 -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <template v-if="overallStatus === 'running'">
            <Loader2 class="w-4 h-4 animate-spin text-primary" />
            <span class="text-sm font-medium text-foreground">{{ t('deploy.inProgress') }}</span>
          </template>
          <template v-else-if="overallStatus === 'success'">
            <CheckCircle class="w-4 h-4 text-success" />
            <span class="text-sm font-medium text-success">{{ t('deploy.success') }}</span>
          </template>
          <template v-else-if="overallStatus === 'error'">
            <XCircle class="w-4 h-4 text-destructive" />
            <span class="text-sm font-medium text-destructive">{{ t('deploy.failed') }}</span>
          </template>
          <template v-else>
            <Clock class="w-4 h-4 text-muted-foreground" />
            <span class="text-sm font-medium text-muted-foreground">{{ t('deploy.waiting') }}</span>
          </template>
        </div>
        <span class="text-xs text-muted-foreground">{{ progressValue }}%</span>
      </div>
      <Progress :value="progressValue" :variant="overallStatus === 'error' ? 'destructive' : overallStatus === 'success' ? 'success' : 'default'" />
    </div>

    <!-- 错误信息 -->
    <div v-if="errorMessage" class="rounded-md border border-destructive/30 bg-destructive/10 p-3">
      <p class="text-sm text-destructive">{{ errorMessage }}</p>
    </div>

    <!-- 步骤列表 -->
    <div class="space-y-1">
      <div
        v-for="(step, index) in steps"
        :key="step.key"
        class="flex items-center gap-3 rounded-md px-3 py-2 transition-colors"
        :class="step.status === 'running' ? 'bg-primary/5 border border-primary/20' : ''"
      >
        <!-- 步骤图标 -->
        <div class="shrink-0">
          <CheckCircle v-if="step.status === 'done'" class="w-4 h-4 text-success" />
          <Loader2 v-else-if="step.status === 'running'" class="w-4 h-4 animate-spin text-primary" />
          <XCircle v-else-if="step.status === 'error'" class="w-4 h-4 text-destructive" />
          <Circle v-else class="w-4 h-4 text-muted-foreground/40" />
        </div>

        <!-- 步骤信息 -->
        <div class="flex-1 min-w-0">
          <p
            class="text-sm"
            :class="{
              'text-foreground font-medium': step.status === 'running',
              'text-foreground': step.status === 'done',
              'text-destructive': step.status === 'error',
              'text-muted-foreground': step.status === 'waiting',
            }"
          >
            {{ step.label }}
          </p>
          <p v-if="step.message && step.status === 'running'" class="text-xs text-muted-foreground mt-0.5 truncate">
            {{ step.message }}
          </p>
          <p v-if="step.message && step.status === 'error'" class="text-xs text-destructive/80 mt-0.5 truncate">
            {{ step.message }}
          </p>
        </div>

        <!-- 步骤状态文字 -->
        <span
          class="text-xs shrink-0"
          :class="{
            'text-primary': step.status === 'running',
            'text-success': step.status === 'done',
            'text-destructive': step.status === 'error',
            'text-muted-foreground/50': step.status === 'waiting',
          }"
        >
          {{ step.status === 'waiting' ? t('deploy.statusWaiting') : step.status === 'running' ? t('deploy.statusRunning') : step.status === 'done' ? t('deploy.statusDone') : t('deploy.statusFailed') }}
        </span>
      </div>
    </div>
  </div>
</template>
