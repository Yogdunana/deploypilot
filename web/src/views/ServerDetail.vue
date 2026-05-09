<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowLeft, Zap, Cpu, Terminal, Server as ServerIcon, Globe, Hash,
  Clock, Tag, Shield, HardDrive,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Card from '@/components/ui/Card.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Separator from '@/components/ui/Separator.vue'
import * as serversApi from '@/api/modules/servers'
import type { Server } from '@/types/models'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = inject<any>('toast')!

// State
const server = ref<Server | null>(null)
const loading = ref(true)
const testing = ref(false)
const detecting = ref(false)

// Map server status
function mapServerStatus(status: string): string {
  const s = status.toLowerCase()
  if (s === 'reachable' || s === 'online' || s === 'connected') return 'success'
  if (s === 'unreachable' || s === 'offline') return 'destructive'
  return 'secondary'
}

// Fetch server detail
async function fetchServer() {
  loading.value = true
  try {
    // Use list and filter since there's no single get endpoint
    const res = await serversApi.list({ page: 1, page_size: 1000 })
    if (res.data.status === 'success') {
      const found = res.data.data.find((s: Server) => s.id === Number(props.id))
      if (found) {
        server.value = found
      } else {
        toast(t('serverDetail.serverNotFound'), 'destructive')
        router.push('/servers')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('serverDetail.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Test connection
async function handleTestConnection() {
  if (!server.value) return
  testing.value = true
  try {
    const res = await serversApi.test(server.value.id)
    if (res.data.status === 'success' && res.data.data.success) {
      toast(t('serverDetail.connectionSuccess'), 'success')
    } else {
      toast(res.data.data?.message || t('serverDetail.connectionFailed'), 'destructive')
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('serverDetail.connectionTestFailed'), 'destructive')
  } finally {
    testing.value = false
  }
}

// Detect environment
async function handleDetect() {
  if (!server.value) return
  detecting.value = true
  try {
    await serversApi.detect(server.value.id, { host: server.value.host, port: server.value.port })
    toast(t('serverDetail.detectTriggered'), 'success')
    fetchServer()
  } catch (err: any) {
    toast(err.response?.data?.message || t('serverDetail.detectFailed'), 'destructive')
  } finally {
    detecting.value = false
  }
}

// Open terminal
function openTerminal() {
  router.push(`/servers/${props.id}/terminal`)
}

onMounted(fetchServer)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader>
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push('/servers')">
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <div>
            <div class="flex items-center gap-2">
              <h1 class="text-xl font-semibold text-foreground">
                {{ server?.name || t('serverDetail.loading') }}
              </h1>
              <StatusBadge v-if="server" :status="mapServerStatus(server.status)" />
            </div>
            <p v-if="server" class="mt-0.5 text-sm text-muted-foreground font-mono">
              {{ server.host }}:{{ server.port }}
            </p>
          </div>
        </div>
      </template>
      <template #actions>
        <Button :loading="testing" @click="handleTestConnection">
          <template #icon><Zap class="w-4 h-4" /></template>
          {{ t('serverDetail.testConnection') }}
        </Button>
        <Button variant="outline" :loading="detecting" @click="handleDetect">
          <template #icon><Cpu class="w-4 h-4" /></template>
          {{ t('serverDetail.detect') }}
        </Button>
        <Button variant="outline" @click="openTerminal">
          <template #icon><Terminal class="w-4 h-4" /></template>
          {{ t('serverDetail.openTerminal') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-4">
      <Skeleton class="h-10 w-96" />
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Skeleton v-for="i in 4" :key="i" class="h-32" />
      </div>
    </div>

    <!-- Server info -->
    <template v-else-if="server">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Basic info card -->
        <Card>
          <template #header>
            <h3 class="text-sm font-medium text-foreground">{{ t('serverDetail.basicInfo') }}</h3>
          </template>
          <div class="space-y-3">
            <div class="flex items-start gap-3">
              <ServerIcon class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.name') }}</p>
                <p class="text-sm text-foreground">{{ server.name }}</p>
              </div>
            </div>
            <Separator />
            <div class="flex items-start gap-3">
              <Globe class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.host') }}</p>
                <p class="text-sm text-foreground font-mono">{{ server.host }}</p>
              </div>
            </div>
            <Separator />
            <div class="flex items-start gap-3">
              <Hash class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.port') }}</p>
                <p class="text-sm text-foreground font-mono">{{ server.port }}</p>
              </div>
            </div>
            <Separator />
            <div class="flex items-start gap-3">
              <Shield class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.status') }}</p>
                <StatusBadge :status="mapServerStatus(server.status)" />
              </div>
            </div>
          </div>
        </Card>

        <!-- Tags & time card -->
        <Card>
          <template #header>
            <h3 class="text-sm font-medium text-foreground">{{ t('serverDetail.tagsAndTime') }}</h3>
          </template>
          <div class="space-y-3">
            <div class="flex items-start gap-3">
              <Tag class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.tags') }}</p>
                <div v-if="server.tags && server.tags.length > 0" class="flex items-center gap-1 flex-wrap mt-1">
                  <Badge v-for="tag in server.tags" :key="tag" variant="outline">
                    {{ tag }}
                  </Badge>
                </div>
                <p v-else class="text-sm text-muted-foreground">-</p>
              </div>
            </div>
            <Separator />
            <div class="flex items-start gap-3">
              <Clock class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.createdAt') }}</p>
                <p class="text-sm text-foreground">
                  <RelativeTime :date="server.created_at" />
                </p>
              </div>
            </div>
            <Separator />
            <div class="flex items-start gap-3">
              <Clock class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
              <div>
                <p class="text-xs text-muted-foreground">{{ t('serverDetail.updatedAt') }}</p>
                <p class="text-sm text-foreground">
                  <RelativeTime :date="server.updated_at" />
                </p>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <!-- Detected environment info -->
      <Card v-if="server.detected_info">
        <template #header>
          <h3 class="text-sm font-medium text-foreground">{{ t('serverDetail.envInfo') }}</h3>
        </template>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div class="flex items-start gap-3">
            <HardDrive class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.os') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.os || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Cpu class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.arch') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.arch || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <ServerIcon class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.dockerVersion') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.docker_version || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <ServerIcon class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.dockerComposeVersion') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.docker_compose_version || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <HardDrive class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.kernelVersion') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.kernel_version || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <HardDrive class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.memory') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.memory_total || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Cpu class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.cpuCores') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.cpu_cores || '-' }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <HardDrive class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div>
              <p class="text-xs text-muted-foreground">{{ t('serverDetail.disk') }}</p>
              <p class="text-sm text-foreground">{{ server.detected_info.disk_total || '-' }}</p>
            </div>
          </div>
        </div>
      </Card>
    </template>
  </div>
</template>
