<template>
  <div v-loading="loading">
    <div class="page-header">
      <h2 style="margin: 0">App Detail</h2>
      <el-button @click="$router.push('/apps')">Back to Apps</el-button>
    </div>

    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>App Information</span>
              <el-tag :type="statusType(app.status)" size="large">
                {{ app.status || 'unknown' }}
              </el-tag>
            </div>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="Name">{{ app.name }}</el-descriptions-item>
            <el-descriptions-item label="Tech Stack">
              <el-tag size="small">{{ app.tech_stack || '-' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Repository">
              <a v-if="app.repo_url" :href="app.repo_url" target="_blank" style="color: #409eff">
                {{ app.repo_url }}
              </a>
              <span v-else>-</span>
            </el-descriptions-item>
            <el-descriptions-item label="Branch">{{ app.branch || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Domain">{{ app.domain || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Server">{{ app.server_name || app.server_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Created">{{ formatDate(app.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="Updated">{{ formatDate(app.updated_at) }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>Actions</template>
          <div class="action-buttons">
            <el-button type="primary" :loading="deploying" @click="handleDeploy" style="width: 100%; margin-bottom: 8px">
              <el-icon><Upload /></el-icon> Deploy
            </el-button>
            <el-button type="success" :loading="building" @click="handleBuild" style="width: 100%; margin-bottom: 8px">
              <el-icon><SetUp /></el-icon> Build & Deploy
            </el-button>
            <el-button type="warning" :loading="rollingBack" @click="handleRollback" style="width: 100%; margin-bottom: 8px">
              <el-icon><RefreshLeft /></el-icon> Rollback
            </el-button>
            <el-button :loading="backingUp" @click="handleBackup" style="width: 100%; margin-bottom: 8px">
              <el-icon><Download /></el-icon> Backup
            </el-button>
            <el-button type="info" :loading="restoring" @click="handleRestore" style="width: 100%">
              <el-icon><RefreshRight /></el-icon> Restore
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>Container Logs</span>
          <div>
            <el-switch v-model="autoRefresh" active-text="Auto Refresh" style="margin-right: 12px" />
            <el-button size="small" @click="fetchLogs">
              <el-icon><Refresh /></el-icon> Refresh
            </el-button>
          </div>
        </div>
      </template>
      <el-input
        v-model="logs"
        type="textarea"
        :rows="15"
        readonly
        placeholder="Loading logs..."
        style="font-family: monospace; font-size: 13px"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { appsApi } from '../api'
import { ElMessage } from 'element-plus'

const route = useRoute()
const appId = route.params.id

const loading = ref(true)
const app = ref({})
const logs = ref('')
const autoRefresh = ref(false)
const deploying = ref(false)
const building = ref(false)
const rollingBack = ref(false)
const backingUp = ref(false)
const restoring = ref(false)

let refreshTimer = null

function statusType(status) {
  const map = { running: 'success', stopped: 'info', failed: 'danger' }
  return map[status] || 'info'
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

async function fetchApp() {
  loading.value = true
  try {
    const data = await appsApi.get(appId)
    app.value = data.data || data
  } catch {
    // handled by interceptor
  } finally {
    loading.value = false
  }
}

async function fetchLogs() {
  try {
    const data = await appsApi.logs(appId)
    logs.value = typeof data === 'string' ? data : data.logs || data.data || JSON.stringify(data, null, 2)
  } catch {
    logs.value = 'Failed to load logs'
  }
}

async function handleDeploy() {
  deploying.value = true
  try {
    await appsApi.deploy(appId, {})
    ElMessage.success('Deployment started')
    fetchApp()
  } catch { /* handled */ } finally {
    deploying.value = false
  }
}

async function handleBuild() {
  building.value = true
  try {
    await appsApi.build(appId, {})
    ElMessage.success('Build & deploy started')
    fetchApp()
  } catch { /* handled */ } finally {
    building.value = false
  }
}

async function handleRollback() {
  rollingBack.value = true
  try {
    await appsApi.rollback(appId)
    ElMessage.success('Rollback started')
    fetchApp()
  } catch { /* handled */ } finally {
    rollingBack.value = false
  }
}

async function handleBackup() {
  backingUp.value = true
  try {
    await appsApi.backup(appId)
    ElMessage.success('Backup started')
  } catch { /* handled */ } finally {
    backingUp.value = false
  }
}

async function handleRestore() {
  restoring.value = true
  try {
    await appsApi.restore(appId)
    ElMessage.success('Restore started')
    fetchApp()
  } catch { /* handled */ } finally {
    restoring.value = false
  }
}

watch(autoRefresh, (val) => {
  if (val) {
    refreshTimer = setInterval(fetchLogs, 3000)
  } else {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})

onMounted(() => {
  fetchApp()
  fetchLogs()
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.action-buttons {
  display: flex;
  flex-direction: column;
}
</style>
