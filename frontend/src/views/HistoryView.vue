<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { api } from '@/api'
import { reportError } from '@/store/toast'
import type { HistoryStats, MovieDetail, MovieRecord } from '@/types'

const keyword = ref('')
const sortBy = ref<'newest' | 'rating'>('newest')
const records = ref<MovieRecord[]>([])
const stats = ref<HistoryStats>({ total: 0, averageRating: 0, posterTotal: 0 })
const loading = ref(false)
const drawerOpen = ref(false)
const detail = ref<MovieDetail | null>(null)
const detailLoading = ref(false)
let debounce: number | undefined

const sorted = computed(() => {
  const list = [...records.value]
  if (sortBy.value === 'rating') {
    return list.sort((a, b) => parseFloat(b.rating || '0') - parseFloat(a.rating || '0'))
  }
  return list
})
const statCards = computed(() => [
  { label: '已整理影片', value: stats.value.total, tone: 'tone-sky', foot: 'SQLite 持久化记录' },
  { label: '平均评分', value: stats.value.averageRating.toFixed(1), tone: 'tone-gold', foot: '豆瓣评分均值' },
  { label: '归档海报', value: stats.value.posterTotal, tone: 'tone-green', foot: '已下载海报记录' },
])

async function load() {
  loading.value = true
  try {
    const [list, s] = await Promise.all([api.listHistory(keyword.value), api.getHistoryStats()])
    records.value = list
    stats.value = s
  } catch (e) {
    reportError(e, '读取历史记录失败')
  } finally {
    loading.value = false
  }
}

