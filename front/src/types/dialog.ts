export type DialogType = 'info' | 'success' | 'warning' | 'error' | 'confirm'

export interface DialogOptions {
  title: string
  message: string
  type?: DialogType
  confirmText?: string
  cancelText?: string
  onConfirm?: () => void
  onCancel?: () => void
}

export interface DialogState extends DialogOptions {
  visible: boolean
  resolve?: (value: boolean) => void
}

export interface Notification {
  id: number
  title?: string
  message: string
  type: DialogType
  duration?: number
}
