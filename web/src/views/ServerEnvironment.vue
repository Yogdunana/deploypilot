<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useRouter } from 'vue-router'
import {
  ArrowLeft, RefreshCw, Monitor, Cpu, HardDrive, Server as ServerIcon,
  Globe, Layers, Shield, Hash, Network,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Separator from '@/components/ui/Separator.vue'
import Badge from '@/components/ui/Badge.vue'
import * as serversApi from '@/api/modules/servers'
import type { Server } from '@/types/models'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = useToast()

const serverName = ref('')
const loading = ref(true)
const detecting = ref(false)

interface EnvironmentInfo {
  os?: string
  arch?: string
  docker_version?: string
  docker_compose_version?: string
  kernel_version?: string
  memory_total?: string
  cpu_cores?: number
  disk_total?: string
  open_ports?: number[]
  installed_services?: string[]
}

const envInfo = ref<EnvironmentInfo | null>(null)

// 信息项定义
const infoSections = computed(() => {
  if (!envInfo.value) return []
  return [
    {
      title: t('serverEnvironment.systemInfo'),
      items: [
        { label: t('serverEnvironment.os'), value: envInfo.value.os || '-', icon: Monitor },
        { label: t('serverEnvironment.arch'), value: envInfo.value.arch || '-', icon: Layers },
        { label: t('serverEnvironment.kernelVersion'), value: envInfo.value.kernel_version || '-', icon: Shield },
      ],
    },
    {
      title: t('serverEnvironment.hardwareResources'),
      items: [
        { label: t('serverEnvironment.cpuCores'), value: String(envInfo.value.cpu_cores || '-'), icon: Cpu },
        { label: t('serverEnvironment.memory'), value: envInfo.value.memory_total || '-', icon: HardDrive },
        { label: t('serverEnvironment.disk'), value: envInfo.value.disk_total || '-', icon: HardDrive },
      ],
    },
    {
      title: t('serverEnvironment.containerEnv'),
      items: [
        { label: t('serverEnvironment.dockerVersion'), value: envInfo.value.docker_version || '-', icon: ServerIcon },
        { label: t('serverEnvironment.dockerComposeVersion'), value: envInfo.value.docker_compose_version || '-', icon: Layers },
      ],
    },
  ]
})

// 获取环境信息
async function fetchEnvironment() {
  loading.value = true
  try {
    const res = await serversApi.getEnvironment(props.id)
    if (res.data.status === 'success') {
      envInfo.value = res.data.data || null
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('serverEnvironment.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// 重新检测
async function handleRedetect() {
  detecting.value = true
  try {
    // 先获取服务器信息
    const serverRes = await serversApi.list({ page: 1, page_size: 1000 })
    if (serverRes.data.status === 'success') {
      const found = (serverRes.data.data as Server[]).find((s) => s.id === props.id)
      if (found) {
        await serversApi.detect(found.id, { host: found.host, port: found.port })
        toast(t('serverEnvironment.detectTriggered'), 'success')
        // 等待一会儿再刷新
        setTimeout(() => {
          fetchEnvironment()
        }, 2000)
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('serverEnvironment.detectFailed'), 'destructive')
  } finally {
    detecting.value = false
  }
}

// 获取服务器信息
async function fetchServer() {
  try {
    const res = await serversApi.list({ page: 1, page_size: 1000 })
    if (res.data.status === 'success') {
      const found = (res.data.data as Server[]).find((s) => s.id === props.id)
      if (found) {
        serverName.value = found.name
      } else {
        toast(t('serverEnvironment.serverNotFound'), 'destructive')
        router.push('/servers')
      }
    }
  } catch {
    // 静默处理
  }
}

onMounted(() => {
  fetchServer()
  fetchEnvironment()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader>
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push(`/servers/${props.id}`)">
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <div>
            <h1 class="text-xl font-semibold text-foreground">
{{ t('serverEnvironment.title', { name: serverName }) }}
            </h1>
            <p class="mt-0.5 text-sm text-muted-foreground">
{{ t('serverEnvironment.subtitle') }}
            </p>
          </div>
        </div>
      </template>
      <template #actions>
        <Button variant="outline" :loading="detecting" @click="handleRedetect">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          {{ t('serverEnvironment.redetect') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 加载状态 -->
    <div v-if="loading" class="space-y-4">
      <Skeleton v-for="i in 3" :key="i" class="h-40" />
    </div>

    <!-- 环境信息 -->
    <template v-else-if="envInfo">
      <!-- 信息卡片 -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card v-for="section in infoSections" :key="section.title">
          <template #header>
            <h3 class="text-sm font-medium text-foreground">{{ section.title }}</h3>
          </template>
          <div class="space-y-3">
            <template v-for="(item, index) in section.items" :key="item.label">
              <div class="flex items-start gap-3">
                <component :is="item.icon" class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
                <div class="min-w-0">
                  <p class="text-xs text-muted-foreground">{{ item.label }}</p>
                  <p class="text-sm text-foreground mt-0.5 font-mono">{{ item.value }}</p>
                </div>
              </div>
              <Separator v-if="index < section.items.length - 1" />
            </template>
          </div>
        </Card>
      </div>

      <!-- 开放端口 -->
      <Card v-if="envInfo.open_ports && envInfo.open_ports.length > 0">
        <template #header>
          <div class="flex items-center gap-2">
            <Network class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-medium text-foreground">{{ t('serverEnvironment.openPorts') }}</h3>
          </div>
        </template>
        <div class="flex flex-wrap gap-2">
          <Badge
            v-for="port in envInfo.open_ports"
            :key="port"
            variant="outline"
            class="font-mono"
          >
            {{ port }}
          </Badge>
        </div>
      </Card>

      <!-- 已安装服务 -->
      <Card v-if="envInfo.installed_services && envInfo.installed_services.length > 0">
        <template #header>
          <div class="flex items-center gap-2">
            <ServerIcon class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-medium text-foreground">{{ t('serverEnvironment.installedServices') }}</h3>
          </div>
        </template>
        <div class="flex flex-wrap gap-2">
          <Badge
            v-for="service in envInfo.installed_services"
            :key="service"
            variant="secondary"
          >
            {{ service }}
          </Badge>
        </div>
      </Card>
    </template>

    <!-- 无数据 -->
    <Card v-else>
      <div class="flex flex-col items-center justify-center py-16 text-center">
        <Monitor class="w-12 h-12 text-muted-foreground mb-4" />
        <h3 class="text-sm font-medium text-foreground">{{ t('serverEnvironment.noEnvInfo') }}</h3>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('serverEnvironment.noEnvInfoDesc') }}</p>
      </div>
    </Card>
  </div>
</template>
