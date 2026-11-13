<script setup lang="ts">
import { computed } from 'vue'
import type { ItemStatus, MovieRow } from '@/types'

const props = withDefaults(
  defineProps<{
    rows: MovieRow[]
    selected: string[]
    selectable?: boolean
    disabled?: boolean
    maxHeight?: number
  }>(),
  { selectable: true, disabled: false, maxHeight: 320 },
)

const emit = defineEmits<{
  'update:selected': [string[]]
  open: [string]
}>()

const labels: Record<ItemStatus, string> = {
  pending: '待处理',
  running: '处理中',
  success: '成功',
  failed: '失败',
  skipped: '跳过',
}

const allSelected = computed(
  () => props.rows.length > 0 && props.rows.every((r) => props.selected.includes(r.filePath)),
)

function toggleAll() {
  if (allSelected.value) {
    emit('update:selected', [])
  } else {
    emit('update:selected', props.rows.map((r) => r.filePath))
  }
}

function toggleOne(path: string) {
  const next = props.selected.includes(path)
    ? props.selected.filter((p) => p !== path)
    : [...props.selected, path]
  emit('update:selected', next)
}
</script>

<template>
  <div class="table-wrap" :style="{ maxHeight: `${props.maxHeight}px` }">
    <table class="data-table">
      <thead>
        <tr>
          <th v-if="props.selectable" style="width: 42px">
            <input type="checkbox" class="cbx" :checked="allSelected" :disabled="props.disabled"
              @change="toggleAll" />
          </th>
          <th style="min-width: 240px">文件名</th>
          <th style="min-width: 150px">清洗片名</th>
          <th style="width: 70px">年份</th>
          <th style="width: 90px">状态</th>
          <th style="min-width: 220px">目标目录 / 说明</th>
          <th style="width: 72px">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in props.rows" :key="row.filePath">
          <td v-if="props.selectable">
            <input type="checkbox" class="cbx" :checked="props.selected.includes(row.filePath)"
              :disabled="props.disabled" @change="toggleOne(row.filePath)" />
          </td>
          <td>
            <div class="ellipsis filename" :title="row.rawFilename">{{ row.rawFilename }}</div>
          </td>
          <td class="cell-dim ellipsis" :title="row.movieName">{{ row.movieName || '—' }}</td>
          <td class="mono cell-dim">{{ row.year || '—' }}</td>
          <td><span class="tag" :class="`tag-${row.status}`">{{ labels[row.status] }}</span></td>
          <td>
            <div class="ellipsis cell-dim" :title="row.error || row.directory">
              {{ row.directory || row.error || '—' }}
            </div>
          </td>
          <td>
            <button v-if="row.directory" class="link-btn" type="button" @click="emit('open', row.directory)">
              打开
            </button>
            <span v-else class="dim">—</span>
          </td>
        </tr>
        <tr v-if="props.rows.length === 0">
          <td :colspan="props.selectable ? 7 : 6">
            <div class="empty-state">
              <div class="empty-icon">▤</div>
              <div>暂无视频文件 · 选择目录后点击「扫描」</div>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.cbx {
  width: 14px;
  height: 14px;
  accent-color: var(--gold-1);
  cursor: pointer;
}
.cbx:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.filename {
  max-width: 320px;
  color: var(--text-1);
}
.link-btn {
  border: none;
  background: transparent;
  color: var(--sky);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  padding: 2px 0;
}
.link-btn:hover {
  color: var(--gold-1);
  text-decoration: underline;
}
</style>
