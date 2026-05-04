<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import StatusBadge from '@/components/common/StatusBadge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Table from '@/components/ui/Table.vue'
import type { DeploymentRecord } from '@/types/models'

const { t } = useI18n()

const props = defineProps<{
  deployments: DeploymentRecord[]
  loading: boolean
}>()

const deploymentColumns = computed(() => [
  { key: 'container_name', label: t('appDetail.deployContainerName') },
  { key: 'image', label: t('appDetail.image') },
  { key: 'status', label: t('appDetail.status') },
  { key: 'error_message', label: t('appDetail.errorMessage') },
  { key: 'created_at', label: t('appDetail.time') },
])
</script>

<template>
  <Table
    :columns="deploymentColumns"
    :data="deployments"
    :loading="loading"
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
</template>
