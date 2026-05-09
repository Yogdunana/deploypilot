<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Ban } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Table from '@/components/ui/ResponsiveTable.vue'

const { t } = useI18n()

const props = defineProps<{
  issuedLicenses: any[]
  loading: boolean
}>()

const emit = defineEmits<{
  revoke: [lic: any]
}>()

const issuedColumns = computed(() => [
  { key: 'tenant', label: t('license.tenant'), mobile: true },
  { key: 'tier', label: t('license.tier'), mobile: true },
  { key: 'use_type', label: t('license.useType'), mobile: true },
  { key: 'status', label: t('license.status'), mobile: true },
  { key: 'issued_at', label: t('license.issuedAt') },
  { key: 'actions', label: t('common.actions'), width: '120px' },
])

function getStatusVariant(status: string): 'success' | 'destructive' | 'warning' | 'default' | 'secondary' {
  const s = (status || '').toLowerCase()
  switch (s) {
    case 'active':
      return 'success'
    case 'expired':
      return 'destructive'
    case 'revoked':
      return 'destructive'
    default:
      return 'secondary'
  }
}

function getStatusLabel(status: string): string {
  const s = (status || '').toLowerCase()
  const map: Record<string, string> = {
    active: t('license.active'),
    expired: t('license.expired'),
    revoked: t('license.revoked'),
  }
  return map[s] || s
}

function getTierLabel(tier: string): string {
  const t2 = (tier || '').toLowerCase()
  const map: Record<string, string> = {
    community: t('license.community'),
    team: t('license.team'),
    pro: t('license.pro'),
    enterprise: t('license.enterprise'),
  }
  return map[t2] || tier
}

function getUseTypeLabel(useType: string): string {
  const u = (useType || '').toLowerCase()
  const map: Record<string, string> = {
    'non-commercial': t('license.nonCommercial'),
    commercial: t('license.commercial'),
  }
  return map[u] || useType
}
</script>

<template>
  <Table
    :columns="issuedColumns"
    :data="issuedLicenses"
    :loading="loading"
  >
    <template #cell-tenant="{ row }">
      <span class="text-sm font-medium text-foreground">{{ row.tenant || row.tenant_name || '-' }}</span>
    </template>
    <template #cell-tier="{ row }">
      <Badge variant="outline">{{ getTierLabel(row.tier) }}</Badge>
    </template>
    <template #cell-use_type="{ row }">
      <Badge variant="secondary">{{ getUseTypeLabel(row.use_type) }}</Badge>
    </template>
    <template #cell-status="{ row }">
      <Badge :variant="getStatusVariant(row.status)">
        {{ getStatusLabel(row.status) }}
      </Badge>
    </template>
    <template #cell-issued_at="{ row }">
      <span class="text-sm text-muted-foreground">{{ row.issued_at || '-' }}</span>
    </template>
    <template #cell-actions="{ row }">
      <Button
        variant="ghost"
        size="sm"
        class="h-8 sm:h-7 text-xs text-muted-foreground hover:text-destructive min-w-[2.5rem]"
        :disabled="row.status === 'revoked'"
        @click="emit('revoke', row)"
      >
        <template #icon><Ban class="w-3.5 h-3.5" /></template>
        <span class="hidden sm:inline">{{ t('license.revoke') }}</span>
      </Button>
    </template>
  </Table>
</template>
