<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { RouterView, RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/composables/useLocale'
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
  KeyRound,
  ShieldCheck,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  User,
  LogOut,
  ChevronRight,
  Languages,
  Menu,
  X,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
const { currentLocale, toggleLocale, localeName } = useLocale()

const collapsed = ref(false)
const sidebarOpen = ref(false)

watch(() => route.path, () => {
  sidebarOpen.value = false
})

watch(sidebarOpen, (val) => {
  if (typeof document !== 'undefined') {
    document.body.style.overflow = val ? 'hidden' : ''
  }
})

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && sidebarOpen.value) {
    sidebarOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = ''
})

const navGroups = computed(() => [
  {
    label: t('layout.overview'),
    items: [
      { path: '/', label: t('layout.dashboard'), icon: LayoutDashboard },
    ],
  },
  {
    label: t('layout.apps'),
    items: [
      { path: '/apps', label: t('layout.apps'), icon: Rocket },
      { path: '/servers', label: t('layout.servers'), icon: Server },
      { path: '/deployments', label: t('layout.deployments'), icon: GitBranch },
    ],
  },
  {
    label: t('layout.infrastructure'),
    items: [
      { path: '/credentials', label: t('layout.credentials'), icon: Key },
      { path: '/dns', label: t('layout.dns'), icon: Globe },
      { path: '/ssl', label: t('layout.ssl'), icon: Shield },
      { path: '/providers', label: t('layout.providers'), icon: Cloud },
    ],
  },
  {
    label: t('layout.ops'),
    items: [
      { path: '/cicd', label: t('layout.cicd'), icon: FileCode },
      { path: '/monitor', label: t('layout.monitor'), icon: Activity },
      { path: '/notifications', label: t('layout.notifications'), icon: Bell },
      { path: '/templates', label: t('layout.templates'), icon: FileCode },
    ],
  },
  {
    label: t('layout.management'),
    items: [
      { path: '/users', label: t('layout.users'), icon: Users },
      { path: '/audit', label: t('layout.audit'), icon: ScrollText },
      { path: '/api-keys', label: t('layout.apiKeys'), icon: KeyRound },
      { path: '/system', label: t('layout.system'), icon: Settings },
      { path: '/settings/security', label: t('layout.security'), icon: ShieldCheck },
    ],
  },
])

const commandOpen = ref(false)

const commandItems = computed(() =>
  navGroups.value.flatMap((group) =>
    group.items.map((item) => ({
      label: item.label,
      icon: item.icon,
      action: () => {
        router.push(item.path)
        commandOpen.value = false
      },
    }))
  )
)

const breadcrumb = computed(() => {
  const matched = route.matched.filter((r) => r.meta?.titleKey || r.name)
  const crumbs: { label: string; path?: string }[] = []
  matched.forEach((r) => {
    const titleKey = r.meta?.titleKey as string | undefined
    const label = titleKey ? t(titleKey) : (r.name as string)
    if (label && label !== 'MainLayout') {
      crumbs.push({ label, path: r.path !== route.path ? r.path : undefined })
    }
  })
  return crumbs
})

