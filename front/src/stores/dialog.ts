import {defineStore} from 'pinia'
import {ref} from 'vue'
import type {DialogOptions, DialogState, Notification} from '@/types/dialog'

export const useDialogStore = defineStore('dialog', () => {
  const state = ref<DialogState>({
    visible: false,
    title: '',
    message: '',
    type: 'info'
  })

  const notifications = ref<Notification[]>([])
  let notificationIdCounter = 0

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

  function notify(options: Omit<Notification, 'id'>) {
    const id = ++notificationIdCounter
    const notification: Notification = {
      id,
      ...options,
      duration: options.duration !== undefined ? options.duration : 3000
    }
    notifications.value.push(notification)
  }

  function removeNotification(id: number) {
    const index = notifications.value.findIndex(n => n.id === id)
    if (index !== -1) {
      notifications.value.splice(index, 1)
    }
  }

  function alert(options: DialogOptions | string) {
    const opts = typeof options === 'string' ? {title: '提示', message: options} : options
    notify({
      title: opts.title || '提示',
      message: opts.message,
      type: opts.type || 'info'
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
    notifications,
    removeNotification,
    handleConfirm,
    handleCancel,
    confirm: confirm,
    alert: alert,
    notify: notify,
  }
})
