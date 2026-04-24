<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Providers</h2>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> Add Provider
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="providers" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="type" label="Type" width="150">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="Enabled" width="100">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              @change="(val) => handleToggle(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openEditDialog(row)">Edit</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && providers.length === 0" description="No providers found" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEditing ? 'Edit Provider' : 'Add Provider'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Provider name" />
        </el-form-item>
        <el-form-item label="Type" prop="type">
          <el-select v-model="form.type" placeholder="Select type" style="width: 100%">
            <el-option label="Docker" value="docker" />
            <el-option label="Kubernetes" value="kubernetes" />
            <el-option label="AWS" value="aws" />
            <el-option label="GCP" value="gcp" />
            <el-option label="Azure" value="azure" />
            <el-option label="Other" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="Config" prop="config">
          <el-input
            v-model="form.configStr"
            type="textarea"
            :rows="6"
            placeholder='{"key": "value"}'
          />
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
import { providersApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const providers = ref([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const editingId = ref(null)
const formRef = ref(null)

const form = ref({ name: '', type: '', configStr: '' })

const rules = {
  name: [{ required: true, message: 'Please enter name', trigger: 'blur' }],
  type: [{ required: true, message: 'Please select type', trigger: 'change' }],
}

function openCreateDialog() {
  isEditing.value = false
  editingId.value = null
  form.value = { name: '', type: '', configStr: '' }
  dialogVisible.value = true
}

function openEditDialog(row) {
  isEditing.value = true
  editingId.value = row.id
  form.value = {
    name: row.name,
    type: row.type,
    configStr: row.config ? JSON.stringify(row.config, null, 2) : '',
  }
  dialogVisible.value = true
}

async function fetchProviders() {
  loading.value = true
  try {
    const data = await providersApi.list()
    providers.value = Array.isArray(data) ? data : data.data || []
  } catch {
    providers.value = []
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  let config = {}
  if (form.value.configStr) {
    try {
      config = JSON.parse(form.value.configStr)
    } catch {
      ElMessage.error('Invalid JSON config')
      return
    }
  }

  saving.value = true
  try {
    const payload = { name: form.value.name, type: form.value.type, config }
    if (isEditing.value) {
      await providersApi.update(editingId.value, payload)
      ElMessage.success('Provider updated')
    } else {
      await providersApi.create(payload)
      ElMessage.success('Provider created')
    }
    dialogVisible.value = false
    fetchProviders()
  } catch { /* handled */ } finally {
    saving.value = false
  }
}

async function handleToggle(row, val) {
  try {
    await providersApi.update(row.id, { ...row, enabled: val })
    ElMessage.success(val ? 'Provider enabled' : 'Provider disabled')
    fetchProviders()
  } catch { /* handled */ }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete provider "${row.name}"?`, 'Confirm', { type: 'warning' })
    await providersApi.delete(row.id)
    ElMessage.success('Provider deleted')
    fetchProviders()
  } catch { /* cancelled or error */ }
}

onMounted(fetchProviders)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
