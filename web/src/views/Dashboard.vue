<template>
  <div>
    <h2 style="margin-top: 0">Dashboard</h2>

    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <el-icon :size="24" color="#409eff"><Monitor /></el-icon>
              <span>Total Apps</span>
            </div>
          </template>
          <div class="card-value">{{ stats.totalApps }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <el-icon :size="24" color="#67c23a"><CircleCheck /></el-icon>
              <span>Running Apps</span>
            </div>
          </template>
          <div class="card-value">{{ stats.runningApps }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <el-icon :size="24" color="#e6a23c"><Connection /></el-icon>
              <span>Total Servers</span>
            </div>
          </template>
          <div class="card-value">{{ stats.totalServers }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <el-icon :size="24" color="#909399"><Upload /></el-icon>
              <span>Recent Deployments</span>
            </div>
          </template>
          <div class="card-value">{{ stats.recentDeployments }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>Recent Deployments</span>
              <el-button text type="primary" @click="$router.push('/deployments')">
                View All
              </el-button>
            </div>
          </template>
          <el-table :data="recentDeployments" stripe style="width: 100%">
            <el-table-column prop="app_name" label="App" />
            <el-table-column prop="status" label="Status">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="Time" width="180">
              <template #default="{ row }">
                {{ formatDate(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="recentDeployments.length === 0" description="No deployments yet" />
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span>Quick Actions</span>
          </template>
          <div class="quick-actions">
            <el-button type="primary" size="large" style="width: 100%; margin-bottom: 12px" @click="$router.push('/apps')">
              <el-icon><Plus /></el-icon>
              Deploy App
            </el-button>
            <el-button type="success" size="large" style="width: 100%; margin-bottom: 12px" @click="$router.push('/servers')">
              <el-icon><Plus /></el-icon>
              Add Server
            </el-button>
            <el-button type="warning" size="large" style="width: 100%" @click="$router.push('/templates')">
              <el-icon><Document /></el-icon>
              Browse Templates
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { appsApi, serversApi, deploymentsApi } from '../api'

const stats = reactive({
  totalApps: 0,
  runningApps: 0,
  totalServers: 0,
  recentDeployments: 0,
})

const recentDeployments = ref([])

function statusType(status) {
  const map = {
    success: 'success',
    running: 'warning',
    failed: 'danger',
    pending: 'info',
  }
  return map[status] || 'info'
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

onMounted(async () => {
  try {
    const [apps, servers, deployments] = await Promise.all([
      appsApi.list().catch(() => []),
      serversApi.list().catch(() => []),
      deploymentsApi.list().catch(() => []),
    ])

    const appsList = Array.isArray(apps) ? apps : apps.data || []
    const serversList = Array.isArray(servers) ? servers : servers.data || []
    const deploymentsList = Array.isArray(deployments) ? deployments : deployments.data || []

    stats.totalApps = appsList.length
    stats.runningApps = appsList.filter((a) => a.status === 'running').length
    stats.totalServers = serversList.length
    stats.recentDeployments = deploymentsList.length
    recentDeployments.value = deploymentsList.slice(0, 10)
  } catch {
    // ignore
  }
})
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-value {
  font-size: 32px;
  font-weight: bold;
  color: #333;
}

.quick-actions {
  display: flex;
  flex-direction: column;
}
</style>
