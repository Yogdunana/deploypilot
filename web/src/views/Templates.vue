<template>
  <div>
    <div class="page-header">
      <h2 style="margin: 0">Templates</h2>
    </div>

    <el-row :gutter="20" v-loading="loading">
      <el-col :span="8" v-for="tpl in templates" :key="tpl.id || tpl.name" style="margin-bottom: 20px">
        <el-card shadow="hover" class="template-card">
          <template #header>
            <div class="card-header">
              <el-tag :type="tagType(tpl.type)" size="small">{{ tpl.type }}</el-tag>
              <span class="template-name">{{ tpl.name }}</span>
            </div>
          </template>
          <p class="template-desc">{{ tpl.description || 'No description available' }}</p>
          <div class="template-meta">
            <span><strong>Default Port:</strong> {{ tpl.default_port || '-' }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <el-empty v-if="!loading && templates.length === 0" description="No templates available" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { templatesApi } from '../api'

const loading = ref(false)
const templates = ref([])

function tagType(type) {
  const map = {
    nodejs: '',
    python: 'success',
    java: 'warning',
    go: 'info',
    docker: 'danger',
    static: 'info',
  }
  return map[type] || ''
}

async function fetchTemplates() {
  loading.value = true
  try {
    const data = await templatesApi.list()
    templates.value = Array.isArray(data) ? data : data.data || []
  } catch {
    templates.value = []
  } finally {
    loading.value = false
  }
}

onMounted(fetchTemplates)
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.template-card {
  height: 100%;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.template-name {
  font-weight: bold;
  font-size: 16px;
}

.template-desc {
  color: #666;
  margin: 0 0 12px;
  min-height: 40px;
}

.template-meta {
  font-size: 13px;
  color: #999;
}
</style>
