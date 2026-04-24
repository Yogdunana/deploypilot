<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">System</h2>
      <el-button type="primary" :loading="checkingUpdate" @click="handleCheckUpdate">
        <el-icon><Refresh /></el-icon> Check for Updates
      </el-button>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card shadow="hover" v-loading="loadingVersion">
          <template #header>
            <div class="card-header">
              <el-icon :size="20" color="#409eff"><InfoFilled /></el-icon>
              <span>Version Information</span>
            </div>
          </template>
          <el-descriptions :column="1" border v-if="versionInfo">
            <el-descriptions-item label="Version">{{ versionInfo.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Build">{{ versionInfo.build || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Go Version">{{ versionInfo.go_version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Commit">{{ versionInfo.commit || '-' }}</el-descriptions-item>
          </el-descriptions>
          <el-empty v-else description="Unable to load version info" :image-size="60" />
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" v-loading="loadingHealth">
          <template #header>
            <div class="card-header">
              <el-icon :size="20" color="#67c23a"><CircleCheck /></el-icon>
              <span>Health Status</span>
            </div>
          </template>
          <el-descriptions :column="1" border v-if="healthInfo">
            <el-descriptions-item label="Status">
              <el-tag :type="healthInfo.status === 'healthy' ? 'success' : 'danger'">
                {{ healthInfo.status || 'unknown' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Uptime">{{ healthInfo.uptime || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Database">
              <el-tag :type="healthInfo.database === 'ok' ? 'success' : 'danger'" size="small">
                {{ healthInfo.database || '-' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <el-empty v-else description="Unable to load health info" :image-size="60" />
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover" style="margin-top: 20px" v-if="updateInfo">
      <template #header>
        <div class="card-header">
          <el-icon :size="20" color="#e6a23c"><Warning /></el-icon>
          <span>Update Information</span>
        </div>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="Latest Version">{{ updateInfo.latest_version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Current Version">{{ updateInfo.current_version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Update Available">
          <el-tag :type="updateInfo.update_available ? 'warning' : 'success'">
            {{ updateInfo.update_available ? 'Yes' : 'No' }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { systemApi } from '../api'
import { ElMessage } from 'element-plus'

const loadingVersion = ref(false)
const loadingHealth = ref(false)
const checkingUpdate = ref(false)
const versionInfo = ref(null)
const healthInfo = ref(null)
const updateInfo = ref(null)

async function fetchVersion() {
  loadingVersion.value = true
  try {
    const data = await systemApi.version()
    versionInfo.value = data.data || data
  } catch {
    versionInfo.value = null
  } finally {
    loadingVersion.value = false
  }
}

async function fetchHealth() {
  loadingHealth.value = true
  try {
    const data = await systemApi.health()
    healthInfo.value = data.data || data
  } catch {
    healthInfo.value = null
  } finally {
    loadingHealth.value = false
  }
}

async function handleCheckUpdate() {
  checkingUpdate.value = true
  try {
    const data = await systemApi.checkUpdate()
    updateInfo.value = data.data || data
    if (updateInfo.value.update_available) {
      ElMessage.warning('A new version is available!')
    } else {
      ElMessage.success('You are on the latest version')
    }
  } catch { /* handled */ } finally {
    checkingUpdate.value = false
  }
}

onMounted(() => {
  fetchVersion()
  fetchHealth()
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
  gap: 8px;
}
</style>
