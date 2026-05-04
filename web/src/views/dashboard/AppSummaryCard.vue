<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Layers, CirclePlay, Server as ServerIcon, Rocket } from 'lucide-vue-next'
import Skeleton from '@/components/ui/Skeleton.vue'
import type { App, Server, DeploymentRecord } from '@/types/models'

const { t } = useI18n()

const props = defineProps<{
  apps: App[]
  servers: Server[]
  recentDeployments: DeploymentRecord[]
  loading: boolean
}>()

const totalApps = computed(() => props.apps.length)
const runningApps = computed(() => props.apps.filter(a => a.status === 'running').length)
const totalServers = computed(() => props.servers.length)
const recentDeployCount = computed(() => props.recentDeployments.length)
</script>

<template>
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Total Apps -->
    <div
      class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
      style="border-left: 3px solid hsl(var(--primary))"
    >
      <div class="flex items-center justify-between">
        <div>
          <p class="text-[13px] text-muted-foreground">{{ t('dashboard.totalApps') }}</p>
          <p v-if="loading" class="mt-1">
            <Skeleton class="h-8 w-16" variant="text" />
          </p>
          <p v-else class="text-[28px] font-bold text-foreground leading-tight mt-1">
            {{ totalApps }}
          </p>
        </div>
        <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-primary/10">
          <Layers class="w-5 h-5 text-primary" />
        </div>
      </div>
    </div>

    <!-- Running Apps -->
    <div
      class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
      style="border-left: 3px solid hsl(var(--success))"
    >
      <div class="flex items-center justify-between">
        <div>
          <p class="text-[13px] text-muted-foreground">{{ t('dashboard.runningApps') }}</p>
          <p v-if="loading" class="mt-1">
            <Skeleton class="h-8 w-16" variant="text" />
          </p>
          <p v-else class="text-[28px] font-bold text-success leading-tight mt-1">
            {{ runningApps }}
          </p>
        </div>
        <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-success/10">
          <CirclePlay class="w-5 h-5 text-success" />
        </div>
      </div>
    </div>

    <!-- Total Servers -->
    <div
      class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
      style="border-left: 3px solid hsl(var(--warning))"
    >
      <div class="flex items-center justify-between">
        <div>
          <p class="text-[13px] text-muted-foreground">{{ t('dashboard.totalServers') }}</p>
          <p v-if="loading" class="mt-1">
            <Skeleton class="h-8 w-16" variant="text" />
          </p>
          <p v-else class="text-[28px] font-bold text-foreground leading-tight mt-1">
            {{ totalServers }}
          </p>
        </div>
        <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-warning/10">
          <ServerIcon class="w-5 h-5 text-warning" />
        </div>
      </div>
    </div>

    <!-- Recent Deployments -->
    <div
      class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
      style="border-left: 3px solid hsl(221, 83%, 53%)"
    >
      <div class="flex items-center justify-between">
        <div>
          <p class="text-[13px] text-muted-foreground">{{ t('dashboard.recentDeploys') }}</p>
          <p v-if="loading" class="mt-1">
            <Skeleton class="h-8 w-16" variant="text" />
          </p>
          <p v-else class="text-[28px] font-bold text-foreground leading-tight mt-1">
            {{ recentDeployCount }}
          </p>
        </div>
        <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-blue-500/10">
          <Rocket class="w-5 h-5 text-blue-500" />
        </div>
      </div>
    </div>
  </div>
</template>
