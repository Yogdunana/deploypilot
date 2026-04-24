<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">DNS Records</h2>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> Add Record
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="records" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="domain" label="Domain" min-width="180" />
        <el-table-column prop="type" label="Type" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="value" label="Value" min-width="200" />
        <el-table-column label="Actions" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openEditDialog(row)">Edit</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && records.length === 0" description="No DNS records found" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEditing ? 'Edit DNS Record' : 'Add DNS Record'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Domain" prop="domain">
          <el-input v-model="form.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="Type" prop="type">
          <el-select v-model="form.type" placeholder="Select type" style="width: 100%">
            <el-option label="A" value="A" />
            <el-option label="AAAA" value="AAAA" />
            <el-option label="CNAME" value="CNAME" />
            <el-option label="MX" value="MX" />
            <el-option label="TXT" value="TXT" />
            <el-option label="NS" value="NS" />
          </el-select>
        </el-form-item>
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="subdomain" />
        </el-form-item>
        <el-form-item label="Value" prop="value">
          <el-input v-model="form.value" placeholder="IP address or hostname" />
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
import { dnsApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const records = ref([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const editingId = ref(null)
const formRef = ref(null)

const form = ref({ domain: '', type: '', name: '', value: '' })

const rules = {
  domain: [{ required: true, message: 'Please enter domain', trigger: 'blur' }],
  type: [{ required: true, message: 'Please select type', trigger: 'change' }],
  name: [{ required: true, message: 'Please enter name', trigger: 'blur' }],
  value: [{ required: true, message: 'Please enter value', trigger: 'blur' }],
}

function openCreateDialog() {
  isEditing.value = false
  editingId.value = null
  form.value = { domain: '', type: '', name: '', value: '' }
  dialogVisible.value = true
}

function openEditDialog(row) {
  isEditing.value = true
  editingId.value = row.id
  form.value = { domain: row.domain, type: row.type, name: row.name, value: row.value }
  dialogVisible.value = true
}

async function fetchRecords() {
  loading.value = true
  try {
    const data = await dnsApi.listRecords()
    records.value = Array.isArray(data) ? data : data.data || []
  } catch {
    records.value = []
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
      await dnsApi.updateRecord(editingId.value, form.value)
      ElMessage.success('Record updated')
    } else {
      await dnsApi.createRecord(form.value)
      ElMessage.success('Record created')
    }
    dialogVisible.value = false
    fetchRecords()
  } catch { /* handled */ } finally {
    saving.value = false
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete DNS record for "${row.name}"?`, 'Confirm', { type: 'warning' })
    await dnsApi.deleteRecord(row.id)
    ElMessage.success('Record deleted')
    fetchRecords()
  } catch { /* cancelled or error */ }
}

onMounted(fetchRecords)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