async function openDetail(sid: string) {
  drawerOpen.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await api.getHistoryDetail(sid)
  } catch (e) {
    reportError(e, '读取详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function openFolder(path: string) {
  try {
    await api.openFolder(path)
  } catch (e) {
    reportError(e, '打开目录失败')
  }
}

function formatTime(value: string) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(load)
onUnmounted(() => {
  if (debounce !== undefined) window.clearTimeout(debounce)
})
watch(keyword, () => {
  if (debounce !== undefined) window.clearTimeout(debounce)
  debounce = window.setTimeout(load, 300)
})
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">历史记录</h1>
        <p class="page-subtitle">检索已整理影片、查看完整元数据与海报归档</p>
      </div>
      <button class="btn btn-ghost btn-sm" type="button" :disabled="loading" @click="load">
        <span v-if="loading" class="spinner" />刷新
      </button>
    </div>

    <section class="glass-card card-pad">
      <div class="search-row">
        <div class="field grow">
          <span class="field-label">关键字</span>
          <input v-model="keyword" class="input" placeholder="按片名、原名、SID 或目录检索…" spellcheck="false" />
        </div>
        <div class="field">
          <span class="field-label">排序</span>
          <select v-model="sortBy" class="select">
            <option value="newest">最近整理</option>
            <option value="rating">评分最高</option>
          </select>
        </div>
      </div>
    </section>

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
        <h2 class="section-title">整理记录</h2>
        <span class="dim" style="font-size: 12px">{{ sorted.length }} 条</span>
      </div>
      <div class="table-wrap" style="max-height: 420px">
        <table class="data-table">
          <thead>
            <tr>
              <th style="min-width: 180px">片名</th>
              <th style="width: 70px">年份</th>
              <th style="width: 80px">评分</th>
              <th style="min-width: 240px">整理目录</th>
              <th style="width: 150px">整理时间</th>
              <th style="width: 130px">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in sorted" :key="r.sid">
              <td>
                <div class="movie-name">{{ r.title }}</div>
                <div class="cell-dim ellipsis" style="max-width: 240px">{{ r.originalTitle || r.sid }}</div>
              </td>
              <td class="mono cell-dim">{{ r.year || '—' }}</td>
              <td><span v-if="r.rating" class="score mono">{{ r.rating }}</span><span v-else class="dim">—</span></td>
              <td class="cell-dim ellipsis" style="max-width: 300px" :title="r.directoryPath">
                {{ r.directoryPath || '—' }}
              </td>
              <td class="cell-dim">{{ formatTime(r.createdAt) }}</td>
              <td>
                <div class="ops">
                  <button class="link-btn" type="button" @click="openDetail(r.sid)">详情</button>
                  <button v-if="r.directoryPath" class="link-btn" type="button" @click="openFolder(r.directoryPath)">
                    打开
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="sorted.length === 0">
              <td colspan="6">
                <div class="empty-state">
                  <div class="empty-icon">◷</div>
                  <div>{{ keyword ? '没有匹配的记录' : '暂无整理记录，先在「批量整理」中执行一次任务' }}</div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <Transition name="drawer">
      <div v-if="drawerOpen" class="drawer-mask" @click.self="drawerOpen = false">
        <aside class="drawer glass-card">
          <div class="drawer-head">
            <h2 class="section-title">影片详情</h2>
            <button class="btn btn-ghost btn-icon" type="button" @click="drawerOpen = false">×</button>
          </div>
          <div v-if="detailLoading" class="drawer-loading">
            <span class="spinner" /> 正在加载详情…
          </div>
          <div v-else-if="detail" class="drawer-body">
            <div class="detail-poster">
              <img v-if="detail.posters?.[0]?.originalUrl || detail.posterUrl"
                :src="detail.posters?.[0]?.originalUrl || detail.posterUrl" alt="海报" />
              <div v-else class="poster-placeholder">◈</div>
            </div>
            <div class="detail-title">{{ detail.title }}</div>
            <div class="dim detail-sub">{{ detail.originalTitle || '—' }}</div>
            <div class="detail-grid">
              <div><span>评分</span><b class="mono accent">{{ detail.rating || '—' }}</b></div>
              <div><span>年份</span><b class="mono">{{ detail.year || '—' }}</b></div>
              <div><span>片长</span><b class="mono">{{ detail.runtime ? `${detail.runtime} 分钟` : '—' }}</b></div>
              <div><span>SID</span><b class="mono">{{ detail.sid }}</b></div>
            </div>
            <template v-for="(items, label) in {
              '导演': detail.directors, '主演': detail.actors,
              '类型': detail.genres, '上映日期': detail.releaseDates,
            }" :key="label">
              <div v-if="items?.length" class="detail-section">
                <div class="detail-label">{{ label }}</div>
                <div class="detail-value">{{ items.join(' / ') }}</div>
              </div>
            </template>
            <div v-if="detail.description" class="detail-section">
              <div class="detail-label">剧情简介</div>
              <div class="detail-value desc">{{ detail.description }}</div>
            </div>
            <div v-if="detail.posters?.length" class="detail-section">
              <div class="detail-label">海报归档（{{ detail.posters.length }}）</div>
              <div class="poster-wall">
                <a v-for="p in detail.posters" :key="p.id" :href="p.originalUrl" target="_blank"
                  :title="p.title">
                  <img :src="p.thumbnailUrl || p.originalUrl" :alt="p.title" />
                </a>
              </div>
            </div>
            <div v-if="detail.directoryPath" class="detail-section">
              <div class="detail-label">整理目录</div>
              <div class="detail-value mono path">{{ detail.directoryPath }}</div>
            </div>
          </div>
          <div v-else class="empty-state">详情加载失败</div>
        </aside>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.search-row {
  display: flex;
  align-items: flex-end;
  gap: 16px;
}
.search-row .select {
  min-width: 150px;
}
.metric-grid.three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.metric-card {
  animation: fadeInUp 0.45s var(--ease) both;
}
.movie-name {
  font-size: 13px;
  font-weight: 500;
}
.score {
  color: var(--gold-1);
  font-weight: 600;
  font-size: 14px;
}
.ops {
  display: flex;
  gap: 10px;
}
.link-btn {
  border: none;
  background: transparent;
  color: var(--sky);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  padding: 0;
}
.link-btn:hover {
  color: var(--gold-1);
  text-decoration: underline;
}
.drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: rgba(4, 7, 14, 0.6);
  backdrop-filter: blur(3px);
  display: flex;
  justify-content: flex-end;
}
.drawer {
  width: 460px;
  max-width: 92vw;
  height: 100%;
  border-radius: 0;
  border-right: none;
  border-top: none;
  border-bottom: none;
  background: linear-gradient(160deg, rgba(28, 36, 56, 0.98), rgba(11, 15, 26, 0.98));
  display: flex;
  flex-direction: column;
  box-shadow: -20px 0 60px rgba(0, 0, 0, 0.5);
}
.drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 22px;
  border-bottom: 1px solid var(--line);
}
.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 22px;
}
.drawer-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 40px 22px;
  color: var(--text-2);
  font-size: 13px;
}
.detail-poster {
  width: 130px;
  height: 186px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--line);
  background: rgba(11, 15, 26, 0.6);
  margin-bottom: 18px;
}
.detail-poster img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.poster-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 38px;
  color: var(--text-3);
}
.detail-title {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.35;
}
.detail-sub {
  font-size: 12px;
  margin-top: 4px;
}
.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin: 20px 0;
  padding: 16px;
  border-radius: 12px;
  background: rgba(11, 15, 26, 0.5);
  border: 1px solid var(--line);
}
.detail-grid div {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.detail-grid span {
  font-size: 11px;
  color: var(--text-3);
}
.detail-grid b {
  font-size: 14px;
  font-weight: 600;
}
.detail-section {
  margin-bottom: 18px;
}
.detail-label {
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.6px;
  margin-bottom: 7px;
}
.detail-value {
  font-size: 12.5px;
  line-height: 1.75;
  color: var(--text-1);
}
.detail-value.desc {
  color: var(--text-2);
}
.detail-value.path {
  font-size: 11px;
  word-break: break-all;
  user-select: text;
}
.poster-wall {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}
.poster-wall a {
  display: block;
  aspect-ratio: 2 / 3;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--line);
  transition: transform 0.2s var(--ease), border-color 0.2s;
}
.poster-wall a:hover {
  transform: translateY(-3px);
  border-color: var(--gold-1);
}
.poster-wall img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.28s var(--ease);
}
.drawer-enter-active .drawer,
.drawer-leave-active .drawer {
  transition: transform 0.32s var(--ease);
}
.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}
.drawer-enter-from .drawer,
.drawer-leave-to .drawer {
  transform: translateX(100%);
}
</style>
