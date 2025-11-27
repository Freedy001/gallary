import {defineStore} from 'pinia'
import {ref} from 'vue'
import type {DialogOptions, DialogState} from '@/types/dialog'

export const useDialogStore = defineStore('dialog', () => {
  const state = ref<DialogState>({
    visible: false,
    title: '',
    message: '',
    type: 'info'
  })

  // Keep track of the promise resolve function
  let resolvePromise: ((value: boolean) => void) | null = null

  function show(options: DialogOptions): Promise<boolean> {
    state.value = {
      ...options,
      visible: true,
      type: options.type || 'info',
      confirmText: options.confirmText || '确定',
      cancelText: options.cancelText
    }

    return new Promise((resolve) => {
      resolvePromise = resolve
    })
  }

  function confirm(options: DialogOptions | string): Promise<boolean> {
    const opts = typeof options === 'string' ? {title: '确认操作', message: options} : options
    return show({
      ...opts,
      type: opts.type || 'confirm',
      cancelText: '取消'
    })
  }

  function alert(options: DialogOptions | string): Promise<boolean> {
    const opts = typeof options === 'string' ? {title: '提示', message: options} : options
    return show({
      ...opts,
      type: opts.type || 'info',
    })
  }

  function handleConfirm() {
    state.value.visible = false
    state.value.onConfirm?.()
    if (resolvePromise) {
      resolvePromise(true)
      resolvePromise = null
    }
  }

  function handleCancel() {
    state.value.visible = false
    state.value.onCancel?.()
    if (resolvePromise) {
      resolvePromise(false)
      resolvePromise = null
    }
  }

  return {
    state,
    show,
    confirm,
    alert,
    handleConfirm,
    handleCancel
  }
})
