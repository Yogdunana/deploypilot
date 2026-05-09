<script setup lang="ts">
import { cn } from '@/lib/utils'
import { Loader2 } from 'lucide-vue-next'

interface TableColumn {
  key: string
  label: string
  width?: string
}

interface Props {
  columns?: TableColumn[]
  data?: Record<string, any>[]
  loading?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  columns: () => [],
  data: () => [],
  loading: false,
})
</script>

<template>
  <div :class="cn('w-full overflow-auto', props.class)">
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
            暂无数据
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
</template>
