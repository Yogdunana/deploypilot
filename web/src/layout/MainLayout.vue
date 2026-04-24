<template>
  <el-container class="layout-container">
    <el-aside :width="isCollapsed ? '64px' : '220px'" class="sidebar">
      <div class="logo">
        <el-icon :size="24"><Promotion /></el-icon>
        <span v-show="!isCollapsed" class="logo-text">DeployPilot</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapsed"
        router
        background-color="#001529"
        text-color="#ffffffa6"
        active-text-color="#ffffff"
        class="sidebar-menu"
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataAnalysis /></el-icon>
          <template #title>Dashboard</template>
        </el-menu-item>
        <el-menu-item index="/apps">
          <el-icon><Monitor /></el-icon>
          <template #title>Applications</template>
        </el-menu-item>
        <el-menu-item index="/servers">
          <el-icon><Connection /></el-icon>
          <template #title>Servers</template>
        </el-menu-item>
        <el-menu-item index="/credentials">
          <el-icon><Key /></el-icon>
          <template #title>Credentials</template>
        </el-menu-item>
        <el-menu-item index="/dns">
          <el-icon><Link /></el-icon>
          <template #title>DNS Records</template>
        </el-menu-item>
        <el-menu-item index="/providers">
          <el-icon><Box /></el-icon>
          <template #title>Providers</template>
        </el-menu-item>
        <el-menu-item index="/notifications">
          <el-icon><Bell /></el-icon>
          <template #title>Notifications</template>
        </el-menu-item>
        <el-menu-item index="/templates">
          <el-icon><Document /></el-icon>
          <template #title>Templates</template>
        </el-menu-item>
        <el-menu-item index="/deployments">
          <el-icon><Upload /></el-icon>
          <template #title>Deployments</template>
        </el-menu-item>
        <el-menu-item index="/users">
          <el-icon><User /></el-icon>
          <template #title>Users</template>
        </el-menu-item>
        <el-menu-item index="/system">
          <el-icon><Setting /></el-icon>
          <template #title>System</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="topbar">
        <div class="topbar-left">
          <el-icon
            class="collapse-btn"
            :size="20"
            @click="isCollapsed = !isCollapsed"
          >
            <Fold v-if="!isCollapsed" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">Home</el-breadcrumb-item>
            <el-breadcrumb-item v-if="$route.meta.title">
              {{ $route.meta.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="topbar-right">
          <span class="username">{{ currentUser }}</span>
          <el-button type="danger" text @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            Logout
          </el-button>
        </div>
      </el-header>

      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { removeToken } from '../utils/auth'
import { usersApi } from '../api'

const route = useRoute()
const router = useRouter()
const isCollapsed = ref(false)
const currentUser = ref('User')

const activeMenu = computed(() => {
  const path = route.path
  if (path.startsWith('/apps/')) return '/apps'
  return path
})

async function fetchUser() {
  try {
    const data = await usersApi.me()
    currentUser.value = data.username || 'User'
  } catch {
    // ignore
  }
}

function handleLogout() {
  removeToken()
  router.push('/login')
}

fetchUser()
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.sidebar {
  background-color: #001529;
  transition: width 0.3s;
  overflow: hidden;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  border-bottom: 1px solid #ffffff1a;
}

.logo-text {
  white-space: nowrap;
}

.sidebar-menu {
  border-right: none;
  height: calc(100vh - 60px);
  overflow-y: auto;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 220px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e8e8e8;
  background: #fff;
  padding: 0 20px;
  height: 60px;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.collapse-btn {
  cursor: pointer;
  color: #333;
}

.collapse-btn:hover {
  color: #409eff;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.username {
  font-size: 14px;
  color: #333;
}

.main-content {
  background-color: #f0f2f5;
  min-height: calc(100vh - 60px);
}
</style>
