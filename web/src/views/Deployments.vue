<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Deployments</h2>
      <el-select v-model="statusFilter" placeholder="Filter by status" clearable style="width: 200px" @change="fetchDeployments">
        <el-option label="All" value="" />
        <el-option label="Success" value="success" />
        <el-option label="Running" value="running" />
        <el-option label="Failed" value="failed" />
        <el-option label="Pending" value="pending" />
      </el-select>
    </div>

    <el-card shadow="hover">
      <el-table :data="filteredDeployments" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="app_name" label="App Name" min-width="150" />
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="Time" width="200">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && filteredDeployments.length === 0" description="No deployments found" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { deploymentsApi } from '../api'

const loading = ref(false)
const deployments = ref([])
const statusFilter = ref('')

const filteredDeployments = computed(() => {
  if (!statusFilter.value) return deployments.value
  return deployments.value.filter((d) => d.status === statusFilter.value)
})

function statusType(status) {
  const map = { success: 'success', running: 'warning', failed: 'danger', pending: 'info' }
  return map[status] || 'info'
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

async function fetchDeployments() {
  loading.value = true
  try {
    const data = await deploymentsApi.list()
    deployments.value = Array.isArray(data) ? data : data.data || []
  } catch {
    deployments.value = []
  } finally {
    loading.value = false
  }
}

onMounted(fetchDeployments)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
