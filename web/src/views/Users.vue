<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Users</h2>
    </div>

    <el-card shadow="hover">
      <el-table :data="users" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="username" label="Username" min-width="150" />
        <el-table-column prop="role" label="Role" width="200">
          <template #default="{ row }">
            <el-select
              :model-value="row.role"
              @change="(val) => handleUpdateRole(row, val)"
              size="small"
              style="width: 150px"
            >
              <el-option
                v-for="r in roles"
                :key="r"
                :label="r"
                :value="r"
              />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="Created" width="200">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="handleDelete(row)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && users.length === 0" description="No users found" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { usersApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const users = ref([])
const roles = ref(['admin', 'user'])

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

async function fetchUsers() {
  loading.value = true
  try {
    const data = await usersApi.list()
    users.value = Array.isArray(data) ? data : data.data || []
  } catch {
    users.value = []
  } finally {
    loading.value = false
  }
}

async function fetchRoles() {
  try {
    const data = await usersApi.getRoles()
    const list = Array.isArray(data) ? data : data.data || []
    if (list.length > 0) roles.value = list
  } catch {
    // use defaults
  }
}

async function handleUpdateRole(row, role) {
  try {
    await usersApi.updateRole(row.id, role)
    ElMessage.success(`Role updated to "${role}"`)
    fetchUsers()
  } catch { /* handled */ }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete user "${row.username}"?`, 'Confirm', { type: 'warning' })
    await usersApi.delete(row.id)
    ElMessage.success('User deleted')
    fetchUsers()
  } catch { /* cancelled or error */ }
}

onMounted(() => {
  fetchUsers()
  fetchRoles()
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
