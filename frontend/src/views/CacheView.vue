<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api'
import { reportError } from '@/store/toast'
import type { CacheEntry, CacheStats, CacheSummary } from '@/types'

defineOptions({ name: 'CacheView' })

const entries = ref<CacheSummary[]>([])
const stats = ref<CacheStats>({ count: 0, totalBodyBytes: 0 })
const loading = ref(false)
const filter = ref<'all' | 'douban' | 'imdb' | 'other'>('all')
const drawerOpen = ref(false)
const entry = ref<CacheEntry | null>(null)
const entryLoading = ref(false)
const copied = ref(false)

const filtered = computed(() =>
  filter.value === 'all' ? entries.value : entries.value.filter((e) => sourceOf(e) === filter.value),
)

const statCards = computed(() => [
  { label: '缓存条目', value: String(stats.value.count), tone: 'tone-sky', foot: '存活中的响应记录' },
  {
    label: '缓存体积',
    value: formatBytes(stats.value.totalBodyBytes),
    tone: 'tone-gold',
    foot: '响应体累计字节',
  },
  { label: '记录上限', value: '16 MB / 条', tone: 'tone-green', foot: '超过单条截断' },
])

function sourceOf(e: CacheSummary): 'douban' | 'imdb' | 'other' {
  if (e.source === 'douban' || e.source === 'imdb') return e.source
  return 'other'
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

function formatTime(value: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

async function load() {
  loading.value = true
  try {
    const [list, s] = await Promise.all([api.listCache(), api.getCacheStats()])
    entries.value = list
    stats.value = s
  } catch (e) {
    reportError(e, '读取缓存列表失败')
  } finally {
    loading.value = false
  }
}

async function openEntry(key: string) {
  drawerOpen.value = true
  entryLoading.value = true
  entry.value = null
  copied.value = false
  try {
    entry.value = await api.getCacheEntry(key)
  } catch (e) {
    reportError(e, '读取缓存详情失败')
  } finally {
    entryLoading.value = false
  }
}

async function clearAll() {
  try {
    await api.clearCache()
    await load()
  } catch (e) {
    reportError(e, '清空缓存失败')
  }
}

function prettyBody(body: string): string {
  const trimmed = body.trim()
  if (!(trimmed.startsWith('{') || trimmed.startsWith('['))) return body
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return body
  }
}

async function copyBody() {
  if (!entry.value) return
  try {
    await navigator.clipboard.writeText(entry.value.body)
    copied.value = true
    window.setTimeout(() => (copied.value = false), 1500)
  } catch {
    reportError(new Error('剪贴板不可用'), '复制失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">响应缓存</h1>
        <p class="page-subtitle">检查外部 API 与页面请求的原始响应（GET 命中直接回放，POST 仅记录）</p>
      </div>
      <div class="head-actions">
        <button class="btn btn-ghost btn-sm" type="button" :disabled="loading" @click="load">
          <span v-if="loading" class="spinner" />刷新
        </button>
        <button class="btn btn-danger btn-sm" type="button" :disabled="stats.count === 0" @click="clearAll">
          清空缓存
        </button>
      </div>
    </div>

    <div class="metric-grid three">
      <div v-for="(c, i) in statCards" :key="c.label" class="glass-card metric-card is-hoverable"
        :class="c.tone" :style="{ animationDelay: `${i * 60}ms` }">
        <div class="metric-label">{{ c.label }}</div>
        <div class="metric-value mono">{{ c.value }}</div>
        <div class="metric-foot">{{ c.foot }}</div>
      </div>
    </div>

    <section class="glass-card card-pad">
      <div class="row-between" style="margin-bottom: 14px">
        <h2 class="section-title">缓存记录</h2>
        <div class="filter-tabs">
          <button v-for="f in (['all', 'douban', 'imdb', 'other'] as const)" :key="f" type="button"
            class="filter-tab" :class="{ active: filter === f }" @click="filter = f">
            {{ f === 'all' ? '全部' : f }}
          </button>
        </div>
      </div>
      <div v-if="filtered.length === 0 && !loading" class="empty-hint">
        暂无缓存记录，运行一次整理任务后这里会显示所有外部请求的响应。
      </div>
      <div v-else class="cache-list">
        <button v-for="e in filtered" :key="e.key" type="button" class="cache-row" @click="openEntry(e.key)">
          <span class="cache-method" :class="`method-${e.method.toLowerCase()}`">{{ e.method }}</span>
          <span class="cache-status" :class="e.statusCode < 400 ? 'ok' : 'bad'">{{ e.statusCode }}</span>
          <span class="cache-main">
            <span class="cache-url mono">{{ e.url }}</span>
            <span class="cache-purpose">{{ e.purpose || '未标注用途' }}</span>
          </span>
          <span class="cache-meta">{{ sourceOf(e) }} · {{ formatBytes(e.bodyBytes) }} · {{ formatTime(e.cachedAt) }}</span>
        </button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="drawerOpen" class="drawer-mask" @click.self="drawerOpen = false">
        <div class="drawer glass-card">
          <div class="row-between" style="margin-bottom: 12px">
            <h2 class="section-title">响应详情</h2>
            <button class="btn btn-ghost btn-sm" type="button" @click="drawerOpen = false">关闭</button>
          </div>
          <div v-if="entryLoading" class="empty-hint">加载中…</div>
          <template v-else-if="entry">
            <div class="detail-meta">
              <div><span class="field-label">URL</span><div class="mono detail-url">{{ entry.url }}</div></div>
              <div class="detail-chips">
                <span class="chip purpose-chip">{{ entry.purpose || '未标注用途' }}</span>
                <span class="cache-method" :class="`method-${entry.method.toLowerCase()}`">{{ entry.method }}</span>
                <span class="cache-status" :class="entry.statusCode < 400 ? 'ok' : 'bad'">{{ entry.statusCode }}</span>
                <span class="chip">{{ formatBytes(entry.bodyBytes) }}</span>
                <span class="chip">缓存于 {{ formatTime(entry.cachedAt) }}</span>
                <span class="chip">过期 {{ formatTime(entry.expiresAt) }}</span>
              </div>
            </div>
            <div class="detail-block">
              <span class="field-label">响应头</span>
              <pre class="code-view small">{{ entry.header }}</pre>
            </div>
            <div class="detail-block grow">
              <div class="row-between">
                <span class="field-label">响应体</span>
                <button class="btn btn-ghost btn-sm" type="button" @click="copyBody">
                  {{ copied ? '已复制' : '复制原文' }}
                </button>
              </div>
              <pre class="code-view grow">{{ prettyBody(entry.body) }}</pre>
            </div>
          </template>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.head-actions {
  display: flex;
  gap: 8px;
}
.filter-tabs {
  display: flex;
  gap: 4px;
}
.filter-tab {
  border: none;
  background: transparent;
  color: var(--text-3);
  font-size: 12px;
  font-family: inherit;
  padding: 5px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: color 0.2s, background 0.2s;
}
.filter-tab:hover {
  color: var(--text-1);
  background: rgba(148, 163, 184, 0.08);
}
.filter-tab.active {
  color: var(--gold-1);
  background: rgba(245, 184, 65, 0.14);
}
.cache-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 46vh;
  overflow: auto;
}
.cache-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 12px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: rgba(148, 163, 184, 0.03);
  color: var(--text-2);
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}
.cache-row:hover {
  border-color: rgba(245, 184, 65, 0.45);
  background: rgba(245, 184, 65, 0.06);
}
.cache-method {
  flex: none;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 6px;
}
.method-get {
  color: #7dd3fc;
  background: rgba(125, 211, 252, 0.12);
}
.method-post {
  color: #fbbf24;
  background: rgba(251, 191, 36, 0.12);
}
.cache-status {
  flex: none;
  font-size: 11px;
  font-weight: 600;
  width: 32px;
  text-align: center;
  padding: 2px 0;
  border-radius: 6px;
}
.cache-status.ok {
  color: #86efac;
  background: rgba(134, 239, 172, 0.12);
}
.cache-status.bad {
  color: #fca5a5;
  background: rgba(252, 165, 165, 0.12);
}
.cache-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cache-url {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cache-purpose {
  font-size: 11px;
  color: var(--text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cache-meta {
  flex: none;
  font-size: 11px;
  color: var(--text-3);
}
.chip {
  font-size: 11px;
  color: var(--text-3);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 2px 8px;
}
.purpose-chip {
  color: var(--gold-1);
  border-color: rgba(245, 184, 65, 0.35);
  background: rgba(245, 184, 65, 0.08);
}
.empty-hint {
  color: var(--text-3);
  font-size: 13px;
  padding: 22px 6px;
  text-align: center;
}
.drawer-mask {
  position: fixed;
  inset: 0;
  background: rgba(5, 8, 16, 0.62);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: stretch;
  justify-content: flex-end;
  z-index: 60;
}
.drawer {
  width: min(720px, 92vw);
  margin: 16px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
  border-radius: 14px;
}
.detail-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.detail-url {
  font-size: 12px;
  word-break: break-all;
  color: var(--text-1);
}
.detail-chips {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.detail-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 0;
}
.detail-block.grow {
  flex: 1;
}
.code-view {
  margin: 0;
  padding: 12px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: rgba(5, 8, 16, 0.55);
  color: var(--text-2);
  font-size: 12px;
  line-height: 1.55;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.code-view.small {
  max-height: 160px;
  flex: none;
}
.code-view.grow {
  flex: 1;
  min-height: 0;
}
</style>
