<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart as LineChartSeries } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
} from 'echarts/components'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChartSeries,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
])

interface Props {
  title?: string
  data: Array<{ name: string; data: Array<[string, number]> }>
  smooth?: boolean
  area?: boolean
  height?: string
  dark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  smooth: true,
  area: false,
  height: '300px',
  dark: true,
})

const option = ref({})

function buildOption() {
  const textColor = props.dark ? '#9ca3af' : '#6b7280'
  const borderColor = props.dark ? '#374151' : '#e5e7eb'

  option.value = {
    title: props.title ? { text: props.title, textStyle: { color: textColor, fontSize: 14 }, left: 0 } : undefined,
    tooltip: { trigger: 'axis', backgroundColor: props.dark ? '#1f2937' : '#fff', borderColor, textStyle: { color: textColor } },
    legend: { textStyle: { color: textColor }, top: props.title ? 30 : 0 },
    grid: { left: '3%', right: '4%', bottom: '14%', top: props.title ? 60 : 40, containLabel: true },
    dataZoom: [{ type: 'inside' }, { type: 'slider', borderColor, textStyle: { color: textColor } }],
    xAxis: { type: 'time', axisLine: { lineStyle: { color: borderColor } }, axisLabel: { color: textColor } },
    yAxis: { type: 'value', axisLine: { lineStyle: { color: borderColor } }, splitLine: { lineStyle: { color: borderColor } }, axisLabel: { color: textColor } },
    series: props.data.map((s, i) => ({
      name: s.name,
      type: 'line',
      smooth: props.smooth,
      data: s.data,
      areaStyle: props.area ? { opacity: 0.15 } : undefined,
      itemStyle: { color: ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6'][i % 5] },
    })),
  }
}

watch(() => [props.data, props.dark, props.title], buildOption, { deep: true })
onMounted(buildOption)
</script>

<template>
  <VChart :option="option" :style="{ height }" autoresize class="w-full" />
</template>
