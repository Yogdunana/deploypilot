<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Plus, Trash2, Eye, EyeOff, Save } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as appsApi from '@/api/modules/apps'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { t } = useI18n()
const { toast } = inject<any>('toast')!

const appName = ref('')
const loading = ref(true)
const saving = ref(false)

// 环境变量列表
interface EnvItem {
  key: string
  value: string
  visible: boolean
  isNew?: boolean
}

const envList = ref<EnvItem[]>([])

// 加载环境变量
async function fetchEnv() {
  loading.value = true
  try {
    const res = await appsApi.getEnv(Number(props.id))
    if (res.data.status === 'success') {
      const envData = res.data.data || {}
      envList.value = Object.entries(envData).map(([key, value]) => ({
        key,
        value: value as string,
        visible: false,
      }))
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('appEnv.loadFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// 添加变量
function addVariable() {
  envList.value.push({
    key: '',
    value: '',
    visible: true,
    isNew: true,
  })
}

// 删除变量
function removeVariable(index: number) {
  envList.value.splice(index, 1)
}

// 切换可见性
function toggleVisibility(index: number) {
  envList.value[index].visible = !envList.value[index].visible
}

// 保存
async function saveEnv() {
  // 验证
  const emptyKeys = envList.value.filter((item) => !item.key.trim())
  if (emptyKeys.length > 0) {
    toast(t('appEnv.emptyVarName'), 'destructive')
    return
  }

  // 检查重复 key
  const keys = envList.value.map((item) => item.key.trim())
  const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index)
  if (duplicates.length > 0) {
    toast(t('appEnv.duplicateVarName', { names: duplicates.join(', ') }), 'destructive')
    return
  }

  saving.value = true
  try {
    const envObject: Record<string, string> = {}
    envList.value.forEach((item) => {
      if (item.key.trim()) {
        envObject[item.key.trim()] = item.value
      }
    })

    await appsApi.updateEnv(Number(props.id), { env_vars: JSON.stringify(envObject) })
    toast(t('appEnv.saved'), 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || t('appEnv.saveFailed'), 'destructive')
  } finally {
    saving.value = false
  }
}

// 获取应用信息
async function fetchApp() {
  try {
    const res = await appsApi.get(Number(props.id))
    if (res.data.status === 'success') {
      appName.value = res.data.data.name
    }
  } catch {
    // 静默处理
  }
}

onMounted(() => {
  fetchApp()
  fetchEnv()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader>
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push('/apps')">
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <div>
            <h1 class="text-xl font-semibold text-foreground">
              {{ t('appEnv.title', { name: appName }) }}
            </h1>
            <p class="mt-0.5 text-sm text-muted-foreground">
              {{ t('appEnv.totalVars', { count: envList.length }) }}
            </p>
          </div>
        </div>
      </template>
      <template #actions>
        <Button :loading="saving" @click="saveEnv">
          <template #icon><Save class="w-4 h-4" /></template>
          {{ t('appEnv.saved') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 加载状态 -->
    <div v-if="loading" class="space-y-3">
      <Skeleton v-for="i in 5" :key="i" class="h-10 w-full" />
    </div>

    <!-- 环境变量编辑器 -->
    <div v-else class="space-y-2">
      <!-- 表头 -->
      <div class="grid grid-cols-[1fr_1fr_80px] gap-3 px-1">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('appEnv.key') }}</span>
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('appEnv.value') }}</span>
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider text-center">{{ t('appEnv.actions') }}</span>
      </div>

      <!-- 变量行 -->
      <div
        v-for="(item, index) in envList"
        :key="index"
        class="grid grid-cols-[1fr_1fr_80px] gap-3 items-center group"
      >
        <Input
          v-model="item.key"
          :placeholder="t('appEnv.keyPlaceholder')"
          :class="item.isNew ? 'border-primary/50' : ''"
        />
        <div class="relative">
          <Input
            v-model="item.value"
            :type="item.visible ? 'text' : 'password'"
            :placeholder="t('appEnv.valuePlaceholder')"
          />
          <button
            class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            @click="toggleVisibility(index)"
          >
            <Eye v-if="!item.visible" class="w-4 h-4" />
            <EyeOff v-else class="w-4 h-4" />
          </button>
        </div>
        <div class="flex justify-center">
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:text-destructive"
            @click="removeVariable(index)"
          >
            <Trash2 class="w-4 h-4" />
          </Button>
        </div>
      </div>

      <!-- 添加按钮 -->
      <Button variant="outline" size="sm" class="mt-2" @click="addVariable">
        <template #icon><Plus class="w-4 h-4" /></template>
        {{ t('appEnv.addVar') }}
      </Button>
    </div>
  </div>
</template>
