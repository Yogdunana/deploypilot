<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Shield, Clock, GitBranch, Globe, Server, Layers,
} from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import DeployProgress from '@/components/common/DeployProgress.vue'
import type { App } from '@/types/models'

const props = defineProps<{ app: App }>()
const { t } = useI18n()

const infoItems = computed(() => {
  return [
    { label: t('appDetail.appName'), value: props.app.name, icon: Layers },
    { label: t('appDetail.repoUrl'), value: props.app.repo_url, icon: GitBranch },
    { label: t('appDetail.branch'), value: props.app.branch, icon: GitBranch },
    { label: t('appDetail.stack'), value: props.app.tech_stack, icon: Layers },
    { label: t('appDetail.domain'), value: props.app.domain, icon: Globe },
    { label: t('appDetail.server'), value: String(props.app.server_id), icon: Server },
    { label: t('appDetail.status'), value: props.app.status, icon: Shield, isStatus: true },
    { label: t('appDetail.currentVersion'), value: props.app.current_version || '-', icon: Layers },
    { label: t('appDetail.containerName'), value: props.app.container_name || '-', icon: Server },
    { label: t('appDetail.createdAt'), value: props.app.created_at, icon: Clock, isTime: true },
    { label: t('appDetail.updatedAt'), value: props.app.updated_at, icon: Clock, isTime: true },
  ]
})
</script>

<template>
  <div class="space-y-4">
    <Card>
      <template #header>
        <h3 class="text-sm font-medium text-foreground">{{ t('appDetail.basicInfo') }}</h3>
      </template>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-0">
        <div
          v-for="(item, index) in infoItems"
          :key="item.label"
          class="flex items-start gap-3 py-3"
          :class="index % 2 === 0 ? 'pr-4' : 'pr-4 md:border-l md:border-border md:pl-4'"
        >
          <component :is="item.icon" class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
          <div class="min-w-0">
            <p class="text-xs text-muted-foreground">{{ item.label }}</p>
            <p v-if="item.isStatus" class="text-sm text-foreground mt-0.5">
              <StatusBadge :status="item.value" />
            </p>
            <p v-else-if="item.label === t('appDetail.stack')" class="text-sm text-foreground mt-0.5">
              <Badge v-if="item.value" variant="secondary">{{ item.value }}</Badge>
              <span v-else>-</span>
            </p>
            <p v-else-if="item.isTime" class="text-sm text-foreground mt-0.5">
              <RelativeTime :date="item.value" />
            </p>
            <p v-else class="text-sm text-foreground mt-0.5 truncate">{{ item.value || '-' }}</p>
          </div>
        </div>
      </div>
    </Card>

    <!-- Resource limits -->
    <Card>
      <template #header>
        <h3 class="text-sm font-medium text-foreground">{{ t('appDetail.resourceLimits') }}</h3>
      </template>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <p class="text-xs text-muted-foreground">{{ t('appDetail.memory') }}</p>
          <p class="text-sm text-foreground mt-0.5">{{ app.resource_limits?.memory || '-' }}</p>
        </div>
        <div>
          <p class="text-xs text-muted-foreground">{{ t('appDetail.cpu') }}</p>
          <p class="text-sm text-foreground mt-0.5">{{ app.resource_limits?.cpu || '-' }}</p>
        </div>
      </div>
    </Card>

    <!-- Deploy Progress -->
    <Card>
      <template #header>
        <h3 class="text-sm font-medium text-foreground">{{ t('appDetail.deployProgress') }}</h3>
      </template>
      <DeployProgress :app-id="String(app.id)" />
    </Card>
  </div>
</template>
