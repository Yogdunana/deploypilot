<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldCheck, Key, Ban, Users, Server, AppWindow, Package, Check, Lock } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Progress from '@/components/ui/Progress.vue'
import type { LicenseInfo } from '@/types/models'

const { t } = useI18n()

const props = defineProps<{
  licenseInfo: LicenseInfo
}>()

const emit = defineEmits<{
  deactivate: []
}>()

function getStatusVariant(status: string): 'success' | 'destructive' | 'warning' | 'default' | 'secondary' {
  const s = (status || '').toLowerCase()
  switch (s) {
    case 'active':
      return 'success'
    case 'expired':
      return 'destructive'
    case 'revoked':
      return 'destructive'
    default:
      return 'secondary'
  }
}

function getStatusLabel(status: string): string {
  const s = (status || '').toLowerCase()
  const map: Record<string, string> = {
    active: t('license.active'),
    expired: t('license.expired'),
    revoked: t('license.revoked'),
  }
  return map[s] || s
}

function getTierLabel(tier: string): string {
  const t2 = (tier || '').toLowerCase()
  const map: Record<string, string> = {
    community: t('license.community'),
    team: t('license.team'),
    pro: t('license.pro'),
    enterprise: t('license.enterprise'),
  }
  return map[t2] || tier
}

function getUseTypeLabel(useType: string): string {
  const u = (useType || '').toLowerCase()
  const map: Record<string, string> = {
    'non-commercial': t('license.nonCommercial'),
    commercial: t('license.commercial'),
  }
  return map[u] || useType
}

function getDaysUntilExpiry(expiresAt: string | null): number {
  if (!expiresAt) return 0
  const now = new Date()
  const expiry = new Date(expiresAt)
  const diff = expiry.getTime() - now.getTime()
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}

function formatExpiry(expiresAt: string | null): { text: string; urgent: boolean } {
  if (!expiresAt) return { text: t('license.neverExpires'), urgent: false }
  const days = getDaysUntilExpiry(expiresAt)
  if (days < 0) return { text: t('license.expiredAgo', { days: Math.abs(days) }), urgent: true }
  if (days <= 7) return { text: t('license.daysRemaining', { days }), urgent: true }
  return { text: t('license.daysRemaining', { days }), urgent: false }
}

const allFeatures = [
  'basic_deploy', 'ssl', 'dns', 'monitor', 'ci_cd',
  'cluster', 'registry', 'plugin', 'api_key', 'audit',
  'batch_ops', 'oauth2', 'webhook', 'template', 'backup',
]

const expiryInfo = computed(() => formatExpiry(props.licenseInfo.expires_at))
</script>

