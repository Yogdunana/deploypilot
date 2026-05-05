<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Badge from '@/components/ui/Badge.vue'

const { t } = useI18n()

interface Props {
  status?: string
  class?: string
}

const props = defineProps<Props>()

const statusMap: Record<string, { variant: 'success' | 'destructive' | 'warning' | 'secondary'; key: string }> = {
  running: { variant: 'success', key: 'running' },
  success: { variant: 'success', key: 'success' },
  active: { variant: 'success', key: 'active' },
  healthy: { variant: 'success', key: 'healthy' },
  online: { variant: 'success', key: 'online' },
  completed: { variant: 'success', key: 'completed' },
  enabled: { variant: 'success', key: 'enabled' },
  failed: { variant: 'destructive', key: 'failed' },
  error: { variant: 'destructive', key: 'error' },
  expired: { variant: 'destructive', key: 'expired' },
  disabled: { variant: 'destructive', key: 'disabled' },
  offline: { variant: 'destructive', key: 'offline' },
  deploying: { variant: 'warning', key: 'deploying' },
  pending: { variant: 'warning', key: 'pending' },
  renewing: { variant: 'warning', key: 'renewing' },
  building: { variant: 'warning', key: 'building' },
  warning: { variant: 'warning', key: 'warning' },
  stopped: { variant: 'secondary', key: 'stopped' },
  unknown: { variant: 'secondary', key: 'unknown' },
  inactive: { variant: 'secondary', key: 'inactive' },
}

const mapped = computed(() => {
  const s = (props.status || '').toLowerCase()
  const mappedStatus = statusMap[s]
  if (mappedStatus) {
    return {
      variant: mappedStatus.variant,
      label: t(`status.${mappedStatus.key}`)
    }
  }
  return { variant: 'secondary' as const, label: props.status || s }
})
</script>

<template>
  <Badge :variant="mapped.variant" :class="props.class">
    {{ mapped.label }}
  </Badge>
</template>
