<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { api } from '@/api'
import MovieTable from '@/components/MovieTable.vue'
import ProgressPanel from '@/components/ProgressPanel.vue'
import { beginTask, clearLogs, installTaskEvents, pushLocalLog, taskState } from '@/store/task'
import { notify, reportError } from '@/store/toast'
import type { MovieRow, ProcessResult } from '@/types'

const dir = ref('')
const recursive = ref(true)
const rows = ref<MovieRow[]>([])
const selected = ref<string[]>([])
const scanning = ref(false)
const elapsed = ref(0)
let timer: number | undefined

const running = computed(() => taskState.running || scanning.value)
const elapsedText = computed(() => {
  const ms = taskState.summary?.elapsedMs ?? elapsed.value
  if (ms < 1000) return `${ms}ms`
  const total = Math.floor(ms / 1000)
  if (total < 60) return `${total}s`
  return `${Math.floor(total / 60)}m ${total % 60}s`
})
const metrics = computed(() => [
  { label: '文件总数', value: rows.value.length, tone: 'tone-sky', foot: selected.value.length ? `已选 ${selected.value.length} 个` : '全部待处理' },
  { label: '整理成功', value: taskState.progress.success, tone: 'tone-green', foot: '已生成目录与元数据' },
  { label: '处理失败', value: taskState.progress.failed, tone: 'tone-red', foot: '详见日志与状态列' },
  { label: '耗时', value: elapsedText.value, tone: 'tone-gold', foot: taskState.summary?.cancelled ? '任务已取消' : '实时统计' },
])

onMounted(async () => {
  installTaskEvents()
  try {
    const cfg = await api.getConfig()
    dir.value = cfg.movieFolder
    recursive.value = cfg.recursive
  } catch (e) {
    reportError(e, '读取配置失败')
  }
  timer = window.setInterval(() => {
    if (taskState.running) elapsed.value += 500
  }, 500)
})

onUnmounted(() => {
  if (timer !== undefined) window.clearInterval(timer)
})

watch(
  () => taskState.results.length,
  () => {
    for (const r of taskState.results) applyResult(r)
  },
)

watch(
  () => taskState.summary,
  (s) => {
    if (!s) return
    if (s.cancelled) notify(`任务已取消，成功 ${s.success} / 失败 ${s.failed}`, 'warning')
    else notify(`整理完成：成功 ${s.success} / 失败 ${s.failed}`, 'success')
  },
)

watch(
  () => taskState.lastError,
  (msg) => {
    if (msg) notify(msg, 'error', 6000)
  },
)

function applyResult(r: ProcessResult) {
  const row = rows.value.find(
    (x) => (r.filePath && x.filePath === r.filePath) || x.rawFilename === r.filename,
  )
  if (!row) return
  row.status = r.success ? 'success' : r.skipped ? 'skipped' : 'failed'
  row.directory = r.directory
  row.error = r.error
  row.sid = r.sid
  row.title = r.title
  row.rating = r.rating
  row.posterCount = r.posterCount
}

