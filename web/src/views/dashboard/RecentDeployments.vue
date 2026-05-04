<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowRight } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Table from '@/components/ui/Table.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import type { DeploymentRecord } from '@/types/models'

const { t } = useI18n()
const router = useRouter()

const props = defineProps<{
  recentDeployments: DeploymentRecord[]
  loading: boolean
}>()

const deploymentColumns = computed(() => [
  { key: 'app_name', label: t('dashboard.appName') },
  { key: 'status', label: t('dashboard.status'), width: '120px' },
  { key: 'created_at', label: t('dashboard.time'), width: '160px' },
])

function goToDeployment(row: DeploymentRecord) {
  router.push(`/deployments/${row.id}`)
}
</script>

<template>
  <Card>
    <template #header>
      <div class="flex items-center justify-between">
        <h2 class="text-base font-semibold text-foreground">{{ t('dashboard.recentDeploys') }}</h2>
        <Button variant="ghost" size="sm" @click="router.push('/deployments')">
          {{ t('dashboard.viewAll') }}
          <template #icon>
            <ArrowRight class="w-4 h-4" />
          </template>
        </Button>
      </div>
    </template>

    <Table
      :columns="deploymentColumns"
      :data="recentDeployments"
      :loading="loading"
    >
      <template #cell-app_name="{ row }">
        <span class="font-medium text-foreground">{{ row.app_name }}</span>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
    </Table>
  </Card>
</template>
