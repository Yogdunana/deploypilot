<script setup lang="ts">
import { cn } from '@/lib/utils'
import { Loader2 } from 'lucide-vue-next'

export interface ResponsiveColumn {
  key: string
  label: string
  width?: string
  /** If true, this column is shown on mobile card view */
  mobile?: boolean
}

interface Props {
  columns?: ResponsiveColumn[]
  data?: Record<string, any>[]
  loading?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  columns: () => [],
  data: () => [],
  loading: false,
})

/** Columns that should be shown on mobile cards */
const mobileColumns = computed(() =>
  props.columns.filter((col) => col.mobile !== false)
)

/** The first mobile column is used as the card title */
const mobileTitleColumn = computed(() =>
  mobileColumns.value[0]?.key || props.columns[0]?.key
)

/** Remaining mobile columns (excluding the title) */
const mobileDetailColumns = computed(() =>
  mobileColumns.value.slice(1)
)
</script>

<script lang="ts">
import { computed } from 'vue'
</script>

<template>
  <!-- Desktop: Standard table (md+) -->
  <div :class="cn('hidden md:block w-full overflow-auto', props.class)">
    <table class="w-full caption-bottom text-sm">
      <thead class="border-b border-border">
        <tr>
          <th
            v-for="col in columns"
            :key="col.key"
            class="h-10 px-4 text-left align-middle font-medium text-muted-foreground"
            :style="col.width ? { width: col.width } : undefined"
          >
            {{ col.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td :colspan="columns.length" class="h-24 text-center">
            <Loader2 class="w-5 h-5 animate-spin mx-auto text-muted-foreground" />
          </td>
        </tr>
        <tr v-else-if="data.length === 0">
          <td :colspan="columns.length" class="h-24 text-center text-muted-foreground">
            &nbsp;
          </td>
        </tr>
        <template v-else>
          <tr
            v-for="(row, index) in data"
            :key="index"
            class="border-b border-border transition-colors hover:bg-accent/50"
            :class="index % 2 === 1 ? 'bg-card' : 'bg-background'"
          >
            <td
              v-for="col in columns"
              :key="col.key"
              class="h-10 px-4 align-middle text-foreground"
            >
              <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                {{ row[col.key] }}
              </slot>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </div>

  <!-- Mobile: Card layout (<md) -->
  <div class="md:hidden">
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <Loader2 class="w-5 h-5 animate-spin text-muted-foreground" />
    </div>

    <!-- Empty state -->
    <div v-else-if="data.length === 0" class="py-12 text-center text-muted-foreground">
      &nbsp;
    </div>

    <!-- Card list -->
    <div v-else class="space-y-3">
      <div
        v-for="(row, index) in data"
        :key="index"
        class="rounded-lg border border-border bg-card p-4"
      >
        <!-- Card title row -->
        <div class="flex items-center justify-between mb-3">
          <div class="min-w-0 flex-1">
            <slot :name="`cell-${mobileTitleColumn}`" :row="row" :value="row[mobileTitleColumn]">
              <span class="text-sm font-medium text-foreground truncate block">
                {{ row[mobileTitleColumn] }}
              </span>
            </slot>
          </div>
          <!-- Actions column on the right -->
          <div v-if="columns.find(c => c.key === 'actions')" class="ml-2 shrink-0">
            <slot name="cell-actions" :row="row" :value="row['actions']">
              {{ row['actions'] }}
            </slot>
          </div>
        </div>

        <!-- Detail fields -->
        <div class="space-y-2">
          <div
            v-for="col in mobileDetailColumns"
            :key="col.key"
            class="flex items-center justify-between"
          >
            <span class="text-xs text-muted-foreground">{{ col.label }}</span>
            <div class="text-sm text-foreground text-right min-w-0 ml-4">
              <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                <span class="truncate block">{{ row[col.key] }}</span>
              </slot>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
