export type ConflictStrategy = 'rename' | 'skip' | 'overwrite'

export interface AppConfig {
  movieFolder: string
  directoryFormat: string
  recursive: boolean
  archiveMove: boolean
  conflictStrategy: ConflictStrategy
  createRatingFile: boolean
  createSidFile: boolean
  downloadPosters: boolean
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

export const EVENT = {
  log: 'log',
  progress: 'progress',
  itemResult: 'item-result',
  taskDone: 'task-done',
  taskError: 'task-error',
} as const
