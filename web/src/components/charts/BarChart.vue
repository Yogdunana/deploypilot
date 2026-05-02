<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart as BarChartSeries } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, BarChartSeries, TitleComponent, TooltipComponent, GridComponent, LegendComponent])

interface Props {
  title?: string
  data: Array<{ name: string; value: number; color?: string }>
  horizontal?: boolean
  height?: string
  dark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  horizontal: false,
  height: '300px',
  dark: true,
})

const option = ref({})

function buildOption() {
  const textColor = props.dark ? '#9ca3af' : '#6b7280'
  const borderColor = props.dark ? '#374151' : '#e5e7eb'

  const categories = props.data.map(d => d.name)
  const values = props.data.map(d => d.value)
  const colors = props.data.map((d, i) => d.color || ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6'][i % 5])

  option.value = {
    title: props.title ? { text: props.title, textStyle: { color: textColor, fontSize: 14 }, left: 0 } : undefined,
    tooltip: { trigger: 'axis', backgroundColor: props.dark ? '#1f2937' : '#fff', borderColor, textStyle: { color: textColor } },
    grid: { left: '3%', right: '4%', bottom: '3%', top: props.title ? 50 : 20, containLabel: true },
    xAxis: props.horizontal
      ? { type: 'value', axisLine: { lineStyle: { color: borderColor } }, axisLabel: { color: textColor } }
      : { type: 'category', data: categories, axisLine: { lineStyle: { color: borderColor } }, axisLabel: { color: textColor, rotate: categories.length > 6 ? 30 : 0 } },
    yAxis: props.horizontal
      ? { type: 'category', data: categories, axisLine: { lineStyle: { color: borderColor } }, axisLabel: { color: textColor } }
      : { type: 'value', axisLine: { lineStyle: { color: borderColor } }, splitLine: { lineStyle: { color: borderColor } }, axisLabel: { color: textColor } },
    series: [{
      type: 'bar',
      data: values.map((v, i) => ({ value: v, itemStyle: { color: colors[i], borderRadius: [4, 4, 0, 0] } })),
      barMaxWidth: 40,
    }],
  }
}

watch(() => [props.data, props.dark], buildOption, { deep: true })
onMounted(buildOption)
</script>

<template>
  <VChart :option="option" :style="{ height }" autoresize class="w-full" />
</template>
