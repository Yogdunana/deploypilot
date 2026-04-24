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
        <el-table-column label="Actions" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="success" :loading="testing === row.id" @click="handleTest(row)">
              Test
            </el-button>
            <el-button size="small" type="primary" @click="openTerminal(row)">
              Terminal
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

    <el-dialog v-model="terminalDialogVisible" :title="`Terminal - ${terminalServerName}`" width="800px" @close="closeTerminal">
      <div class="terminal-container">
        <div class="terminal-header">
          <el-tag :type="terminalConnected ? 'success' : 'danger'" size="small">
            {{ terminalConnected ? 'Connected' : 'Disconnected' }}
          </el-tag>
        </div>
        <div ref="terminalOutput" class="terminal-output" v-html="terminalHtml"></div>
        <div class="terminal-input">
          <el-input
            v-model="terminalInput"
            placeholder="Enter command..."
            @keyup.enter="sendTerminalCommand"
            :disabled="!terminalConnected"
          >
            <template #append>
              <el-button @click="sendTerminalCommand" :disabled="!terminalConnected">Send</el-button>
            </template>
          </el-input>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, nextTick, onUnmounted } from 'vue'
import { serversApi } from '../api'
import { getToken } from '../utils/auth'
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

// Terminal state
const terminalDialogVisible = ref(false)
const terminalServerName = ref('')
const terminalConnected = ref(false)
const terminalOutput = ref(null)
const terminalHtml = ref('')
const terminalInput = ref('')
let terminalWs = null

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

function openTerminal(row) {
  terminalServerName.value = row.name
  terminalHtml.value = ''
  terminalInput.value = ''
  terminalDialogVisible.value = true

  nextTick(() => {
    connectTerminal(row.id)
  })
}

function connectTerminal(serverId) {
  const token = getToken()
  if (!token) {
    ElMessage.error('Not authenticated')
    return
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws/terminal/${serverId}?token=${token}`

  terminalWs = new WebSocket(wsUrl)

  terminalWs.onopen = () => {
    terminalConnected.value = true
    appendTerminalOutput('<span style="color: #67c23a">Connected to ' + terminalServerName.value + '</span>\n')
  }

  terminalWs.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'output') {
        appendTerminalOutput(escapeHtml(String(msg.data)))
      } else if (msg.type === 'error') {
        appendTerminalOutput('<span style="color: #f56c6c">Error: ' + escapeHtml(String(msg.data)) + '</span>\n')
      }
    } catch (e) {
      appendTerminalOutput(String(event.data))
    }
  }

  terminalWs.onclose = () => {
    terminalConnected.value = false
    appendTerminalOutput('<span style="color: #909399">Connection closed</span>\n')
  }

  terminalWs.onerror = (err) => {
    console.error('Terminal WebSocket error:', err)
  }
}

function sendTerminalCommand() {
  const cmd = terminalInput.value.trim()
  if (!cmd || !terminalWs || terminalWs.readyState !== WebSocket.OPEN) return

  appendTerminalOutput('<span style="color: #409eff">$ ' + escapeHtml(cmd) + '</span>\n')
  terminalWs.send(JSON.stringify({
    type: 'input',
    data: cmd,
    timestamp: new Date().toISOString(),
  }))
  terminalInput.value = ''
}

function appendTerminalOutput(text) {
  terminalHtml.value += text
  nextTick(() => {
    if (terminalOutput.value) {
      terminalOutput.value.scrollTop = terminalOutput.value.scrollHeight
    }
  })
}

function escapeHtml(text) {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

function closeTerminal() {
  if (terminalWs) {
    terminalWs.onclose = null
    terminalWs.close()
    terminalWs = null
  }
  terminalConnected.value = false
}

onUnmounted(() => {
  closeTerminal()
})

fetchServers()
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.terminal-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.terminal-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal-output {
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  padding: 12px;
  border-radius: 4px;
  height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
}

.terminal-input {
  display: flex;
}
</style>