<template>
  <div class="space-y-4">
    <!-- License status card -->
    <Card>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <ShieldCheck class="w-5 h-5 text-primary" />
            <h3 class="text-sm font-medium text-foreground">{{ t('license.status') }}</h3>
          </div>
          <div class="flex items-center gap-2">
            <Badge :variant="getStatusVariant(licenseInfo.status)">
              {{ getStatusLabel(licenseInfo.status) }}
            </Badge>
            <Badge variant="outline">{{ getTierLabel(licenseInfo.tier) }}</Badge>
            <Badge variant="secondary">{{ getUseTypeLabel(licenseInfo.use_type) }}</Badge>
          </div>
        </div>
      </template>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="space-y-1">
          <p class="text-xs text-muted-foreground">{{ t('license.tier') }}</p>
          <p class="text-sm font-medium text-foreground">{{ getTierLabel(licenseInfo.tier) }}</p>
        </div>
        <div class="space-y-1">
          <p class="text-xs text-muted-foreground">{{ t('license.useType') }}</p>
          <p class="text-sm font-medium text-foreground">{{ getUseTypeLabel(licenseInfo.use_type) }}</p>
        </div>
        <div class="space-y-1">
          <p class="text-xs text-muted-foreground">{{ t('license.expiresAt') }}</p>
          <p
            class="text-sm font-medium"
            :class="expiryInfo.urgent ? 'text-destructive' : 'text-foreground'"
          >
            {{ expiryInfo.text }}
          </p>
        </div>
        <div class="space-y-1">
          <p class="text-xs text-muted-foreground">{{ t('license.machineId') }}</p>
          <p class="text-xs font-mono text-muted-foreground break-all">{{ licenseInfo.machine_id || '-' }}</p>
        </div>
      </div>
      <div class="flex justify-end gap-2 mt-4 pt-4 border-t border-border">
        <Button variant="outline" size="sm" @click="emit('deactivate')">
          <template #icon><Ban class="w-4 h-4" /></template>
          <span class="hidden sm:inline">{{ t('license.deactivate') }}</span>
        </Button>
      </div>
    </Card>

    <!-- Resource usage -->
    <Card>
      <template #header>
        <div class="flex items-center gap-2">
          <Users class="w-5 h-5 text-primary" />
          <h3 class="text-sm font-medium text-foreground">{{ t('license.resourceUsage') }}</h3>
        </div>
      </template>
      <div class="space-y-4">
        <!-- Servers -->
        <div class="space-y-2">
          <div class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <Server class="w-4 h-4 text-muted-foreground" />
              <span class="text-foreground">{{ t('license.servers') }}</span>
            </div>
            <span class="text-muted-foreground">
              {{ licenseInfo.limits?.max_servers === -1 ? t('license.unlimited') : licenseInfo.limits?.max_servers }}
            </span>
          </div>
          <Progress
            v-if="licenseInfo.limits?.max_servers !== -1"
            :value="100"
            variant="default"
          />
        </div>
        <!-- Apps -->
        <div class="space-y-2">
          <div class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <AppWindow class="w-4 h-4 text-muted-foreground" />
              <span class="text-foreground">{{ t('license.apps') }}</span>
            </div>
            <span class="text-muted-foreground">
              {{ licenseInfo.limits?.max_apps === -1 ? t('license.unlimited') : licenseInfo.limits?.max_apps }}
            </span>
          </div>
          <Progress
            v-if="licenseInfo.limits?.max_apps !== -1"
            :value="100"
            variant="default"
          />
        </div>
        <!-- Users -->
        <div class="space-y-2">
          <div class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <Users class="w-4 h-4 text-muted-foreground" />
              <span class="text-foreground">{{ t('license.users') }}</span>
            </div>
            <span class="text-muted-foreground">
              {{ licenseInfo.limits?.max_users === -1 ? t('license.unlimited') : licenseInfo.limits?.max_users }}
            </span>
          </div>
          <Progress
            v-if="licenseInfo.limits?.max_users !== -1"
            :value="100"
            variant="default"
          />
        </div>
      </div>
    </Card>

    <!-- Features list -->
    <Card>
      <template #header>
        <div class="flex items-center gap-2">
          <Package class="w-5 h-5 text-primary" />
          <h3 class="text-sm font-medium text-foreground">{{ t('license.features') }}</h3>
        </div>
      </template>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <div
          v-for="feature in allFeatures"
          :key="feature"
          class="flex items-center gap-2 text-sm"
        >
          <Check v-if="licenseInfo.features?.includes(feature)" class="w-4 h-4 text-success shrink-0" />
          <Lock v-else class="w-4 h-4 text-muted-foreground shrink-0" />
          <span :class="licenseInfo.features?.includes(feature) ? 'text-foreground' : 'text-muted-foreground'">
            {{ feature }}
          </span>
        </div>
      </div>
    </Card>

    <!-- Addons -->
    <Card>
      <template #header>
        <div class="flex items-center gap-2">
          <Package class="w-5 h-5 text-primary" />
          <h3 class="text-sm font-medium text-foreground">{{ t('license.addons') }}</h3>
        </div>
      </template>
      <div v-if="licenseInfo.addons && licenseInfo.addons.length > 0" class="space-y-3">
        <div
          v-for="(addon, index) in licenseInfo.addons"
          :key="index"
          class="flex items-center justify-between py-2 border-b border-border last:border-0"
        >
          <div class="flex items-center gap-2">
            <Badge variant="secondary">{{ addon.key }}</Badge>
            <span class="text-sm text-foreground">x{{ addon.amount }}</span>
          </div>
          <span class="text-xs text-muted-foreground">{{ addon.expires_at || t('license.neverExpires') }}</span>
        </div>
      </div>
      <p v-else class="text-sm text-muted-foreground text-center py-4">{{ t('license.noAddons') }}</p>
    </Card>
  </div>
</template>
