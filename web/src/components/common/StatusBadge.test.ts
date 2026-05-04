import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusBadge from '@/components/common/StatusBadge.vue'

describe('StatusBadge', () => {
  // 辅助函数：创建包装器
  function createWrapper(status?: string) {
    return mount(StatusBadge, {
      props: { status },
      global: {
        stubs: {
          Badge: {
            template: `<span :class="$attrs.class" :data-variant="$attrs.variant"><slot /></span>`,
            inheritAttrs: true,
          },
        },
      },
    })
  }

  it('渲染 running 状态为成功样式', () => {
    const wrapper = createWrapper('running')
    expect(wrapper.text()).toBe('运行中')
    expect(wrapper.find('[data-variant="success"]').exists()).toBe(true)
  })

  it('渲染 stopped 状态为 secondary 样式', () => {
    const wrapper = createWrapper('stopped')
    expect(wrapper.text()).toBe('已停止')
    expect(wrapper.find('[data-variant="secondary"]').exists()).toBe(true)
  })

  it('渲染 error 状态为 destructive 样式', () => {
    const wrapper = createWrapper('error')
    expect(wrapper.text()).toBe('错误')
    expect(wrapper.find('[data-variant="destructive"]').exists()).toBe(true)
  })

  it('渲染 deploying 状态为 warning 样式', () => {
    const wrapper = createWrapper('deploying')
    expect(wrapper.text()).toBe('部署中')
    expect(wrapper.find('[data-variant="warning"]').exists()).toBe(true)
  })

  it('渲染未映射的状态为 secondary 样式并显示原始文本', () => {
    const wrapper = createWrapper('custom-status')
    expect(wrapper.text()).toBe('custom-status')
    expect(wrapper.find('[data-variant="secondary"]').exists()).toBe(true)
  })

  it('不传 status 时显示空文本', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).toBe('')
  })

  it('正确映射所有 success 类型的状态', () => {
    const successStatuses = ['running', 'success', 'active', 'healthy', 'online', 'completed', 'enabled']
    for (const status of successStatuses) {
      const wrapper = createWrapper(status)
      expect(wrapper.find('[data-variant="success"]').exists()).toBe(true)
    }
  })

  it('正确映射所有 destructive 类型的状态', () => {
    const destructiveStatuses = ['failed', 'error', 'expired', 'disabled', 'offline']
    for (const status of destructiveStatuses) {
      const wrapper = createWrapper(status)
      expect(wrapper.find('[data-variant="destructive"]').exists()).toBe(true)
    }
  })

  it('正确映射所有 warning 类型的状态', () => {
    const warningStatuses = ['deploying', 'pending', 'renewing', 'building', 'warning']
    for (const status of warningStatuses) {
      const wrapper = createWrapper(status)
      expect(wrapper.find('[data-variant="warning"]').exists()).toBe(true)
    }
  })

  it('支持传入自定义 class', () => {
    const wrapper = mount(StatusBadge, {
      props: { status: 'running', class: 'my-custom-class' },
      global: {
        stubs: {
          Badge: {
            template: `<span :class="$attrs.class"><slot /></span>`,
            inheritAttrs: true,
          },
        },
      },
    })
    expect(wrapper.find('span').classes()).toContain('my-custom-class')
  })
})
