<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import * as serversApi from '@/api/modules/servers'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const { t } = useI18n()
const router = useRouter()
const { toast } = useToast()

const name = ref('')
const host = ref('')
const port = ref(22)
const user = ref('root')
const loading = ref(false)

async function handleSubmit() {
  if (!name.value || !host.value) {
    toast(t('common.required') || '请填写必填项', 'destructive')
    return
  }
  loading.value = true
  try {
    await serversApi.create({ name: name.value, host: host.value, port: port.value, user: user.value } as any)
    toast(t('servers.createSuccess') || '服务器添加成功', 'success')
    router.push('/servers')
  } catch (err: any) {
    toast(err.response?.data?.message || t('servers.createFailed') || '添加失败', 'destructive')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader :title="t('servers.addServer')">
      <template #actions>
        <Button variant="outline" @click="router.push('/servers')">
          {{ t('common.back') || '返回' }}
        </Button>
      </template>
    </PageHeader>

    <div class="max-w-lg">
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Name -->
        <div class="space-y-2">
          <label class="text-sm font-medium">{{ t('servers.createServerName') }}</label>
          <Input v-model="name" placeholder="例如: production-01" />
        </div>
        <!-- Host -->
        <div class="space-y-2">
          <label class="text-sm font-medium">{{ t('servers.createHost') }}</label>
          <Input v-model="host" placeholder="192.168.1.100" />
        </div>
        <!-- Port -->
        <div class="space-y-2">
          <label class="text-sm font-medium">{{ t('servers.createPort') }}</label>
          <Input v-model.number="port" type="number" placeholder="22" />
        </div>
        <!-- User -->
        <div class="space-y-2">
          <label class="text-sm font-medium">{{ t('servers.createSshUser') }}</label>
          <Input v-model="user" placeholder="root" />
        </div>
        <!-- Submit -->
        <Button type="submit" :disabled="loading">
          {{ loading ? '...' : (t('servers.addServer') || '添加服务器') }}
        </Button>
      </form>
    </div>
  </div>
</template>
