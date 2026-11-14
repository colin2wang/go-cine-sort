import { reactive, ref } from 'vue'
import { api, onItemResult, onLog, onProgress, onTaskDone, onTaskError } from '@/api'
import type { LogEntry, LogLevel, ProcessResult, TaskProgress, TaskSummary } from '@/types'

const MAX_LOGS = 600

/** 日志级别从低到高，供显示过滤使用。 */
export const LOG_LEVELS: LogLevel[] = ['DEBUG', 'INFO', 'WARN', 'ERROR']

/** 日志面板是否折叠（折叠时面板隐藏，展开按钮在状态栏）。 */
export const logPanelCollapsed = ref(false)

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
  logLevel: 'INFO' as LogLevel,
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

/** 切换日志级别：同时调整显示过滤与后端 logger 的最低级别（会话级，不持久化）。 */
export async function setLogLevel(level: LogLevel): Promise<void> {
  taskState.logLevel = level
  try {
    await api.setLogLevel(level)
  } catch (err) {
    pushLocalLog('WARN', `日志级别切换失败：${err}`, 'ui')
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
