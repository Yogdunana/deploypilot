<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { GaugeChart as GaugeChartSeries } from 'echarts/charts'
import { TitleComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, GaugeChartSeries, TitleComponent, TooltipComponent])

interface Props {
  title?: string
  value: number
  min?: number
  max?: number
  unit?: string
  height?: string
  dark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  min: 0,
  max: 100,
  unit: '%',
  height: '200px',
  dark: true,
})

const option = ref({})

function buildOption() {
  const textColor = props.dark ? '#9ca3af' : '#6b7280'
  const color = props.value >= 95 ? '#10b981' : props.value >= 80 ? '#f59e0b' : '#ef4444'

  option.value = {
    title: props.title ? { text: props.title, textStyle: { color: textColor, fontSize: 12 }, left: 'center', bottom: 5 } : undefined,
    series: [{
      type: 'gauge',
      startAngle: 200,
      endAngle: -20,
      min: props.min,
      max: props.max,
      progress: { show: true, width: 12, itemStyle: { color } },
      axisLine: { lineStyle: { width: 12, color: [[1, props.dark ? '#1f2937' : '#e5e7eb']] } },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      pointer: { show: false },
      anchor: { show: false },
      detail: {
        valueAnimation: true,
        fontSize: 24,
        fontWeight: 'bold',
        color: textColor,
        offsetCenter: [0, '0%'],
        formatter: `{value}${props.unit}`,
      },
      data: [{ value: props.value }],
    }],
  }
}

watch(() => [props.value, props.dark], buildOption)
onMounted(buildOption)
</script>

<template>
  <VChart :option="option" :style="{ height }" autoresize class="w-full" />
</template>
