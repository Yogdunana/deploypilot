<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Applications</h2>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon>
        Create App
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="apps" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="tech_stack" label="Tech Stack" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.tech_stack || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">
              {{ row.status || 'unknown' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="domain" label="Domain" min-width="150">
          <template #default="{ row }">
            {{ row.domain || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="Created" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="$router.push(`/apps/${row.id}`)">
              Detail
            </el-button>
            <el-button size="small" type="success" @click="handleDeploy(row)">
              Deploy
            </el-button>
            <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, row)">
              <el-button size="small">
                More<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="logs">View Logs</el-dropdown-item>
                  <el-dropdown-item command="backup">Backup</el-dropdown-item>
                  <el-dropdown-item command="restore">Restore</el-dropdown-item>
                  <el-dropdown-item command="rollback">Rollback</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <span style="color: #f56c6c">Delete</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && apps.length === 0" description="No applications found" />
    </el-card>

    <el-dialog v-model="createDialogVisible" title="Create Application" width="500px">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="createForm.name" placeholder="App name" />
        </el-form-item>
        <el-form-item label="Repo URL" prop="repo_url">
          <el-input v-model="createForm.repo_url" placeholder="https://github.com/..." />
        </el-form-item>
        <el-form-item label="Branch" prop="branch">
          <el-input v-model="createForm.branch" placeholder="main" />
        </el-form-item>
        <el-form-item label="Tech Stack" prop="tech_stack">
          <el-select v-model="createForm.tech_stack" placeholder="Select tech stack" style="width: 100%">
            <el-option label="Node.js" value="nodejs" />
            <el-option label="Python" value="python" />
            <el-option label="Java" value="java" />
            <el-option label="Go" value="go" />
            <el-option label="Docker" value="docker" />
            <el-option label="Static" value="static" />
          </el-select>
        </el-form-item>
        <el-form-item label="Domain">
          <el-input v-model="createForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="Server">
          <el-select v-model="createForm.server_id" placeholder="Select server" style="width: 100%">
            <el-option
              v-for="s in servers"
              :key="s.id"
              :label="s.name"
              :value="s.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">Create</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { appsApi, serversApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const apps = ref([])
const servers = ref([])
const createDialogVisible = ref(false)
const creating = ref(false)
const createFormRef = ref(null)

const createForm = ref({
  name: '',
  repo_url: '',
  branch: 'main',
  tech_stack: '',
  domain: '',
  server_id: '',
})

const createRules = {
  name: [{ required: true, message: 'Please enter app name', trigger: 'blur' }],
  repo_url: [{ required: true, message: 'Please enter repo URL', trigger: 'blur' }],
  tech_stack: [{ required: true, message: 'Please select tech stack', trigger: 'change' }],
}

function statusType(status) {
  const map = { running: 'success', stopped: 'info', failed: 'danger' }
  return map[status] || 'info'
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

function openCreateDialog() {
  createForm.value = { name: '', repo_url: '', branch: 'main', tech_stack: '', domain: '', server_id: '' }
  createDialogVisible.value = true
}

async function fetchApps() {
  loading.value = true
  try {
    const data = await appsApi.list()
    apps.value = Array.isArray(data) ? data : data.data || []
  } catch {
    apps.value = []
  } finally {
    loading.value = false
  }
}

async function fetchServers() {
  try {
    const data = await serversApi.list()
    servers.value = Array.isArray(data) ? data : data.data || []
  } catch {
    servers.value = []
  }
}

async function handleCreate() {
  const valid = await createFormRef.value.validate().catch(() => false)
  if (!valid) return

  creating.value = true
  try {
    await appsApi.create(createForm.value)
    ElMessage.success('App created successfully')
    createDialogVisible.value = false
    fetchApps()
  } catch {
    // handled by interceptor
  } finally {
    creating.value = false
  }
}

async function handleDeploy(row) {
  try {
    await appsApi.deploy(row.id, {})
    ElMessage.success('Deployment started')
    fetchApps()
  } catch {
    // handled by interceptor
  }
}

async function handleCommand(cmd, row) {
  switch (cmd) {
    case 'logs':
      router.push(`/apps/${row.id}`)
      break
    case 'backup':
      try {
        await appsApi.backup(row.id)
        ElMessage.success('Backup started')
      } catch { /* handled */ }
      break
    case 'restore':
      try {
        await appsApi.restore(row.id)
        ElMessage.success('Restore started')
      } catch { /* handled */ }
      break
    case 'rollback':
      try {
        await appsApi.rollback(row.id)
        ElMessage.success('Rollback started')
      } catch { /* handled */ }
      break
    case 'delete':
      try {
        await ElMessageBox.confirm(`Delete app "${row.name}"?`, 'Confirm', { type: 'warning' })
        await appsApi.delete(row.id)
        ElMessage.success('App deleted')
        fetchApps()
      } catch { /* cancelled or error */ }
      break
  }
}

onMounted(() => {
  fetchApps()
  fetchServers()
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
