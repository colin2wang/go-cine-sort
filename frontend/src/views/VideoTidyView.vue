<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api'
import { beginTask, installTaskEvents, pushLocalLog, taskState } from '@/store/task'
import { notify, reportError } from '@/store/toast'
import type { MediaFileMeta } from '@/types'

const dir = ref('')
const recursive = ref(true)
const scanning = ref(false)
const files = ref<MediaFileMeta[]>([])

/** 各文件正在编辑的标签值，按 path 索引。 */
const editing = ref<Record<string, { title: string; comment: string; saving: boolean }>>({})

const renamableCount = computed(() => files.value.filter((f) => f.renamable).length)
const taggableCount = computed(() => files.value.filter((f) => f.taggable).length)

onMounted(async () => {
  installTaskEvents()
})

async function chooseDir() {
  try {
    const picked = await api.selectDirectory(dir.value)
    if (picked) dir.value = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

const scanInvalid = computed(() => !dir.value.trim())

async function doScan() {
  if (scanInvalid.value) return
  scanning.value = true
  files.value = []
  editing.value = {}
  try {
    files.value = await api.scanMediaFiles(dir.value, recursive.value)
    for (const f of files.value) {
      editing.value[f.path] = {
        title: f.title,
        comment: f.comment,
        saving: false,
      }
    }
    if (files.value.length === 0) {
      notify('目录下没有视频文件', 'info')
    }
  } catch (e) {
    reportError(e, '扫描目录失败')
  } finally {
    scanning.value = false
  }
}

function editOf(path: string) {
  return (
    editing.value[path] ?? (editing.value[path] = { title: '', comment: '', saving: false })
  )
}

async function saveTags(file: MediaFileMeta) {
  const edit = editOf(file.path)
  if (!edit.title.trim() && !edit.comment.trim()) {
    notify('标题与备注不能同时为空', 'warning')
    return
  }
  edit.saving = true
  try {
    await api.updateMediaTags(file.path, edit.title, edit.comment)
    file.title = edit.title
    file.comment = edit.comment
    pushLocalLog('INFO', `标签已更新：${file.filename}`)
    notify('标签保存成功', 'success')
  } catch (e) {
    reportError(e, `保存标签失败：${file.filename}`)
  } finally {
    edit.saving = false
  }
}

async function startRename() {
  if (scanInvalid.value || renamableCount.value === 0 || taskState.running) return
  try {
    beginTask(renamableCount.value)
    await api.startMediaRename(dir.value, recursive.value)
    pushLocalLog('INFO', `批量重命名任务已启动，共 ${renamableCount.value} 个文件`)
  } catch (e) {
    reportError(e, '启动重命名失败')
  }
}

async function refreshAfterRename() {
  const summary = taskState.summary
  if (summary && !summary.cancelled) {
    await doScan()
  }
}
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">视频整理</h1>
        <p class="page-subtitle">为视频文件添加 / 更新标签（标题、备注），并按 “标题 (YY.MM.DD)” 规范批量重命名</p>
      </div>
      <span class="badge-soft">MP4 标签 · 重命名</span>
    </div>

    <section class="glass-card card-pad">
      <div class="dir-row">
        <div class="field">
          <span class="field-label">视频目录</span>
          <div class="input-group">
            <input v-model="dir" class="input" placeholder="视频文件所在目录…" spellcheck="false" />
            <button class="btn" type="button" @click="chooseDir">浏览…</button>
          </div>
        </div>
        <label class="checkbox-inline">
          <input v-model="recursive" type="checkbox" /> 包含子目录
        </label>
      </div>
      <div class="plan-row">
        <button class="btn btn-primary" type="button" :disabled="scanning || scanInvalid" @click="doScan">
          <span v-if="scanning" class="spinner" />{{ scanning ? '扫描中…' : '扫描文件' }}
        </button>
        <span class="field-hint">扫描后可编辑标签并预览重命名结果。</span>
      </div>
    </section>

    <section v-if="files.length" class="glass-card card-pad">
      <div class="section-head">
        <h2 class="section-title">文件列表</h2>
        <span class="stat-chips">
          <span class="stat-chip">共 {{ files.length }} 个</span>
          <span class="stat-chip dim">可编辑标签 {{ taggableCount }}（MP4）</span>
          <span class="stat-chip warn">待重命名 {{ renamableCount }}</span>
        </span>
      </div>

      <div class="table-wrap">
        <table class="plan-table">
          <thead>
            <tr>
              <th>文件</th>
              <th>标签（标题 / 备注）</th>
              <th>新文件名</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="f in files" :key="f.path">
              <td class="mono cell-file" :title="f.path">{{ f.filename }}</td>
              <td>
                <div v-if="f.taggable" class="tag-editor">
                  <input v-model="editOf(f.path).title" class="input input-sm" placeholder="标题…" spellcheck="false" />
                  <input v-model="editOf(f.path).comment" class="input input-sm" placeholder="备注…" spellcheck="false" />
                  <button
                    class="btn btn-sm"
                    type="button"
                    :disabled="editOf(f.path).saving || taskState.running"
                    @click="saveTags(f)"
                  >
                    <span v-if="editOf(f.path).saving" class="spinner" />保存
                  </button>
                </div>
                <span v-else class="field-hint">不支持标签编辑</span>
              </td>
              <td class="mono" :class="{ rename: f.renamable }">
                {{ f.renamable ? f.newName : '—' }}
              </td>
              <td>{{ f.renamable ? '待重命名' : '保持不变' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="start-row">
        <button
          class="btn btn-primary"
          type="button"
          :disabled="renamableCount === 0 || taskState.running"
          @click="startRename"
        >
          {{ taskState.running ? '任务进行中…' : `批量重命名（${renamableCount} 个文件）` }}
        </button>
        <button v-if="taskState.running" class="btn" type="button" @click="api.cancelTask()">取消</button>
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
          <button class="btn btn-sm" type="button" @click="refreshAfterRename">刷新列表</button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dir-row {
  display: flex;
  align-items: flex-end;
  gap: 16px;
}
.dir-row .field {
  flex: 1;
}
.checkbox-inline {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-1);
  cursor: pointer;
  white-space: nowrap;
  padding-bottom: 8px;
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
  flex-wrap: wrap;
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
  max-height: 420px;
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
  vertical-align: middle;
}
.plan-table th {
  position: sticky;
  top: 0;
  background: rgba(11, 15, 26, 0.95);
  color: var(--text-2);
  font-weight: 500;
}
.cell-file {
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.plan-table td.rename {
  color: var(--gold-1);
}
.tag-editor {
  display: flex;
  gap: 6px;
  align-items: center;
}
.tag-editor .input-sm {
  height: 26px;
  font-size: 11.5px;
  padding: 0 8px;
}
.tag-editor .input-sm:first-child {
  width: 160px;
}
.tag-editor .input-sm:nth-child(2) {
  width: 160px;
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
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>
