<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api'
import { beginTask, installTaskEvents, pushLocalLog, taskState } from '@/store/task'
import { notify, reportError } from '@/store/toast'
import type { YouTubeConflictAction, YouTubePlanResult, YouTubeMovePlan } from '@/types'

const sourceDir = ref('')
const targetDir = ref('')
const conflictAction = ref<YouTubeConflictAction>('skip')
const plan = ref<YouTubePlanResult | null>(null)
const planning = ref(false)
const starting = ref(false)

const moves = computed<YouTubeMovePlan[]>(() => plan.value?.moves ?? [])
const conflicts = computed<string[]>(() => plan.value?.conflicts ?? [])
const unmatched = computed<string[]>(() => plan.value?.unmatched ?? [])

onMounted(async () => {
  installTaskEvents()
  try {
    const cfg = await api.getConfig()
    sourceDir.value = cfg.youtubeSourceDir ?? ''
    targetDir.value = cfg.youtubeTargetDir ?? ''
  } catch (e) {
    reportError(e, '读取配置失败')
  }
})

async function chooseSource() {
  try {
    const picked = await api.selectDirectory(sourceDir.value)
    if (picked) sourceDir.value = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

async function chooseTarget() {
  try {
    const picked = await api.selectDirectory(targetDir.value)
    if (picked) targetDir.value = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

const planInvalid = computed(() => !sourceDir.value.trim() || !targetDir.value.trim())

async function doPlan() {
  if (planInvalid.value) return
  planning.value = true
  plan.value = null
  try {
    plan.value = await api.planYouTubeSort(sourceDir.value, targetDir.value)
    if (moves.value.length === 0) {
      notify('未发现符合 “上传者 - 标题” 命名规则的文件', 'info')
    }
  } catch (e) {
    reportError(e, '扫描目录失败')
  } finally {
    planning.value = false
  }
}

async function start() {
  if (planInvalid.value || moves.value.length === 0 || taskState.running) return
  starting.value = true
  try {
    beginTask(moves.value.length)
    await api.startYouTubeSort(sourceDir.value, targetDir.value, conflictAction.value)
    pushLocalLog('INFO', `YouTube 归档任务已启动，共 ${moves.value.length} 个文件`)
  } catch (e) {
    reportError(e, '启动归档失败')
  } finally {
    starting.value = false
  }
}
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">YouTube 归档</h1>
        <p class="page-subtitle">按 “上传者 - 标题” 命名规则将文件移动到以上传者命名的子目录</p>
      </div>
      <span class="badge-soft">yt-dlp 命名</span>
    </div>

    <section class="glass-card card-pad">
      <div class="dir-grid">
        <div class="field">
          <span class="field-label">源目录</span>
          <div class="input-group">
            <input v-model="sourceDir" class="input" placeholder="待整理文件所在目录…" spellcheck="false" />
            <button class="btn" type="button" @click="chooseSource">浏览…</button>
          </div>
        </div>
        <div class="field">
          <span class="field-label">目标目录</span>
          <div class="input-group">
            <input v-model="targetDir" class="input" placeholder="按上传者分类的根目录…" spellcheck="false" />
            <button class="btn" type="button" @click="chooseTarget">浏览…</button>
          </div>
        </div>
      </div>
      <div class="plan-row">
        <button class="btn btn-primary" type="button" :disabled="planning || planInvalid" @click="doPlan">
          <span v-if="planning" class="spinner" />{{ planning ? '扫描中…' : '扫描预览' }}
        </button>
        <span class="field-hint">默认目录来自「设置」页，可在此临时修改。</span>
      </div>
    </section>

    <section v-if="plan" class="glass-card card-pad">
      <div class="section-head">
        <h2 class="section-title">预览结果</h2>
        <span class="stat-chips">
          <span class="stat-chip">待移动 {{ moves.length }}</span>
          <span class="stat-chip warn">目标已存在 {{ conflicts.length }}</span>
          <span class="stat-chip dim">不匹配 {{ unmatched.length }}</span>
        </span>
      </div>

      <div v-if="moves.length" class="table-wrap">
        <table class="plan-table">
          <thead>
            <tr><th>文件</th><th>上传者</th><th>目标子目录</th><th class="th-warn">冲突</th></tr>
          </thead>
          <tbody>
            <tr v-for="m in moves" :key="m.filename">
              <td class="mono cell-file">{{ m.filename }}</td>
              <td>{{ m.uploader }}</td>
              <td class="mono">{{ m.uploader }}/</td>
              <td :class="{ warn: conflicts.includes(m.filename) }">
                {{ conflicts.includes(m.filename) ? '已存在' : '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="conflicts.length" class="conflict-box">
        <div class="conflict-label">目标目录下已存在同名文件（{{ conflicts.length }}）</div>
        <div class="conflict-actions">
          <label class="radio"><input v-model="conflictAction" type="radio" value="skip" /> 跳过，保留已存在文件</label>
          <label class="radio"><input v-model="conflictAction" type="radio" value="overwrite" /> 覆盖目标文件</label>
        </div>
        <div class="mono conflict-list">{{ conflicts.join('、') }}</div>
      </div>

      <details v-if="unmatched.length" class="unmatched">
        <summary>不符合命名规则的文件（{{ unmatched.length }}）</summary>
        <div class="mono unmatched-list">{{ unmatched.join('\n') }}</div>
      </details>

      <div class="start-row">
        <button class="btn btn-primary" type="button" :disabled="starting || moves.length === 0 || taskState.running" @click="start">
          <span v-if="starting" class="spinner" />
          {{ taskState.running ? '任务进行中…' : `开始归档（${moves.length} 个文件）` }}
        </button>
        <button v-if="taskState.running" class="btn" type="button" @click="api.cancelTask()">
          取消
        </button>
      </div>

      <div v-if="taskState.running || taskState.summary" class="progress-wrap">
        <div class="progress-meta">
          <span>{{ taskState.progress.stage }}</span>
          <span class="mono">{{ taskState.progress.completed }}/{{ taskState.progress.total }}</span>
        </div>
        <div class="progress-track">
          <div class="progress-fill" :style="{ width: `${taskState.progress.percent}%` }" />
        </div>
        <div v-if="taskState.summary" class="progress-summary">
          完成：成功 {{ taskState.summary.success }} / 失败 {{ taskState.summary.failed }} / 总计 {{ taskState.summary.total }}
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dir-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.plan-row {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 16px;
}
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.stat-chips {
  display: flex;
  gap: 8px;
}
.stat-chip {
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 6px;
  background: rgba(56, 189, 248, 0.12);
  color: var(--sky);
  border: 1px solid rgba(56, 189, 248, 0.25);
}
.stat-chip.warn {
  background: rgba(245, 184, 65, 0.12);
  color: var(--gold-1);
  border-color: rgba(245, 184, 65, 0.3);
}
.stat-chip.dim {
  background: rgba(148, 163, 184, 0.09);
  color: var(--text-2);
  border-color: var(--line);
}
.table-wrap {
  max-height: 340px;
  overflow-y: auto;
  border: 1px solid var(--line);
  border-radius: 10px;
}
.plan-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.plan-table th,
.plan-table td {
  text-align: left;
  padding: 8px 12px;
  border-bottom: 1px solid var(--line);
}
.plan-table th {
  position: sticky;
  top: 0;
  background: rgba(11, 15, 26, 0.95);
  color: var(--text-2);
  font-weight: 500;
}
.plan-table .th-warn {
  color: var(--gold-1);
}
.cell-file {
  max-width: 340px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.plan-table td.warn {
  color: var(--gold-1);
}
.conflict-box {
  margin-top: 16px;
  padding: 13px 15px;
  border-radius: 10px;
  border: 1px solid rgba(245, 184, 65, 0.35);
  background: rgba(245, 184, 65, 0.06);
}
.conflict-label {
  font-size: 12px;
  color: var(--gold-1);
  margin-bottom: 8px;
}
.conflict-actions {
  display: flex;
  gap: 20px;
  margin-bottom: 8px;
}
.radio {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-1);
  cursor: pointer;
}
.conflict-list {
  font-size: 10.5px;
  color: var(--text-2);
  line-height: 1.7;
  word-break: break-all;
  user-select: text;
}
.unmatched {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-2);
}
.unmatched summary {
  cursor: pointer;
}
.unmatched-list {
  margin-top: 8px;
  padding: 10px 12px;
  max-height: 180px;
  overflow-y: auto;
  border-radius: 9px;
  background: rgba(6, 10, 18, 0.6);
  border: 1px solid var(--line);
  font-size: 10.5px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  user-select: text;
}
.start-row {
  display: flex;
  gap: 12px;
  margin-top: 18px;
}
.progress-wrap {
  margin-top: 18px;
}
.progress-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11.5px;
  color: var(--text-2);
  margin-bottom: 6px;
}
.progress-track {
  height: 6px;
  border-radius: 6px;
  background: rgba(148, 163, 184, 0.15);
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  border-radius: 6px;
  background: linear-gradient(90deg, var(--gold-1), var(--sky));
  transition: width 0.3s var(--ease);
}
.progress-summary {
  margin-top: 8px;
  font-size: 11.5px;
  color: var(--text-2);
}
</style>
