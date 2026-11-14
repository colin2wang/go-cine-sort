<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api'
import { beginTask, clearLogs, installTaskEvents, pushLocalLog, taskState } from '@/store/task'
import { notify, reportError } from '@/store/toast'
import type { AppConfig, MovieDetail, ProcessResult } from '@/types'

const sid = ref('')
const cfg = ref<AppConfig | null>(null)
const busy = ref(false)
const result = ref<ProcessResult | null>(null)
const detail = ref<MovieDetail | null>(null)
const files = ref<string[]>([])
const posterBroken = ref(false)

const validSID = computed(() => /^\d{4,}$/.test(sid.value.trim()))
const posterURL = computed(() => {
  if (posterBroken.value) return ''
  return detail.value?.posters?.[0]?.originalUrl || detail.value?.posterUrl || ''
})
const flags = computed(() => [
  { key: 'downloadPosters' as const, label: '下载官方海报', hint: '抓取并筛选高清海报' },
  { key: 'fetchImdb' as const, label: '获取 IMDb 评分', hint: '写入 IMDb 评分文件' },
  { key: 'archiveMove' as const, label: '归档移动视频', hint: '生成目录后移动视频文件' },
])

onMounted(async () => {
  installTaskEvents()
  try {
    cfg.value = await api.getConfig()
  } catch (e) {
    reportError(e, '读取配置失败')
  }
})

function onPaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData('text')?.trim() ?? ''
  if (/^\d+$/.test(text)) {
    event.preventDefault()
    sid.value = text
    notify(`已从剪贴板填入 SID ${text}`, 'info', 2000)
  }
}

async function toggleFlag(field: 'downloadPosters' | 'fetchImdb' | 'archiveMove') {
  if (!cfg.value || busy.value) return
  const next = { ...cfg.value, [field]: !cfg.value[field] }
  cfg.value = next
  try {
    await api.saveConfig(next)
    notify('设置已保存', 'success', 2000)
  } catch (e) {
    reportError(e, '保存设置失败')
    cfg.value = await api.getConfig()
  }
}

async function run() {
  if (!validSID.value) {
    notify('请输入有效的豆瓣 SID（纯数字）', 'warning')
    return
  }
  busy.value = true
  clearLogs()
  beginTask(1)
  result.value = null
  detail.value = null
  files.value = []
  posterBroken.value = false
  pushLocalLog('INFO', `开始处理 SID ${sid.value.trim()}`, 'ui')
  try {
    const r = await api.processBySID(sid.value.trim())
    result.value = r
    if (r.success) {
      notify(`整理完成：${r.title || r.movieName}`, 'success')
      try {
        detail.value = await api.getHistoryDetail(sid.value.trim())
      } catch {
        /* 详情仅用于预览，失败不影响整理结果 */
      }
      if (r.directory) {
        try {
          files.value = await api.listDirectory(r.directory)
        } catch {
          files.value = []
        }
      }
    } else {
      notify(r.error || '整理失败', 'error', 6000)
    }
  } catch (e) {
    taskState.running = false
    reportError(e, '处理失败')
  } finally {
    busy.value = false
  }
}

