<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
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
import { useI18n } from 'vue-i18n'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

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
const columns = computed(() => [
  { key: 'username', label: t('users.username') },
  { key: 'email', label: t('users.email') },
  { key: 'role', label: t('users.role') },
  { key: 'created_at', label: t('users.createdAt') },
  { key: 'actions', label: t('users.actions'), width: '80px' },
])

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
    toast(err.response?.data?.message || t('users.fetchFailed'), 'destructive')
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
    toast(t('users.roleUpdated', { username: user.username }), 'success')
    fetchUsers()
  } catch (err: any) {
    toast(err.response?.data?.message || t('users.roleUpdateFailed'), 'destructive')
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
    toast(t('users.deleted', { username: deletingItem.value.username }), 'success')
    fetchUsers()
  } catch (err: any) {
    toast(err.response?.data?.message || t('users.deleteFailed'), 'destructive')
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
    { label: t('users.delete'), icon: Trash2, danger: true, action: () => openDeleteDialog(user) },
  ]
}

onMounted(fetchUsers)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('users.title')" />

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
      :title="t('users.noUsers')"
      :description="t('users.noUsersDesc')"
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
      :title="t('users.deleteConfirm')"
      :description="t('users.deleteConfirmDesc', { username: deletingItem?.username || '' })"
      :confirm-text="t('users.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
