<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { MoreHorizontal, Pencil, Trash2, RefreshCw } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Table from '@/components/ui/ResponsiveTable.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import type { Credential } from '@/types/models'

const { t } = useI18n()

const props = defineProps<{
  credentials: Credential[]
  loading: boolean
}>()

const emit = defineEmits<{
  edit: [item: Credential]
  rotate: [item: Credential]
  delete: [item: Credential]
  detail: [item: Credential]
}>()

const columns = computed(() => [
  { key: 'name', label: t('credentials.name'), mobile: true },
  { key: 'type', label: t('credentials.type'), mobile: true },
  { key: 'expiry_status', label: t('credentials.expiryStatus'), mobile: true },
  { key: 'created_at', label: t('credentials.createdAt') },
  { key: 'actions', label: t('credentials.actions'), width: '80px' },
])

function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    ssh: { variant: 'default', label: 'SSH' },
    api_key: { variant: 'success', label: 'API Key' },
    token: { variant: 'warning', label: 'Token' },
  }
  return map[type] || { variant: 'secondary' as const, label: type }
}

function getExpiryStatus(item: Credential): { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive'; label: string } {
  if (item.is_expired) {
    return { variant: 'destructive', label: t('credentials.expired') }
  }
  if (item.days_until_expiry !== undefined && item.days_until_expiry !== -1 && item.days_until_expiry <= 7) {
    return { variant: 'warning', label: t('credentials.expiringSoon') }
  }
  if (item.days_until_expiry === -1 || !item.expires_at) {
    return { variant: 'success', label: t('credentials.neverExpires') }
  }
  return { variant: 'success', label: t('credentials.valid') }
}

function getDropdownItems(item: Credential) {
  return [
    { label: t('credentials.edit'), icon: Pencil, action: () => emit('edit', item) },
    { label: t('credentials.rotate'), icon: RefreshCw, action: () => emit('rotate', item) },
    { label: t('credentials.delete'), icon: Trash2, danger: true, action: () => emit('delete', item) },
  ]
}
</script>

<template>
  <Table
    :columns="columns"
    :data="credentials"
  >
    <template #cell-name="{ row }">
      <span class="text-sm font-medium text-foreground cursor-pointer hover:underline" @click="emit('detail', row as Credential)">{{ row.name }}</span>
    </template>
    <template #cell-type="{ row }">
      <Badge :variant="getTypeBadge(row.type).variant">
        {{ getTypeBadge(row.type).label }}
      </Badge>
    </template>
    <template #cell-expiry_status="{ row }">
      <Badge :variant="getExpiryStatus(row as Credential).variant">
        {{ getExpiryStatus(row as Credential).label }}
      </Badge>
    </template>
    <template #cell-created_at="{ row }">
      <RelativeTime :date="row.created_at" />
    </template>
    <template #cell-actions="{ row }">
      <DropdownMenu :items="getDropdownItems(row as Credential)">
        <template #trigger>
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="w-4 h-4" />
          </Button>
        </template>
      </DropdownMenu>
    </template>
  </Table>
</template>
