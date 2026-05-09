<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Badge from '@/components/ui/Badge.vue'

interface Props {
  status?: string
  class?: string
}

const props = defineProps<Props>()
const { t } = useI18n()

const variantMap: Record<string, 'success' | 'destructive' | 'warning' | 'secondary'> = {
  running: 'success',
  success: 'success',
  active: 'success',
  healthy: 'success',
  online: 'success',
  completed: 'success',
  enabled: 'success',
  failed: 'destructive',
  error: 'destructive',
  expired: 'destructive',
  disabled: 'destructive',
  offline: 'destructive',
  deploying: 'warning',
  pending: 'warning',
  renewing: 'warning',
  building: 'warning',
  warning: 'warning',
  stopped: 'secondary',
  unknown: 'secondary',
  inactive: 'secondary',
}

const mapped = computed(() => {
  const s = (props.status || '').toLowerCase()
  const variant = variantMap[s] || 'secondary'
  const translated = t(`status.${s}`)
  const label = translated !== `status.${s}` ? translated : (props.status || s)
  return { variant, label }
})
</script>

<template>
  <Badge :variant="mapped.variant" :class="props.class">
    {{ mapped.label }}
  </Badge>
</template>
