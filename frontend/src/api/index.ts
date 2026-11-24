import { Events } from '@wailsio/runtime'
import {
  CancelTask,
  ClearCache,
  GetCacheEntry,
  GetCacheStats,
  GetConfig,
  GetDefaults,
  GetHistoryDetail,
  GetHistoryStats,
  IsRunning,
  ListCache,
  ListDirectory,
  ListHistory,
  OpenFolder,
  Ping,
  PlanYouTubeSort,
  ProcessBySID,
  SaveConfig,
  ScanDirectory,
  ScanMediaFiles,
  SelectDirectory,
  SetLogLevel,
  StartBatch,
  StartMediaRename,
  StartYouTubeSort,
  UpdateMediaTags,
} from '../../bindings/go-cine-sort/app'
import type {
  AppConfig,
  CacheEntry,
  CacheStats,
  CacheSummary,
  HistoryStats,
  LogEntry,
  MediaFileMeta,
  LogLevel,
  MovieDetail,
  MovieFileInfo,
  MovieRecord,
  ProcessResult,
  TaskProgress,
  TaskSummary,
  YouTubeConflictAction,
  YouTubePlanResult,
} from '@/types'

export const api = {
  ping: () => Ping() as Promise<string>,
  getConfig: () => GetConfig() as Promise<AppConfig>,
  // 生成的绑定把 DataDir（运行时解析的 yaml:"-" 字段）推成必填 string，
  // 与手写契约的可选类型不兼容，此处显式转换。
  saveConfig: (config: AppConfig) =>
    SaveConfig(config as unknown as Parameters<typeof SaveConfig>[0]) as Promise<AppConfig>,
  selectDirectory: (current: string) => SelectDirectory(current) as Promise<string>,
  scanDirectory: (dir: string, recursive: boolean) =>
    ScanDirectory(dir, recursive) as Promise<MovieFileInfo[]>,
  startBatch: (dir: string, recursive: boolean, filePaths: string[]) =>
    StartBatch(dir, recursive, filePaths) as Promise<void>,
  processBySID: (sid: string) => ProcessBySID(sid) as Promise<ProcessResult>,
  cancelTask: () => CancelTask() as Promise<boolean>,
  listHistory: (keyword: string) => ListHistory(keyword) as Promise<MovieRecord[]>,
  getHistoryDetail: (sid: string) => GetHistoryDetail(sid) as Promise<MovieDetail>,
  getHistoryStats: () => GetHistoryStats() as Promise<HistoryStats>,
  openFolder: (path: string) => OpenFolder(path) as Promise<void>,
  getDefaults: () => GetDefaults() as Promise<AppConfig>,
  isRunning: () => IsRunning() as Promise<boolean>,
  listDirectory: (dir: string) => ListDirectory(dir) as Promise<string[]>,
  setLogLevel: (level: LogLevel) => SetLogLevel(level) as Promise<void>,
  listCache: () => ListCache() as Promise<CacheSummary[]>,
  planYouTubeSort: (sourceDir: string, targetBase: string) =>
    PlanYouTubeSort(sourceDir, targetBase) as Promise<YouTubePlanResult>,
  startYouTubeSort: (sourceDir: string, targetBase: string, conflictAction: YouTubeConflictAction) =>
    StartYouTubeSort(sourceDir, targetBase, conflictAction) as Promise<void>,
  getCacheEntry: (key: string) => GetCacheEntry(key) as Promise<CacheEntry>,
  getCacheStats: () => GetCacheStats() as Promise<CacheStats>,
  clearCache: () => ClearCache() as Promise<void>,
  scanMediaFiles: (dir: string, recursive: boolean) =>
    ScanMediaFiles(dir, recursive) as Promise<MediaFileMeta[]>,
  updateMediaTags: (path: string, title: string, comment: string) =>
    UpdateMediaTags(path, title, comment) as Promise<void>,
  startMediaRename: (dir: string, recursive: boolean) =>
    StartMediaRename(dir, recursive) as Promise<void>,
}

type Unsubscribe = () => void

/**
 * 订阅 Wails 事件。约定每个事件携带一个 payload 对象。
 * v3 的 Events.On 回调收到事件对象，payload 在 data 字段中；
 * 若运行时直接给出 payload（老版本行为），也一并兼容。
 */
export function on<T>(eventName: string, handler: (payload: T) => void): Unsubscribe {
  return Events.On(eventName, (event: unknown) => {
    const data = (event as { data?: unknown })?.data
    if (data !== undefined) {
      handler(data as T)
      return
    }
    handler(event as T)
  })
}

export const onLog = (h: (p: LogEntry) => void) => on<LogEntry>('log', h)
export const onProgress = (h: (p: TaskProgress) => void) => on<TaskProgress>('progress', h)
export const onItemResult = (h: (p: ProcessResult) => void) => on<ProcessResult>('item-result', h)
export const onTaskDone = (h: (p: TaskSummary) => void) => on<TaskSummary>('task-done', h)
export const onTaskError = (h: (p: { message: string }) => void) =>
  on<{ message: string }>('task-error', h)