async function chooseDir() {
  try {
    const picked = await api.selectDirectory(dir.value)
    if (picked) dir.value = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

async function scan() {
  if (!dir.value) {
    notify('请先选择电影目录', 'warning')
    return
  }
  scanning.value = true
  clearLogs()
  pushLocalLog('INFO', `扫描目录：${dir.value}（递归=${recursive.value}）`, 'ui')
  try {
    const files = await api.scanDirectory(dir.value, recursive.value)
    rows.value = files.map<MovieRow>((f) => ({
      ...f,
      status: 'pending',
      directory: '',
      error: '',
      sid: '',
      title: '',
      rating: '',
      posterCount: 0,
    }))
    selected.value = []
    pushLocalLog('INFO', `扫描完成，发现 ${files.length} 个视频文件`, 'ui')
    if (files.length === 0) notify('未发现视频文件，请检查目录与扩展名设置', 'warning')
  } catch (e) {
    reportError(e, '扫描失败')
    pushLocalLog('ERROR', e instanceof Error ? e.message : String(e), 'ui')
  } finally {
    scanning.value = false
  }
}

async function start() {
  if (rows.value.length === 0) {
    notify('请先扫描出视频文件', 'warning')
    return
  }
  const targets = selected.value.length > 0 ? selected.value : rows.value.map((r) => r.filePath)
  for (const row of rows.value) {
    if (targets.includes(row.filePath)) row.status = 'running'
  }
  elapsed.value = 0
  beginTask(targets.length)
  pushLocalLog('INFO', `开始批量整理，共 ${targets.length} 个文件`, 'ui')
  try {
    await api.startBatch(dir.value, recursive.value, targets)
  } catch (e) {
    taskState.running = false
    reportError(e, '启动任务失败')
  }
}

async function cancel() {
  try {
    const ok = await api.cancelTask()
    notify(ok ? '已发送取消请求' : '当前没有运行中的任务', ok ? 'info' : 'warning')
  } catch (e) {
    reportError(e, '取消失败')
  }
}

async function openFolder(path: string) {
  try {
    await api.openFolder(path)
  } catch (e) {
    reportError(e, '打开目录失败')
  }
}
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">批量整理</h1>
        <p class="page-subtitle">扫描目录、抓取豆瓣元数据并生成规范化整理结构</p>
      </div>
      <span class="badge-soft">STEP 01 / 工作台</span>
    </div>

    <section class="glass-card card-pad control-card">
      <div class="control-row">
        <div class="field grow">
          <span class="field-label">电影目录</span>
          <div class="input-group">
            <input v-model="dir" class="input" placeholder="选择存放视频文件的目录…" spellcheck="false" />
            <button class="btn" type="button" :disabled="running" @click="chooseDir">浏览…</button>
          </div>
        </div>
        <div class="switch-wrap">
          <div class="switch" :class="{ 'is-on': recursive }" @click="recursive = !recursive">
            <span class="switch-track" />
            <span class="switch-label">递归扫描</span>
          </div>
        </div>
        <div class="action-group">
          <button class="btn" type="button" :disabled="running" @click="scan">
            <span v-if="scanning" class="spinner" />{{ scanning ? '扫描中' : '扫描' }}
          </button>
          <button class="btn btn-primary" type="button" :disabled="running || rows.length === 0" @click="start">
            <svg viewBox="0 0 24 24" fill="currentColor" class="play-icon"><path d="M8 5v14l11-7z" /></svg>
            开始整理
          </button>
          <button class="btn btn-danger" type="button" :disabled="!taskState.running" @click="cancel">
            取消
          </button>
        </div>
      </div>
      <div class="control-foot">
        <span class="dim">提示：勾选表格行可只整理选中的文件；不勾选则处理全部。</span>
        <span v-if="taskState.progress.current" class="accent ellipsis now">当前：{{ taskState.progress.current }}</span>
      </div>
    </section>

    <div class="metric-grid">
      <div v-for="(m, i) in metrics" :key="m.label" class="glass-card metric-card is-hoverable"
        :class="m.tone" :style="{ animationDelay: `${i * 60}ms` }">
        <div class="metric-label">{{ m.label }}</div>
        <div class="metric-value mono">{{ m.value }}</div>
        <div class="metric-foot">{{ m.foot }}</div>
      </div>
    </div>

    <section class="glass-card card-pad">
      <div class="row-between" style="margin-bottom: 14px">
        <h2 class="section-title">文件列表</h2>
        <span class="dim" style="font-size: 12px">共 {{ rows.length }} 个文件 · 已选 {{ selected.length }} 个</span>
      </div>
      <MovieTable v-model:selected="selected" :rows="rows" :disabled="taskState.running" :max-height="300"
        @open="openFolder" />
    </section>

    <ProgressPanel :progress="taskState.progress" />
  </div>
</template>

<style scoped>
.control-card {
  background: linear-gradient(120deg, rgba(36, 47, 72, 0.75), rgba(20, 27, 45, 0.72));
}
.control-row {
  display: flex;
  align-items: flex-end;
  gap: 18px;
  flex-wrap: wrap;
}
.switch-wrap {
  padding-bottom: 8px;
}
.action-group {
  display: flex;
  gap: 9px;
  padding-bottom: 0;
}
.play-icon {
  width: 15px;
  height: 15px;
}
.control-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--line);
  font-size: 11.5px;
}
.now {
  max-width: 46%;
  font-size: 11.5px;
}
.metric-card {
  animation: fadeInUp 0.45s var(--ease) both;
}
</style>
