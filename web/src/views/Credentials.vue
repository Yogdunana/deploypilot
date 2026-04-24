<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Credentials</h2>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> Add Credential
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="credentials" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="type" label="Type" width="150">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="value" label="Value" min-width="200">
          <template #default="{ row }">
            {{ maskValue(row.value) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="Created" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openEditDialog(row)">Edit</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && credentials.length === 0" description="No credentials found" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEditing ? 'Edit Credential' : 'Add Credential'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Credential name" />
        </el-form-item>
        <el-form-item label="Type" prop="type">
          <el-select v-model="form.type" placeholder="Select type" style="width: 100%">
            <el-option label="SSH Key" value="ssh_key" />
            <el-option label="API Token" value="api_token" />
            <el-option label="Password" value="password" />
            <el-option label="Certificate" value="certificate" />
            <el-option label="Other" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="Value" prop="value">
          <el-input v-model="form.value" type="textarea" :rows="3" placeholder="Credential value" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ isEditing ? 'Update' : 'Create' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { credentialsApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const credentials = ref([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const editingId = ref(null)
const formRef = ref(null)

const form = ref({ name: '', type: '', value: '' })

const rules = {
  name: [{ required: true, message: 'Please enter name', trigger: 'blur' }],
  type: [{ required: true, message: 'Please select type', trigger: 'change' }],
  value: [{ required: true, message: 'Please enter value', trigger: 'blur' }],
}

function maskValue(val) {
  if (!val) return '-'
  if (val.length <= 8) return '****'
  return val.substring(0, 4) + '****' + val.substring(val.length - 4)
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

function openCreateDialog() {
  isEditing.value = false
  editingId.value = null
  form.value = { name: '', type: '', value: '' }
  dialogVisible.value = true
}

function openEditDialog(row) {
  isEditing.value = true
  editingId.value = row.id
  form.value = { name: row.name, type: row.type, value: '' }
  dialogVisible.value = true
}

async function fetchCredentials() {
  loading.value = true
  try {
    const data = await credentialsApi.list()
    credentials.value = Array.isArray(data) ? data : data.data || []
  } catch {
    credentials.value = []
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    if (isEditing.value) {
      await credentialsApi.update(editingId.value, form.value)
      ElMessage.success('Credential updated')
    } else {
      await credentialsApi.create(form.value)
      ElMessage.success('Credential created')
    }
    dialogVisible.value = false
    fetchCredentials()
  } catch { /* handled */ } finally {
    saving.value = false
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete credential "${row.name}"?`, 'Confirm', { type: 'warning' })
    await credentialsApi.delete(row.id)
    ElMessage.success('Credential deleted')
    fetchCredentials()
  } catch { /* cancelled or error */ }
}

onMounted(fetchCredentials)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
