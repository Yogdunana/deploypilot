<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Notifications</h2>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> Add Notification
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="notifications" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="channel" label="Channel" width="150">
          <template #default="{ row }">
            <el-tag size="small">{{ row.channel }}</el-tag>
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
      <el-empty v-if="!loading && notifications.length === 0" description="No notifications configured" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEditing ? 'Edit Notification' : 'Add Notification'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Notification name" />
        </el-form-item>
        <el-form-item label="Channel" prop="channel">
          <el-select v-model="form.channel" placeholder="Select channel" style="width: 100%">
            <el-option label="Email" value="email" />
            <el-option label="Slack" value="slack" />
            <el-option label="Webhook" value="webhook" />
            <el-option label="DingTalk" value="dingtalk" />
          </el-select>
        </el-form-item>
        <el-form-item label="URL / Email" prop="url">
          <el-input v-model="form.url" placeholder="Webhook URL or email address" />
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
import { notificationsApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const notifications = ref([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const editingId = ref(null)
const formRef = ref(null)

const form = ref({ name: '', channel: '', url: '' })

const rules = {
  name: [{ required: true, message: 'Please enter name', trigger: 'blur' }],
  channel: [{ required: true, message: 'Please select channel', trigger: 'change' }],
  url: [{ required: true, message: 'Please enter URL or email', trigger: 'blur' }],
}

function openCreateDialog() {
  isEditing.value = false
  editingId.value = null
  form.value = { name: '', channel: '', url: '' }
  dialogVisible.value = true
}

function openEditDialog(row) {
  isEditing.value = true
  editingId.value = row.id
  form.value = { name: row.name, channel: row.channel, url: row.url || '' }
  dialogVisible.value = true
}

async function fetchNotifications() {
  loading.value = true
  try {
    const data = await notificationsApi.list()
    notifications.value = Array.isArray(data) ? data : data.data || []
  } catch {
    notifications.value = []
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
      await notificationsApi.update(editingId.value, form.value)
      ElMessage.success('Notification updated')
    } else {
      await notificationsApi.create(form.value)
      ElMessage.success('Notification created')
    }
    dialogVisible.value = false
    fetchNotifications()
  } catch { /* handled */ } finally {
    saving.value = false
  }
}

async function handleToggle(row, val) {
  try {
    await notificationsApi.update(row.id, { ...row, enabled: val })
    ElMessage.success(val ? 'Notification enabled' : 'Notification disabled')
    fetchNotifications()
  } catch { /* handled */ }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`Delete notification "${row.name}"?`, 'Confirm', { type: 'warning' })
    await notificationsApi.delete(row.id)
    ElMessage.success('Notification deleted')
    fetchNotifications()
  } catch { /* cancelled or error */ }
}

onMounted(fetchNotifications)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
