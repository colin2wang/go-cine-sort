export type ConflictStrategy = 'rename' | 'skip' | 'overwrite'

export interface AppConfig {
  movieFolder: string
  /** YouTube 归档源目录（!待整理）与目标根目录 */
  youtubeSourceDir: string
  youtubeTargetDir: string
  directoryFormat: string
  recursive: boolean
  preferSourceDir: boolean
  archiveMove: boolean
  conflictStrategy: ConflictStrategy
  createRatingFile: boolean
  createSidFile: boolean
  downloadPosters: boolean
  /** 海报保存格式：none 不转换 / png / jpg，默认 jpg */
  posterFormat: 'none' | 'png' | 'jpg'
  fetchImdb: boolean
  extensions: string[]
  cleanupPatterns: string[]
  techPatterns: string[]
  languagePatterns: string[]
  configPath: string
  dataDir?: string
}

export interface MovieFileInfo {
  filePath: string
  movieName: string
  year: string
  extension: string
  rawFilename: string
}

export type ItemStatus = 'pending' | 'running' | 'success' | 'failed' | 'skipped'

export interface MovieRow extends MovieFileInfo {
  status: ItemStatus
  directory: string
  error: string
  sid: string
  title: string
  rating: string
  posterCount: number
}

export interface ProcessResult {
  filename: string
  filePath: string
  movieName: string
  year: string
  directory: string
  videoPath: string
  error: string
  success: boolean
  skipped: boolean
  sid: string
  title: string
  rating: string
  posterCount: number
}

export interface TaskProgress {
  total: number
  completed: number
  success: number
  failed: number
  current: string
  stage: string
  percent: number
}

export interface TaskSummary {
  total: number
  processed: number
  success: number
  failed: number
  elapsedMs: number
  cancelled: boolean
}

export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

export interface LogEntry {
  time: string
  level: LogLevel
  message: string
  source: string
}

export interface MovieRecord {
  id: number
  sid: string
  title: string
  originalTitle: string
  year: string
  rating: string
  ratingCount: number
  imdbId: string
  directoryPath: string
  posterUrl: string
  createdAt: string
  posterCount: number
}

export interface PosterRecord {
  id: number
  posterId: string
  title: string
  keyword: string
  region: string
  thumbnailUrl: string
  originalUrl: string
  localPath: string
  fileSize: number
  isDownloaded: boolean
}

export interface MovieDetail extends MovieRecord {
  aliases: string
  description: string
  country: string
  language: string
  runtime: number
  top250Rank: number
  shortCommentsCount: number
  reviewsCount: number
  directors: string[]
  actors: string[]
  genres: string[]
  releaseDates: string[]
  awards: string[]
  similarMovies: string[]
  posters: PosterRecord[]
}

export interface HistoryStats {
  total: number
  averageRating: number
  posterTotal: number
}

export interface CacheSummary {
  key: string
  method: string
  url: string
  source: string
  purpose: string
  statusCode: number
  bodyBytes: number
  cachedAt: string
  expiresAt: string
}

export interface CacheEntry extends CacheSummary {
  header: Record<string, string>
  body: string
}

export interface CacheStats {
  count: number
  totalBodyBytes: number
}

export type YouTubeConflictAction = 'skip' | 'overwrite'

export interface YouTubeMovePlan {
  filename: string
  sourceDir: string
  uploader: string
  destDir: string
  destPath: string
}

export interface YouTubePlanResult {
  moves: YouTubeMovePlan[]
  conflicts: string[]
  unmatched: string[]
}

export interface MediaFileMeta {
  path: string
  dir: string
  filename: string
  sizeMB: number
  title: string
  comment: string
  newName: string
  renamable: boolean
  taggable: boolean
}

export const EVENT = {
  log: 'log',
  progress: 'progress',
  itemResult: 'item-result',
  taskDone: 'task-done',
  taskError: 'task-error',
} as const
