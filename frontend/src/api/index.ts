import type {
  AppConfig,
  CacheEntry,
  CacheStats,
  CacheSummary,
  HistoryStats,
  LogEntry,
  LogLevel,
  MovieDetail,
  MovieFileInfo,
  MovieRecord,
  ProcessResult,
  TaskProgress,
  TaskSummary,
} from '@/types'

type BindingFn = (...args: any[]) => Promise<any>

function bindings(): Record<string, BindingFn> {
  const app = window.go?.main?.App
  if (!app) {
    throw new Error('Wails 运行时未就绪，请在桌面窗口中打开本应用')
  }
  return app
}

async function invoke<T>(method: string, ...args: any[]): Promise<T> {
  const fn = bindings()[method]
  if (typeof fn !== 'function') {
    throw new Error(`后端方法 ${method} 不可用，请重新构建应用`)
  }
  return (await fn(...args)) as T
}

export const api = {
  ping: () => invoke<string>('Ping'),
  getConfig: () => invoke<AppConfig>('GetConfig'),
  saveConfig: (config: AppConfig) => invoke<AppConfig>('SaveConfig', config),
  selectDirectory: (current: string) => invoke<string>('SelectDirectory', current),
  scanDirectory: (dir: string, recursive: boolean) =>
    invoke<MovieFileInfo[]>('ScanDirectory', dir, recursive),
  startBatch: (dir: string, recursive: boolean, filePaths: string[]) =>
    invoke<void>('StartBatch', dir, recursive, filePaths),
  processBySID: (sid: string) => invoke<ProcessResult>('ProcessBySID', sid),
  cancelTask: () => invoke<boolean>('CancelTask'),
  listHistory: (keyword: string) => invoke<MovieRecord[]>('ListHistory', keyword),
  getHistoryDetail: (sid: string) => invoke<MovieDetail>('GetHistoryDetail', sid),
  getHistoryStats: () => invoke<HistoryStats>('GetHistoryStats'),
  openFolder: (path: string) => invoke<void>('OpenFolder', path),
  getDefaults: () => invoke<AppConfig>('GetDefaults'),
  isRunning: () => invoke<boolean>('IsRunning'),
  listDirectory: (dir: string) => invoke<string[]>('ListDirectory', dir),
  setLogLevel: (level: LogLevel) => invoke<void>('SetLogLevel', level),
  listCache: () => invoke<CacheSummary[]>('ListCache'),
  getCacheEntry: (key: string) => invoke<CacheEntry>('GetCacheEntry', key),
  getCacheStats: () => invoke<CacheStats>('GetCacheStats'),
  clearCache: () => invoke<void>('ClearCache'),
}

type Unsubscribe = () => void

/** 订阅 Wails 事件。约定每个事件携带一个 payload 对象。 */
export function on<T>(eventName: string, handler: (payload: T) => void): Unsubscribe {
  const rt = window.runtime
  if (!rt) return () => {}
  return rt.EventsOn(eventName, (...data: any[]) => {
    if (data.length > 0) handler(data[0] as T)
  })
}

export function off(eventName: string, ...additionalEventNames: string[]): void {
  window.runtime?.EventsOff(eventName, ...additionalEventNames)
}

export const onLog = (h: (p: LogEntry) => void) => on<LogEntry>('log', h)
export const onProgress = (h: (p: TaskProgress) => void) => on<TaskProgress>('progress', h)
export const onItemResult = (h: (p: ProcessResult) => void) => on<ProcessResult>('item-result', h)
export const onTaskDone = (h: (p: TaskSummary) => void) => on<TaskSummary>('task-done', h)
export const onTaskError = (h: (p: { message: string }) => void) =>
  on<{ message: string }>('task-error', h)