function toggleCollapse() { collapsed.value = !collapsed.value }
function handleLogout() { authStore.logout(); router.push('/login') }
function closeMobileSidebar() { sidebarOpen.value = false }
function handleNavClick() { closeMobileSidebar() }
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-background">
    <Transition enter-active-class="transition-opacity duration-200 ease-in-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition-opacity duration-150 ease-in-out" leave-from-class="opacity-100" leave-to-class="opacity-0">
      <div v-if="sidebarOpen" class="fixed inset-0 z-40 bg-black/50 lg:hidden" @click="closeMobileSidebar" />
    </Transition>
    <Transition enter-active-class="transition-transform duration-200 ease-out" enter-from-class="-translate-x-full" enter-to-class="translate-x-0" leave-active-class="transition-transform duration-150 ease-in" leave-from-class="translate-x-0" leave-to-class="-translate-x-full">
      <aside v-if="sidebarOpen" class="fixed inset-y-0 left-0 z-50 flex flex-col w-60 border-r border-border bg-card lg:hidden">
        <div class="flex h-12 items-center justify-between px-4 border-b border-border">
          <div class="flex items-center gap-2">
            <img src="/icon.svg" alt="DeployPilot" class="h-7 w-7 rounded shrink-0 object-contain" />
            <span class="text-sm font-semibold text-foreground whitespace-nowrap">DeployPilot</span>
          </div>
          <button class="inline-flex items-center justify-center w-8 h-8 rounded-md text-muted-foreground hover:bg-accent hover:text-foreground transition-colors cursor-pointer" :aria-label="t('layout.closeMenu')" @click="closeMobileSidebar"><X class="w-4 h-4" /></button>
        </div>
        <ScrollArea class="flex-1 py-2">
          <div v-for="group in navGroups" :key="group.label" class="mb-2">
            <div class="px-4 py-1.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{{ group.label }}</div>
            <nav class="px-2">
              <RouterLink v-for="item in group.items" :key="item.path" :to="item.path" :class="cn('flex items-center gap-3 rounded-md px-2 py-2 text-sm transition-colors duration-100', route.path === item.path || (item.path !== '/' && route.path.startsWith(item.path)) ? 'bg-accent text-foreground' : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground')" @click="handleNavClick">
                <component :is="item.icon" class="w-4 h-4 shrink-0" />
                <span class="truncate">{{ item.label }}</span>
              </RouterLink>
            </nav>
          </div>
        </ScrollArea>
        <div class="border-t border-border p-2">
          <DropdownMenu :items="[{ label: t('layout.profile'), icon: User, action: () => {} }, { label: t('layout.logout'), icon: LogOut, action: handleLogout, danger: true }]">
            <template #trigger>
              <div class="flex items-center gap-3 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors duration-100 cursor-pointer">
                <Avatar :name="authStore.currentUser?.username || ''" size="sm" />
                <div class="flex flex-col items-start min-w-0">
                  <span class="truncate text-foreground text-xs font-medium">{{ authStore.currentUser?.username || t('layout.profile') }}</span>
                  <span class="truncate text-[11px]">{{ authStore.currentUser?.email || '' }}</span>
                </div>
              </div>
            </template>
          </DropdownMenu>
        </div>
      </aside>
    </Transition>
    <aside :class="cn('hidden lg:flex flex-col border-r border-border bg-card transition-all duration-200 ease-in-out shrink-0', collapsed ? 'w-16' : 'w-60')">
      <div class="flex h-12 items-center gap-2 px-4 border-b border-border">
        <img src="/icon.svg" alt="DeployPilot" class="h-7 w-7 rounded shrink-0 object-contain" />
        <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
          <span v-if="!collapsed" class="text-sm font-semibold text-foreground whitespace-nowrap">DeployPilot</span>
        </Transition>
      </div>
      <ScrollArea class="flex-1 py-2">
        <div v-for="group in navGroups" :key="group.label" class="mb-2">
          <div v-if="!collapsed" class="px-4 py-1.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{{ group.label }}</div>
          <div v-else class="px-2 py-1.5"><Separator /></div>
          <nav class="px-2">
            <RouterLink v-for="item in group.items" :key="item.path" :to="item.path" :class="cn('flex items-center gap-3 rounded-md px-2 py-1.5 text-sm transition-colors duration-100 group', collapsed && 'justify-center px-0', route.path === item.path || (item.path !== '/' && route.path.startsWith(item.path)) ? 'bg-accent text-foreground' : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground')">
              <component :is="item.icon" class="w-4 h-4 shrink-0" />
              <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
              </Transition>
            </RouterLink>
          </nav>
        </div>
      </ScrollArea>
      <div class="border-t border-border p-2">
        <button :class="cn('flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors duration-100 cursor-pointer', collapsed && 'justify-center px-0')" @click="toggleCollapse">
          <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="w-4 h-4 shrink-0" />
          <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <span v-if="!collapsed">{{ t('layout.collapseSidebar') }}</span>
          </Transition>
        </button>
        <Separator class="my-2" />
        <DropdownMenu :items="[{ label: t('layout.profile'), icon: User, action: () => {} }, { label: t('layout.logout'), icon: LogOut, action: handleLogout, danger: true }]">
          <template #trigger>
            <div :class="cn('flex items-center gap-3 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors duration-100 cursor-pointer', collapsed && 'justify-center px-0')">
              <Avatar :name="authStore.currentUser?.username || ''" size="sm" />
              <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="!collapsed" class="flex flex-col items-start min-w-0">
                  <span class="truncate text-foreground text-xs font-medium">{{ authStore.currentUser?.username || t('layout.profile') }}</span>
                  <span class="truncate text-[11px]">{{ authStore.currentUser?.email || '' }}</span>
                </div>
              </Transition>
            </div>
          </template>
        </DropdownMenu>
      </div>
    </aside>
    <div class="flex flex-1 flex-col overflow-hidden min-w-0">
      <header class="flex h-12 items-center justify-between border-b border-border bg-card px-3 sm:px-4 shrink-0">
        <div class="flex items-center gap-2 min-w-0">
          <button class="inline-flex items-center justify-center w-8 h-8 rounded-md text-muted-foreground hover:bg-accent hover:text-foreground transition-colors cursor-pointer shrink-0 lg:hidden" :aria-label="t('layout.openMenu')" @click="sidebarOpen = true"><Menu class="w-5 h-5" /></button>
          <div class="flex items-center gap-1 text-sm min-w-0 overflow-hidden">
            <template v-for="(crumb, index) in breadcrumb" :key="index">
              <span :class="cn('text-muted-foreground truncate', index === breadcrumb.length - 1 && 'text-foreground font-medium')">{{ crumb.label }}</span>
              <ChevronRight v-if="index < breadcrumb.length - 1" class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
            </template>
          </div>
        </div>
        <div class="flex items-center gap-1.5 sm:gap-2 shrink-0">
          <button class="inline-flex items-center justify-center gap-1.5 rounded-md border border-border bg-background px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors duration-150 cursor-pointer" @click="toggleLocale"><Languages class="w-3.5 h-3.5" /><span class="hidden sm:inline">{{ localeName }}</span></button>
          <button class="inline-flex items-center justify-center gap-2 rounded-md border border-border bg-background px-2.5 sm:px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors duration-150 cursor-pointer" @click="commandOpen = true"><Search class="w-3.5 h-3.5" /><span class="hidden sm:inline">{{ t('layout.search') }}</span><kbd class="hidden md:inline-flex h-5 items-center gap-1 rounded border border-border bg-card px-1.5 text-[10px] font-medium text-muted-foreground"><span class="text-xs">&#8984;</span>K</kbd></button>
        </div>
      </header>
      <main class="flex-1 overflow-auto">
        <div class="p-3 sm:p-4 lg:p-6">
          <RouterView v-slot="{ Component, route: currentRoute }">
            <Transition mode="out-in" enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0 translate-y-1" enter-to-class="opacity-100 translate-y-0" leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100 translate-y-0" leave-to-class="opacity-0 -translate-y-1">
              <component :is="Component" :key="currentRoute.path" />
            </Transition>
          </RouterView>
        </div>
      </main>
    </div>
    <Command v-model:open="commandOpen" :items="commandItems" :placeholder="t('layout.searchPlaceholder')" />
    <Toast />
  </div>
</template>
