<script setup lang="ts">
import { ref } from 'vue'
import { useToast } from '@/composables/useToast'
import { useI18n } from 'vue-i18n'

import { Play, CheckCircle, XCircle } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import { batchDeploy } from '@/api/modules/batchDeploy'

const { t } = useI18n()
const { toast } = useToast()

interface AppEntry {
  image: string
  container_name: string
  ports: string
  restart_policy: string
  env_vars: string
}

const apps = ref<AppEntry[]>([{ image: '', container_name: '', ports: '', restart_policy: 'unless-stopped', env_vars: '' }])
const strategy = ref<'sequential' | 'parallel' | 'rolling'>('sequential')
const maxConcurrent = ref(5)
const batchSize = ref(3)
const deploying = ref(false)
const result = ref<{ total: number; success: number; failed: number; results: { index: number; app_name: string; success: boolean; error?: string }[]; duration_seconds: number } | null>(null)

function addApp() {
  apps.value.push({ image: '', container_name: '', ports: '', restart_policy: 'unless-stopped', env_vars: '' })
}

function removeApp(index: number) {
  if (apps.value.length <= 1) return
  apps.value.splice(index, 1)
}

async function deploy() {
  const validApps = apps.value.filter(a => a.image.trim())
  if (validApps.length === 0) {
    toast(t('batch.noApps'), 'error')
    return
  }

  deploying.value = true
  result.value = null
  try {
    const res = await batchDeploy({
      apps: validApps.map(a => ({ ...a })),
      strategy: strategy.value,
      max_concurrent: strategy.value === 'parallel' ? maxConcurrent.value : undefined,
      batch_size: strategy.value === 'rolling' ? batchSize.value : undefined,
    })
    if (res.data.status === 'success') {
      result.value = res.data.data
      toast(t('batch.completed'), 'success')
    }
  } catch {
    toast(t('batch.deployFailed'), 'error')
  } finally {
    deploying.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader :title="t('batch.title')" :description="t('batch.description')" />

    <div class="space-y-6">
      <!-- Strategy Selection -->
      <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
        <h3 class="text-sm font-medium text-white mb-3">{{ t('batch.strategy') }}</h3>
        <div class="flex gap-3">
          <button v-for="s in ['sequential', 'parallel', 'rolling']" :key="s" class="px-4 py-2 text-sm rounded-lg border transition-colors" :class="strategy === s ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'" @click="strategy = s as any">
            {{ t(`batch.strategy_${s}`) }}
          </button>
        </div>
        <div v-if="strategy === 'parallel'" class="mt-3">
          <label class="block text-xs text-gray-400 mb-1">{{ t('batch.maxConcurrent') }}</label>
          <input v-model.number="maxConcurrent" type="number" min="1" max="20" class="w-24 bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white" />
        </div>
        <div v-if="strategy === 'rolling'" class="mt-3">
          <label class="block text-xs text-gray-400 mb-1">{{ t('batch.batchSize') }}</label>
          <input v-model.number="batchSize" type="number" min="1" max="10" class="w-24 bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white" />
        </div>
      </div>

      <!-- App List -->
      <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-medium text-white">{{ t('batch.apps') }} ({{ apps.length }})</h3>
          <Button variant="ghost" size="sm" @click="addApp">+ {{ t('batch.addApp') }}</Button>
        </div>
        <div class="space-y-3">
          <div v-for="(app, i) in apps" :key="i" class="bg-gray-800/50 rounded-lg p-3">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs text-gray-500">#{{ i + 1 }}</span>
              <button v-if="apps.length > 1" class="text-xs text-red-400 hover:text-red-300" @click="removeApp(i)">{{ t('common.delete') }}</button>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
              <div>
                <label class="block text-xs text-gray-500 mb-1">Image</label>
                <input v-model="app.image" placeholder="nginx:latest" class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white" />
              </div>
              <div>
                <label class="block text-xs text-gray-500 mb-1">{{ t('batch.containerName') }}</label>
                <input v-model="app.container_name" placeholder="my-app" class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white" />
              </div>
              <div>
                <label class="block text-xs text-gray-500 mb-1">{{ t('batch.ports') }}</label>
                <input v-model="app.ports" placeholder="8080:80" class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white" />
              </div>
              <div>
                <label class="block text-xs text-gray-500 mb-1">Env Vars (JSON)</label>
                <input v-model="app.env_vars" placeholder='{"KEY":"VALUE"}' class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white font-mono" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Deploy Button -->
      <div class="flex justify-end">
        <Button :disabled="deploying" @click="deploy">
          <Play class="w-4 h-4 mr-2" :class="{ 'animate-pulse': deploying }" />
          {{ deploying ? t('batch.deploying') : t('batch.deploy') }}
        </Button>
      </div>

      <!-- Results -->
      <div v-if="result" class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
        <h3 class="text-sm font-medium text-white mb-3">{{ t('batch.results') }}</h3>
        <div class="grid grid-cols-3 gap-4 mb-4">
          <div class="text-center">
            <p class="text-2xl font-bold text-white">{{ result.total }}</p>
            <p class="text-xs text-gray-400">{{ t('batch.total') }}</p>
          </div>
          <div class="text-center">
            <p class="text-2xl font-bold text-green-400">{{ result.success }}</p>
            <p class="text-xs text-gray-400">{{ t('batch.success') }}</p>
          </div>
          <div class="text-center">
            <p class="text-2xl font-bold text-red-400">{{ result.failed }}</p>
            <p class="text-xs text-gray-400">{{ t('batch.failed') }}</p>
          </div>
        </div>
        <div class="space-y-1">
          <div v-for="r in result.results" :key="r.index" class="flex items-center gap-2 text-xs">
            <component :is="r.success ? CheckCircle : XCircle" class="w-3 h-3" :class="r.success ? 'text-green-400' : 'text-red-400'" />
            <span class="text-gray-300">{{ r.app_name || `#${r.index + 1}` }}</span>
            <span v-if="r.error" class="text-red-400">{{ r.error }}</span>
          </div>
        </div>
        <div class="mt-3 text-xs text-gray-500">{{ t('batch.duration') }}: {{ result.duration_seconds.toFixed(1) }}s</div>
      </div>
    </div>
  </div>
</template>
