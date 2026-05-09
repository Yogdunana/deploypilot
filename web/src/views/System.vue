<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { Info, HeartPulse, RefreshCw, CheckCircle, XCircle, Download, ArrowUpCircle, AlertTriangle } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Card from '@/components/ui/Card.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Button from '@/components/ui/Button.vue'
import * as systemApi from '@/api/modules/system'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

// State
const loading = ref(true)
const versionInfo = ref<{ version: string; build_time: string; git_commit: string } | null>(null)
const healthInfo = ref<{ status: string; database: { status: string } } | null>(null)
const updateInfo = ref<{ current_version: string; latest_version: string; update_available: boolean; message: string; release_notes?: string } | null>(null)
const checkingUpdate = ref(false)
const upgrading = ref(false)
const upgradeProgress = ref<{ step: string; message: string; percent: number; status: string } | null>(null)
const showReleaseNotes = ref(false)

// Fetch all system info
async function fetchSystemInfo() {
  loading.value = true
  try {
    const [versionRes, healthRes] = await Promise.all([
      systemApi.getVersion(),
      systemApi.getHealth(),
    ])
    if (versionRes.data.status === 'success') {
      versionInfo.value = versionRes.data.data
    }
    if (healthRes.data.status === 'success') {
      healthInfo.value = healthRes.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('system.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Check for updates
async function handleCheckUpdate() {
  checkingUpdate.value = true
  try {
    const res = await systemApi.checkUpdate()
    if (res.data.status === 'success') {
      updateInfo.value = res.data.data
      if (res.data.data.update_available) {
        toast(t('system.updateAvailable', { version: res.data.data.latest_version }), 'warning')
      } else {
        toast(t('system.alreadyLatest'), 'success')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('system.checkFailed'), 'destructive')
  } finally {
    checkingUpdate.value = false
  }
}

// Perform upgrade
async function handleUpgrade() {
  if (!updateInfo.value?.update_available) {
    toast(t('system.noUpdateAvailable'), 'destructive')
    return
  }

  if (!confirm(t('system.upgradeConfirm', { version: updateInfo.value.latest_version }))) {
    return
  }

  upgrading.value = true
  upgradeProgress.value = { step: 'start', message: t('system.upgradeStarting'), percent: 0, status: 'running' }

  try {
    const res = await systemApi.performUpdate(updateInfo.value.latest_version)
    if (res.data.status === 'success') {
      const result = res.data.data
      upgradeProgress.value = { step: 'complete', message: result.message, percent: 100, status: 'success' }
      toast(t('system.upgradeSuccess', { version: result.new_version }), 'success')

      // Refresh version info after a delay (service is restarting)
      setTimeout(() => {
        fetchSystemInfo()
        updateInfo.value = null
        upgradeProgress.value = null
      }, 5000)
    }
  } catch (err: any) {
    upgradeProgress.value = { step: 'error', message: err.response?.data?.message || t('system.upgradeFailed'), percent: 0, status: 'error' }
    toast(t('system.upgradeFailed'), 'destructive')
  } finally {
    upgrading.value = false
  }
}

onMounted(fetchSystemInfo)
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <PageHeader :title="t('system.title')" />

    <!-- Loading skeleton -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div v-for="i in 3" :key="i" class="rounded-lg border border-border bg-card p-6 space-y-4">
        <Skeleton class="h-5 w-24" />
        <div class="space-y-3">
          <Skeleton class="h-4 w-full" />
          <Skeleton class="h-4 w-3/4" />
          <Skeleton class="h-4 w-1/2" />
        </div>
      </div>
    </div>

    <!-- Cards grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <!-- Version Info -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Info class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">{{ t('system.versionInfo') }}</h3>
          </div>
        </template>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.versionNumber') }}</span>
            <code class="text-sm font-mono text-foreground">{{ versionInfo?.version || '-' }}</code>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.buildTime') }}</span>
            <span class="text-sm text-foreground">{{ versionInfo?.build_time || '-' }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.gitCommit') }}</span>
            <code class="text-xs font-mono text-foreground truncate max-w-[140px]">{{ versionInfo?.git_commit || '-' }}</code>
          </div>
        </div>
      </Card>

      <!-- Health Status -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <HeartPulse class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">{{ t('system.healthStatus') }}</h3>
          </div>
        </template>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.systemStatus') }}</span>
            <div class="flex items-center gap-1.5">
              <CheckCircle v-if="healthInfo?.status === 'healthy'" class="w-4 h-4 text-success" />
              <XCircle v-else class="w-4 h-4 text-destructive" />
              <span class="text-sm text-foreground">{{ healthInfo?.status === 'healthy' ? t('system.normal') : t('system.abnormal') }}</span>
            </div>
          </div>
          <div v-if="healthInfo?.database" class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Database</span>
            <div class="flex items-center gap-1.5">
              <CheckCircle v-if="healthInfo.database.status === 'ok'" class="w-3.5 h-3.5 text-success" />
              <XCircle v-else class="w-3.5 h-3.5 text-destructive" />
              <span class="text-xs text-foreground">{{ healthInfo.database.status === 'ok' ? t('system.normal') : t('system.abnormal') }}</span>
            </div>
          </div>
        </div>
      </Card>

      <!-- Update Check & Upgrade -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <ArrowUpCircle class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">{{ t('system.updateCheck') }}</h3>
          </div>
        </template>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.currentVersion') }}</span>
            <code class="text-sm font-mono text-foreground">{{ versionInfo?.version || '-' }}</code>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.latestVersion') }}</span>
            <code class="text-sm font-mono text-foreground">{{ updateInfo?.latest_version || '-' }}</code>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ t('system.status') }}</span>
            <span v-if="updateInfo" class="text-sm" :class="updateInfo.update_available ? 'text-warning' : 'text-success'">
              {{ updateInfo.update_available ? t('system.newVersionAvailable') : t('system.isLatest') }}
            </span>
            <span v-else class="text-sm text-muted-foreground">{{ t('system.notChecked') }}</span>
          </div>

          <!-- Release notes toggle -->
          <button
            v-if="updateInfo?.release_notes"
            class="text-xs text-muted-foreground hover:text-foreground transition-colors w-full text-left"
            @click="showReleaseNotes = !showReleaseNotes"
          >
            {{ showReleaseNotes ? '▾' : '▸' }} {{ t('system.releaseNotes') }}
          </button>
          <div v-if="showReleaseNotes && updateInfo?.release_notes" class="text-xs text-muted-foreground bg-muted/50 rounded p-2 max-h-32 overflow-y-auto whitespace-pre-wrap">
            {{ updateInfo.release_notes }}
          </div>

          <!-- Upgrade progress -->
          <div v-if="upgradeProgress" class="space-y-2">
            <div class="w-full bg-muted rounded-full h-2">
              <div
                class="h-2 rounded-full transition-all duration-500"
                :class="upgradeProgress.status === 'error' ? 'bg-destructive' : upgradeProgress.status === 'success' ? 'bg-success' : 'bg-primary'"
                :style="{ width: upgradeProgress.percent + '%' }"
              />
            </div>
            <p class="text-xs text-muted-foreground">{{ upgradeProgress.message }}</p>
          </div>

          <!-- Action buttons -->
          <div class="flex gap-2 mt-2">
            <Button
              variant="outline"
              size="sm"
              class="flex-1"
              :loading="checkingUpdate"
              :disabled="upgrading"
              @click="handleCheckUpdate"
            >
              <template #icon><RefreshCw class="w-3.5 h-3.5" /></template>
              {{ t('system.checkUpdate') }}
            </Button>
            <Button
              v-if="updateInfo?.update_available"
              variant="default"
              size="sm"
              class="flex-1"
              :loading="upgrading"
              :disabled="checkingUpdate"
              @click="handleUpgrade"
            >
              <template #icon><Download class="w-3.5 h-3.5" /></template>
              {{ t('system.doUpgrade') }}
            </Button>
          </div>

          <!-- Warning for upgrade -->
          <div v-if="updateInfo?.update_available && !upgrading" class="flex items-start gap-1.5 text-xs text-muted-foreground">
            <AlertTriangle class="w-3.5 h-3.5 mt-0.5 shrink-0 text-warning" />
            <span>{{ t('system.upgradeWarning') }}</span>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