async function openFolder() {
  if (!result.value?.directory) return
  try {
    await api.openFolder(result.value.directory)
  } catch (e) {
    reportError(e, '打开目录失败')
  }
}
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">单部处理</h1>
        <p class="page-subtitle">已知豆瓣 SID 时，跳过文件名识别直接整理一部影片</p>
      </div>
      <RouterLink to="/workbench" class="btn btn-ghost btn-sm">前往批量整理 →</RouterLink>
    </div>

    <section class="glass-card card-pad">
      <div class="row" style="align-items: flex-end; gap: 16px">
        <div class="field grow">
          <span class="field-label">豆瓣 SID</span>
          <input v-model="sid" class="input mono" placeholder="例如 1292052" spellcheck="false"
            :disabled="busy" @paste="onPaste" @keyup.enter="run" />
        </div>
        <button class="btn btn-primary" type="button" :disabled="busy || !validSID" @click="run">
          <span v-if="busy" class="spinner" />{{ busy ? '抓取中…' : '开始整理' }}
        </button>
      </div>
      <p class="field-hint" style="margin-top: 10px">
        支持从剪贴板粘贴纯数字 SID；也可在豆瓣影片页 URL 中找到，例如 /subject/<span class="accent">1292052</span>/
      </p>
    </section>

    <div class="grid-2 single-grid">
      <section class="glass-card card-pad preview-card">
        <h2 class="section-title" style="margin-bottom: 16px">影片信息</h2>
        <div v-if="busy" class="skeleton-wrap">
          <div class="skeleton sk-poster" />
          <div class="skeleton-lines">
            <div class="skeleton sk-line" style="width: 70%" />
            <div class="skeleton sk-line" style="width: 45%" />
            <div class="skeleton sk-line" style="width: 85%" />
            <div class="skeleton sk-line" style="width: 60%" />
          </div>
        </div>

        <div v-else-if="detail || result?.success" class="preview">
          <div class="poster-frame">
            <img v-if="posterURL" :src="posterURL" alt="影片海报" @error="posterBroken = true" />
            <div v-else class="poster-empty">
              <span>◈</span>
              <small>{{ posterBroken ? '海报加载失败' : '暂无海报' }}</small>
            </div>
          </div>
          <div class="preview-info">
            <div class="preview-title">{{ detail?.title || result?.title || result?.movieName }}</div>
            <div class="preview-sub dim">{{ detail?.originalTitle || '—' }}</div>
            <div class="preview-meta">
              <span v-if="detail?.rating || result?.rating" class="rating-badge mono">
                ★ {{ detail?.rating || result?.rating }}
              </span>
              <span v-if="detail?.year" class="dim mono">{{ detail.year }}</span>
              <span v-if="detail?.runtime" class="dim">{{ detail.runtime }} 分钟</span>
            </div>
            <p v-if="detail?.directors?.length" class="preview-line">
              <span class="dim">导演</span> {{ detail.directors.join(' / ') }}
            </p>
            <p v-if="detail?.actors?.length" class="preview-line">
              <span class="dim">主演</span> {{ detail.actors.slice(0, 5).join(' / ') }}
            </p>
            <p v-if="detail?.genres?.length" class="preview-line">
              <span class="dim">类型</span> {{ detail.genres.join(' / ') }}
            </p>
            <p v-if="detail?.description" class="preview-desc ellipsis-3">{{ detail.description }}</p>
          </div>
        </div>

        <div v-else class="empty-state" style="padding: 56px 10px">
          <div class="empty-icon">◉</div>
          <div>输入 SID 并点击「开始整理」后，这里会显示影片信息</div>
        </div>
      </section>

      <section class="glass-card card-pad">
        <h2 class="section-title" style="margin-bottom: 16px">处理选项</h2>
        <div class="flag-list">
          <div v-for="f in flags" :key="f.key" class="flag-row">
            <div>
              <div class="flag-label">{{ f.label }}</div>
              <div class="field-hint">{{ f.hint }}</div>
            </div>
            <div class="switch" :class="{ 'is-on': !!cfg?.[f.key] }" @click="toggleFlag(f.key)">
              <span class="switch-track" />
            </div>
          </div>
        </div>
        <div class="divider" style="margin: 16px 0" />
        <div class="field-hint">选项会写入应用配置，对批量整理同样生效。</div>
        <RouterLink to="/settings" class="link-more">前往设置查看更多 →</RouterLink>
      </section>
    </div>

    <section class="glass-card card-pad">
      <div class="row-between" style="margin-bottom: 14px">
        <h2 class="section-title">整理产物</h2>
        <button v-if="result?.directory" class="btn btn-ghost btn-sm" type="button" @click="openFolder">
          打开目录
        </button>
      </div>
      <div v-if="result" class="result-box">
        <div class="result-status">
          <span class="tag" :class="result.success ? 'tag-success' : 'tag-failed'">
            {{ result.success ? '整理成功' : '处理失败' }}
          </span>
          <span v-if="result.sid" class="dim mono">SID {{ result.sid }}</span>
          <span v-if="result.posterCount" class="dim">海报 {{ result.posterCount }} 张</span>
        </div>
        <div v-if="result.directory" class="result-path mono" :title="result.directory">
          {{ result.directory }}
        </div>
        <div v-if="result.videoPath" class="result-path mono">视频归档：{{ result.videoPath }}</div>
        <div v-if="result.error" class="result-error">{{ result.error }}</div>
        <div v-if="files.length" class="file-chips">
          <span v-for="f in files" :key="f" class="file-chip mono">{{ f }}</span>
        </div>
      </div>
      <div v-else class="empty-state" style="padding: 30px 10px">
        <div>尚无产物记录</div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.single-grid {
  align-items: stretch;
}
.preview {
  display: flex;
  gap: 18px;
}
.poster-frame {
  width: 132px;
  height: 190px;
  flex: none;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--line);
  background: rgba(11, 15, 26, 0.6);
  box-shadow: var(--shadow-soft);
}
.poster-frame img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  animation: fadeIn 0.4s var(--ease);
}
.poster-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-3);
}
.poster-empty span {
  font-size: 34px;
}
.preview-info {
  flex: 1;
  min-width: 0;
}
.preview-title {
  font-size: 19px;
  font-weight: 600;
  line-height: 1.35;
}
.preview-sub {
  font-size: 12px;
  margin-top: 3px;
}
.preview-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 12px 0 14px;
  font-size: 12px;
}
.rating-badge {
  padding: 3px 10px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--gold-1);
  background: rgba(245, 184, 65, 0.13);
  border: 1px solid rgba(245, 184, 65, 0.28);
}
.preview-line {
  font-size: 12px;
  line-height: 1.8;
  margin: 0;
  color: var(--text-1);
}
.preview-desc {
  margin: 10px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-2);
}
.ellipsis-3 {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.skeleton-wrap {
  display: flex;
  gap: 18px;
}
.skeleton {
  background: linear-gradient(90deg, rgba(148, 163, 184, 0.08), rgba(148, 163, 184, 0.2), rgba(148, 163, 184, 0.08));
  background-size: 200% 100%;
  animation: flow 1.6s linear infinite;
  border-radius: 8px;
}
.sk-poster {
  width: 132px;
  height: 190px;
  flex: none;
}
.skeleton-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-top: 10px;
}
.sk-line {
  height: 12px;
}
.flag-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.flag-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.flag-label {
  font-size: 13px;
  margin-bottom: 3px;
}
.link-more {
  display: inline-block;
  margin-top: 12px;
  font-size: 12px;
  color: var(--sky);
  text-decoration: none;
}
.link-more:hover {
  text-decoration: underline;
}
.result-box {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.result-status {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 12px;
}
.result-path {
  font-size: 11.5px;
  color: var(--text-2);
  word-break: break-all;
  user-select: text;
}
.result-error {
  font-size: 12px;
  color: #fca5a5;
  line-height: 1.6;
}
.file-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 4px;
}
.file-chip {
  font-size: 10.5px;
  padding: 4px 9px;
  border-radius: 6px;
  background: rgba(56, 189, 248, 0.09);
  border: 1px solid rgba(56, 189, 248, 0.2);
  color: var(--text-2);
}
</style>
