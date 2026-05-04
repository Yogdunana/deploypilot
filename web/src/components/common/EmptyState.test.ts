import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import EmptyState from '@/components/common/EmptyState.vue'

// Mock Button 组件
const MockButton = {
  template: `<button data-testid="action-button" @click="$emit('click')"><slot /></button>`,
  props: ['size'],
  emits: ['click'],
}

describe('EmptyState', () => {
  it('渲染标题和描述', () => {
    const wrapper = mount(EmptyState, {
      props: {
        title: '暂无数据',
        description: '请添加第一条记录',
      },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('h3').text()).toBe('暂无数据')
    expect(wrapper.find('p').text()).toBe('请添加第一条记录')
  })

  it('不传 title 时不渲染 h3 元素', () => {
    const wrapper = mount(EmptyState, {
      props: { description: '描述文本' },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('h3').exists()).toBe(false)
  })

  it('不传 description 时不渲染 p 元素', () => {
    const wrapper = mount(EmptyState, {
      props: { title: '标题' },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('p').exists()).toBe(false)
  })

  it('渲染操作按钮', () => {
    const wrapper = mount(EmptyState, {
      props: {
        title: '暂无数据',
        actionText: '添加数据',
      },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('[data-testid="action-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="action-button"]').text()).toBe('添加数据')
  })

  it('点击操作按钮触发 action 事件', async () => {
    const wrapper = mount(EmptyState, {
      props: {
        title: '暂无数据',
        actionText: '添加',
      },
      global: { stubs: { Button: MockButton } },
    })

    await wrapper.find('[data-testid="action-button"]').trigger('click')
    expect(wrapper.emitted('action')).toBeTruthy()
    expect(wrapper.emitted('action')!.length).toBe(1)
  })

  it('不传 actionText 时不渲染操作按钮', () => {
    const wrapper = mount(EmptyState, {
      props: { title: '标题' },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('[data-testid="action-button"]').exists()).toBe(false)
  })

  it('通过 icon prop 渲染图标', () => {
    const MockIcon = { template: '<svg data-testid="mock-icon" />' }
    const wrapper = mount(EmptyState, {
      props: {
        title: '标题',
        icon: MockIcon,
      },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('[data-testid="mock-icon"]').exists()).toBe(true)
  })

  it('通过 icon slot 渲染图标（优先级高于 icon prop）', () => {
    const MockIconProp = { template: '<svg data-testid="prop-icon" />' }
    const wrapper = mount(EmptyState, {
      props: {
        title: '标题',
        icon: MockIconProp,
      },
      slots: {
        icon: '<span data-testid="slot-icon">slot icon</span>',
      },
      global: { stubs: { Button: MockButton } },
    })
    // slot 优先于 prop
    expect(wrapper.find('[data-testid="slot-icon"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="prop-icon"]').exists()).toBe(false)
  })

  it('支持传入自定义 class', () => {
    const wrapper = mount(EmptyState, {
      props: { class: 'my-empty-class' },
      global: { stubs: { Button: MockButton } },
    })
    expect(wrapper.find('div').classes()).toContain('my-empty-class')
  })
})
