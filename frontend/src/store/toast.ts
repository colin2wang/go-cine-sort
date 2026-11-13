import { reactive } from 'vue'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
}

export const toasts = reactive<ToastItem[]>([])
let sequence = 0

export function notify(message: string, type: ToastType = 'info', ttl = 3400): void {
  const id = ++sequence
  toasts.push({ id, type, message })
  window.setTimeout(() => dismiss(id), ttl)
}

export function dismiss(id: number): void {
  const index = toasts.findIndex((t) => t.id === id)
  if (index >= 0) toasts.splice(index, 1)
}

export function reportError(err: unknown, fallback = '操作失败'): void {
  const message = err instanceof Error ? err.message : String(err ?? fallback)
  notify(message || fallback, 'error', 5000)
}
