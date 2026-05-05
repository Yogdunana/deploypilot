<script setup lang="ts">
import { computed } from 'vue'
import Badge from '@/components/ui/Badge.vue'

interface Props {
  status?: string
  class?: string
}

const props = defineProps<Props>()

const statusMap: Record<string, { variant: 'success' | 'destructive' | 'warning' | 'secondary'; label: string }> = {
  running: { variant: 'success', label: '运行中' },
  success: { variant: 'success', label: '成功' },
  active: { variant: 'success', label: '活跃' },
  healthy: { variant: 'success', label: '健康' },
  online: { variant: 'success', label: '在线' },
  completed: { variant: 'success', label: '已完成' },
  enabled: { variant: 'success', label: '已启用' },
  failed: { variant: 'destructive', label: '失败' },
  error: { variant: 'destructive', label: '错误' },
  expired: { variant: 'destructive', label: '已过期' },
  disabled: { variant: 'destructive', label: '已禁用' },
  offline: { variant: 'destructive', label: '离线' },
  deploying: { variant: 'warning', label: '部署中' },
  pending: { variant: 'warning', label: '等待中' },
  renewing: { variant: 'warning', label: '续期中' },
  building: { variant: 'warning', label: '构建中' },
  warning: { variant: 'warning', label: '警告' },
  stopped: { variant: 'secondary', label: '已停止' },
  unknown: { variant: 'secondary', label: '未知' },
  inactive: { variant: 'secondary', label: '未激活' },
}

const mapped = computed(() => {
  const s = (props.status || '').toLowerCase()
  return statusMap[s] || { variant: 'secondary' as const, label: props.status || s }
})
</script>

<template>
  <Badge :variant="mapped.variant" :class="props.class">
    {{ mapped.label }}
  </Badge>
</template>
