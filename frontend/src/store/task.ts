import { reactive } from 'vue'
import { onItemResult, onLog, onProgress, onTaskDone, onTaskError } from '@/api'
import type { LogEntry, ProcessResult, TaskProgress, TaskSummary } from '@/types'

const MAX_LOGS = 600

const emptyProgress = (): TaskProgress => ({
  total: 0,
  completed: 0,
  success: 0,
  failed: 0,
  current: '',
  stage: '空闲',
  percent: 0,
})

export const taskState = reactive({
  running: false,
  logs: [] as LogEntry[],
  results: [] as ProcessResult[],
  progress: emptyProgress(),
  summary: null as TaskSummary | null,
  lastError: '',
  lastResult: null as ProcessResult | null,
})

export function clearLogs(): void {
  taskState.logs.splice(0)
}

export function resetTask(): void {
  taskState.progress = emptyProgress()
  taskState.summary = null
  taskState.lastError = ''
  taskState.results.splice(0)
  clearLogs()
}

export function beginTask(total: number): void {
  taskState.running = true
  taskState.summary = null
  taskState.lastError = ''
  taskState.results.splice(0)
  taskState.progress = {
    ...emptyProgress(),
    total,
    stage: '准备中',
  }
}

export function pushLocalLog(level: LogEntry['level'], message: string, source = 'ui'): void {
  taskState.logs.push({
    time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
    level,
    message,
    source,
  })
  if (taskState.logs.length > MAX_LOGS) {
    taskState.logs.splice(0, taskState.logs.length - MAX_LOGS)
  }
}

let installed = false

/** 只安装一次全局事件监听，避免路由切换造成重复订阅。 */
export function installTaskEvents(): void {
  if (installed) return
  installed = true

  onLog((entry) => {
    taskState.logs.push(entry)
    if (taskState.logs.length > MAX_LOGS) {
      taskState.logs.splice(0, taskState.logs.length - MAX_LOGS)
    }
  })

  onProgress((p) => {
    taskState.progress = p
  })

  onItemResult((r) => {
    taskState.lastResult = r
    taskState.results.push(r)
  })

  onTaskDone((s) => {
    taskState.running = false
    taskState.summary = s
    taskState.progress = {
      ...taskState.progress,
      completed: s.processed,
      success: s.success,
      failed: s.failed,
      percent: s.total > 0 ? 100 : 0,
      stage: s.cancelled ? '已取消' : '已完成',
    }
  })

  onTaskError((e) => {
    taskState.running = false
    taskState.lastError = e.message
    taskState.progress = { ...taskState.progress, stage: '出错' }
    pushLocalLog('ERROR', e.message, 'backend')
  })
}
