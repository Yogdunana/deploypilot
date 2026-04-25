<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { MoreHorizontal, Trash2, Users as UsersIcon, Shield } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Table from '@/components/ui/Table.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as usersApi from '@/api/modules/users'
import type { User, Role } from '@/types/models'

const { toast } = inject<any>('toast')!

// State
const users = ref<User[]>([])
const roles = ref<Role[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<User | null>(null)
const deleting = ref(false)

// Table columns
const columns = [
  { key: 'username', label: '用户名' },
  { key: 'email', label: '邮箱' },
  { key: 'role', label: '角色' },
  { key: 'created_at', label: '创建时间' },
  { key: 'actions', label: '操作', width: '80px' },
]

// Role badge mapping
function getRoleBadge(role: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive'; label: string }> = {
    owner: { variant: 'default', label: 'Owner' },
    admin: { variant: 'destructive', label: 'Admin' },
    dev: { variant: 'success', label: 'Dev' },
    viewer: { variant: 'secondary', label: 'Viewer' },
  }
  return map[role] || { variant: 'secondary' as const, label: role }
}

// Fetch users
async function fetchUsers() {
  loading.value = true
  try {
    const [usersRes, rolesRes] = await Promise.all([
      usersApi.list({ page: page.value, page_size: pageSize.value }),
      usersApi.getRoles(),
    ])
    if (usersRes.data.status === 'success') {
      users.value = usersRes.data.data
      total.value = usersRes.data.pagination?.total || 0
    }
    if (rolesRes.data.status === 'success') {
      roles.value = rolesRes.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取用户列表失败', 'destructive')
  } finally {
    loading.value = false
  }
}

// Get user role name
function getUserRoleName(user: User): string {
  if (user.role?.name) return user.role.name.toLowerCase()
  return 'viewer'
}

// Change role
async function handleChangeRole(user: User, roleId: number) {
  try {
    await usersApi.updateRole(user.id, roleId)
    toast(`用户「${user.username}」角色已更新`, 'success')
    fetchUsers()
  } catch (err: any) {
    toast(err.response?.data?.message || '更新角色失败', 'destructive')
  }
}

// Delete
function openDeleteDialog(item: User) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await usersApi.deleteUser(deletingItem.value.id)
    toast(`用户「${deletingItem.value.username}」已删除`, 'success')
    fetchUsers()
  } catch (err: any) {
    toast(err.response?.data?.message || '删除失败', 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

function getDropdownItems(user: User) {
  const roleItems = roles.value.map((role) => ({
    label: role.name,
    action: () => handleChangeRole(user, role.id),
  }))
  return [
    ...roleItems,
    { label: '删除', icon: Trash2, danger: true, action: () => openDeleteDialog(user) },
  ]
}

onMounted(fetchUsers)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader title="用户管理" />

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-36" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="users.length > 0"
      :columns="columns"
      :data="users"
    >
      <template #cell-username="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.username }}</span>
      </template>
      <template #cell-email="{ row }">
        <span class="text-sm text-muted-foreground">{{ row.email }}</span>
      </template>
      <template #cell-role="{ row }">
        <Badge :variant="getRoleBadge(getUserRoleName(row as User)).variant">
          {{ getRoleBadge(getUserRoleName(row as User)).label }}
        </Badge>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getDropdownItems(row as User)">
          <template #trigger>
            <Button variant="ghost" size="icon">
              <MoreHorizontal class="w-4 h-4" />
            </Button>
          </template>
        </DropdownMenu>
      </template>
    </Table>

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="UsersIcon"
      title="暂无用户"
      description="暂无其他用户数据"
    />

    <!-- Pagination -->
    <div v-if="total > pageSize" class="flex justify-end">
      <Pagination
        v-model:page="page"
        :page-size="pageSize"
        :total="total"
        @update:page="fetchUsers"
      />
    </div>

    <!-- Delete AlertDialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      title="删除用户"
      :description="`确定要删除用户「${deletingItem?.username}」吗？此操作不可撤销。`"
      confirm-text="删除"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
