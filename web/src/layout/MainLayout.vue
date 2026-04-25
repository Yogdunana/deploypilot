<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RouterView, RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { cn } from '@/lib/utils'
import Avatar from '@/components/ui/Avatar.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Command from '@/components/ui/Command.vue'
import Toast from '@/components/ui/Toast.vue'
import ScrollArea from '@/components/ui/ScrollArea.vue'
import Separator from '@/components/ui/Separator.vue'
import {
  LayoutDashboard,
  Rocket,
  Server,
  GitBranch,
  Key,
  Globe,
  Shield,
  Cloud,
  Activity,
  Bell,
  FileCode,
  Users,
  ScrollText,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  User,
  LogOut,
  ChevronRight,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const collapsed = ref(false)

const navGroups = [
  {
    label: '概览',
    items: [
      { path: '/', label: '仪表盘', icon: LayoutDashboard },
    ],
  },
  {
    label: '应用',
    items: [
      { path: '/apps', label: '应用管理', icon: Rocket },
      { path: '/servers', label: '服务器', icon: Server },
      { path: '/deployments', label: '部署记录', icon: GitBranch },
    ],
  },
  {
    label: '基础设施',
    items: [
      { path: '/credentials', label: '凭证管理', icon: Key },
      { path: '/dns', label: 'DNS 管理', icon: Globe },
      { path: '/ssl', label: 'SSL 证书', icon: Shield },
      { path: '/providers', label: '服务提供商', icon: Cloud },
    ],
  },
  {
    label: '运维',
    items: [
      { path: '/cicd', label: 'CI/CD', icon: FileCode },
      { path: '/monitor', label: '监控', icon: Activity },
      { path: '/notifications', label: '通知', icon: Bell },
      { path: '/templates', label: '模板', icon: FileCode },
    ],
  },
  {
    label: '管理',
    items: [
      { path: '/users', label: '用户管理', icon: Users },
      { path: '/audit', label: '审计日志', icon: ScrollText },
      { path: '/system', label: '系统设置', icon: Settings },
    ],
  },
]

const commandOpen = ref(false)

const commandItems = computed(() =>
  navGroups.flatMap((group) =>
    group.items.map((item) => ({
      label: item.label,
      icon: item.icon,
      action: () => router.push(item.path),
    }))
  )
)

const breadcrumb = computed(() => {
  const matched = route.matched.filter((r) => r.meta?.title || r.name)
  const crumbs: { label: string; path?: string }[] = []
  matched.forEach((r) => {
    const label = (r.meta?.title as string) || (r.name as string)
    if (label && label !== 'MainLayout') {
      crumbs.push({
        label,
        path: r.path !== route.path ? r.path : undefined,
      })
    }
  })
  return crumbs
})

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-background">
    <!-- Sidebar -->
    <aside
      :class="cn(
        'flex flex-col border-r border-border bg-card transition-all duration-200 ease-in-out shrink-0',
        collapsed ? 'w-16' : 'w-60'
      )"
    >
      <!-- Logo -->
      <div class="flex h-12 items-center gap-2 px-4 border-b border-border">
        <div class="flex items-center justify-center w-7 h-7 rounded-md bg-primary text-primary-foreground font-bold text-sm shrink-0">
          D
        </div>
        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0"
          enter-to-class="opacity-100"
          leave-active-class="transition duration-75 ease-in"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0"
        >
          <span
            v-if="!collapsed"
            class="text-sm font-semibold text-foreground whitespace-nowrap"
          >
            DeployPilot
          </span>
        </Transition>
      </div>

      <!-- Navigation -->
      <ScrollArea class="flex-1 py-2">
        <div v-for="group in navGroups" :key="group.label" class="mb-2">
          <div
            v-if="!collapsed"
            class="px-4 py-1.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground"
          >
            {{ group.label }}
          </div>
          <div v-else class="px-2 py-1.5">
            <Separator />
          </div>
          <nav class="px-2">
            <RouterLink
              v-for="item in group.items"
              :key="item.path"
              :to="item.path"
              :class="cn(
                'flex items-center gap-3 rounded-md px-2 py-1.5 text-sm transition-colors duration-100 group',
                collapsed && 'justify-center px-0',
                route.path === item.path || (item.path !== '/' && route.path.startsWith(item.path))
                  ? 'bg-accent text-foreground'
                  : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
              )"
            >
              <component :is="item.icon" class="w-4 h-4 shrink-0" />
              <Transition
                enter-active-class="transition duration-150 ease-out"
                enter-from-class="opacity-0"
                enter-to-class="opacity-100"
                leave-active-class="transition duration-75 ease-in"
                leave-from-class="opacity-100"
                leave-to-class="opacity-0"
              >
                <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
              </Transition>
            </RouterLink>
          </nav>
        </div>
      </ScrollArea>

      <!-- Bottom section -->
      <div class="border-t border-border p-2">
        <button
          :class="cn(
            'flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors duration-100 cursor-pointer',
            collapsed && 'justify-center px-0'
          )"
          @click="toggleCollapse"
        >
          <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="w-4 h-4 shrink-0" />
          <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0"
            enter-to-class="opacity-100"
            leave-active-class="transition duration-75 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
          >
            <span v-if="!collapsed">收起侧栏</span>
          </Transition>
        </button>
        <Separator class="my-2" />
        <DropdownMenu
          :items="[
            { label: '个人设置', icon: User, action: () => {} },
            { label: '退出登录', icon: LogOut, action: handleLogout, danger: true },
          ]"
        >
          <template #trigger>
            <div
              :class="cn(
                'flex items-center gap-3 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors duration-100 cursor-pointer',
                collapsed && 'justify-center px-0'
              )"
            >
              <Avatar
                :name="authStore.currentUser?.username || ''"
                size="sm"
              />
              <Transition
                enter-active-class="transition duration-150 ease-out"
                enter-from-class="opacity-0"
                enter-to-class="opacity-100"
                leave-active-class="transition duration-75 ease-in"
                leave-from-class="opacity-100"
                leave-to-class="opacity-0"
              >
                <div v-if="!collapsed" class="flex flex-col items-start min-w-0">
                  <span class="truncate text-foreground text-xs font-medium">
                    {{ authStore.currentUser?.username || '用户' }}
                  </span>
                  <span class="truncate text-[11px]">
                    {{ authStore.currentUser?.email || '' }}
                  </span>
                </div>
              </Transition>
            </div>
          </template>
        </DropdownMenu>
      </div>
    </aside>

    <!-- Main content area -->
    <div class="flex flex-1 flex-col overflow-hidden">
      <!-- Top bar -->
      <header class="flex h-12 items-center justify-between border-b border-border bg-card px-4 shrink-0">
        <div class="flex items-center gap-1 text-sm">
          <template v-for="(crumb, index) in breadcrumb" :key="index">
            <span
              :class="cn(
                'text-muted-foreground',
                index === breadcrumb.length - 1 && 'text-foreground font-medium'
              )"
            >
              {{ crumb.label }}
            </span>
            <ChevronRight
              v-if="index < breadcrumb.length - 1"
              class="w-3.5 h-3.5 text-muted-foreground"
            />
          </template>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex items-center gap-2 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors duration-150 cursor-pointer"
            @click="commandOpen = true"
          >
            <Search class="w-3.5 h-3.5" />
            <span class="hidden sm:inline">搜索...</span>
            <kbd class="hidden sm:inline-flex h-5 items-center gap-1 rounded border border-border bg-card px-1.5 text-[10px] font-medium text-muted-foreground">
              <span class="text-xs">&#8984;</span>K
            </kbd>
          </button>
        </div>
      </header>

      <!-- Content area -->
      <main class="flex-1 overflow-auto">
        <div class="p-6">
          <RouterView v-slot="{ Component, route: currentRoute }">
            <Transition
              mode="out-in"
              enter-active-class="transition duration-150 ease-out"
              enter-from-class="opacity-0 translate-y-1"
              enter-to-class="opacity-100 translate-y-0"
              leave-active-class="transition duration-100 ease-in"
              leave-from-class="opacity-100 translate-y-0"
              leave-to-class="opacity-0 -translate-y-1"
            >
              <component :is="Component" :key="currentRoute.path" />
            </Transition>
          </RouterView>
        </div>
      </main>
    </div>

    <!-- Command palette -->
    <Command
      v-model:open="commandOpen"
      :items="commandItems"
      placeholder="搜索页面或输入命令..."
    />

    <!-- Toast provider -->
    <Toast />
  </div>
</template>
