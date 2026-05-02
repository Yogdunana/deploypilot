<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart as PieChartSeries } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, PieChartSeries, TitleComponent, TooltipComponent, LegendComponent])

interface Props {
  title?: string
  data: Array<{ name: string; value: number; color?: string }>
  height?: string
  dark?: boolean
  donut?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  height: '300px',
  dark: true,
  donut: true,
})

const option = ref({})

function buildOption() {
  const textColor = props.dark ? '#9ca3af' : '#6b7280'
  const colors = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4']

  option.value = {
    title: props.title ? { text: props.title, textStyle: { color: textColor, fontSize: 14 }, left: 0 } : undefined,
    tooltip: { trigger: 'item', backgroundColor: props.dark ? '#1f2937' : '#fff', borderColor: props.dark ? '#374151' : '#e5e7eb', textStyle: { color: textColor } },
    legend: { orient: 'vertical', right: 10, top: 'center', textStyle: { color: textColor } },
    series: [{
      type: 'pie',
      radius: props.donut ? ['45%', '70%'] : '70%',
      center: ['40%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 6, borderColor: props.dark ? '#111827' : '#fff', borderWidth: 2 },
      label: { show: !props.donut, color: textColor },
      data: props.data.map((d, i) => ({
        name: d.name,
        value: d.value,
        itemStyle: { color: d.color || colors[i % colors.length] },
      })),
    }],
  }
}

watch(() => [props.data, props.dark], buildOption, { deep: true })
onMounted(buildOption)
</script>

<template>
  <VChart :option="option" :style="{ height }" autoresize class="w-full" />
</template>
