<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Servers</h2>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> Add Server
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="servers" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="host" label="Host" min-width="200" />
        <el-table-column prop="port" label="Port" width="100" />
        <el-table-column prop="user" label="User" width="120" />
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="row.status === 'connected' ? 'success' : 'info'" size="small">
              {{ row.status || 'unknown' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="success" :loading="testing === row.id" @click="handleTest(row)">
              Test
            </el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">
              Delete
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && servers.length === 0" description="No servers found" />
    </el-card>

    <el-dialog v-model="createDialogVisible" title="Add Server" width="500px">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="createForm.name" placeholder="Server name" />
        </el-form-item>
        <el-form-item label="Host" prop="host">
          <el-input v-model="createForm.host" placeholder="192.168.1.1 or example.com" />
        </el-form-item>
        <el-form-item label="Port" prop="port">
          <el-input-number v-model="createForm.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="User" prop="user">
          <el-input v-model="createForm.user" placeholder="root" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">Add</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { serversApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const servers = ref([])
const createDialogVisible = ref(false)
const creating = ref(false)
const testing = ref(null)
const createFormRef = ref(null)

const createForm = ref({
  name: '',
  host: '',
  port: 22,
  user: 'root',
})

const createRules = {
  name: [{ required: true, message: 'Please enter server name', trigger: 'blur' }],
  host: [{ required: true, message: 'Please enter host', trigger: 'blur' }],
  port: [{ required: true, message: 'Please enter port', trigger: 'blur' }],
}

function openCreateDialog() {
  createForm.value = { name: '', host: '', port: 22, user: 'root' }
  createDialogVisible.value = true
}

async function fetchServers() {
  loading.value = true
  try {
    const data = await serversApi.list()
    servers.value = Array.isArray(data) ? data : data.data || []
  } catch {
    servers.value = []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  const valid = await createFormRef.value.validate().catch(() => false)
  if (!valid) return

  creating.value = true
  try {
    await serversApi.create(createForm.value)
    ElMessage.success('Server added successfully')
    createDialogVisible.value = false
    fetchServers()
  } catch { /* handled */ } finally {
    creating.value = false
  }
}

async function handleTest(row) {
  testing.value = row.id
  try {
    await serversApi.test(row.id)
    ElMessage.success('Connection successful')
  } catch { /* handled */ } finally {
    testing.value = null
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete server "${row.name}"?`, 'Confirm', { type: 'warning' })
    await serversApi.delete(row.id)
    ElMessage.success('Server deleted')
    fetchServers()
  } catch { /* cancelled or error */ }
}

onMounted(fetchServers)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
