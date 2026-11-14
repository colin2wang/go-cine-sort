<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { clearLogs, LOG_LEVELS, setLogLevel, taskState } from '@/store/task'
import type { LogLevel } from '@/types'

const props = withDefaults(
  defineProps<{
    logs: { time: string; level: string; message: string; source?: string }[]
    height?: number
    autoScroll?: boolean
  }>(),
  { height: 220, autoScroll: true },
)

const box = ref<HTMLElement | null>(null)
const pinned = ref(true)

const levelRank: Record<string, number> = { DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3 }

/** 仅显示级别不低于所选级别的日志。 */
const visible = computed(() => {
  const min = levelRank[taskState.logLevel] ?? 1
  return props.logs.filter((line) => (levelRank[line.level] ?? 1) >= min)
})

function onLevelChange(event: Event) {
  void setLogLevel((event.target as HTMLSelectElement).value as LogLevel)
}

function onScroll() {
  const el = box.value
  if (!el) return
  pinned.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

watch(
  () => props.logs.length,
  async () => {
    if (!props.autoScroll || !pinned.value) return
    await nextTick()
    const el = box.value
    if (el) el.scrollTop = el.scrollHeight
  },
)
</script>

<template>
  <div class="log-console-wrap">
    <div class="log-toolbar">
      <span class="log-toolbar-label">级别</span>
      <select
        class="select log-level-select"
        :value="taskState.logLevel"
        @change="onLevelChange"
      >
        <option v-for="l in LOG_LEVELS" :key="l" :value="l">{{ l }}</option>
      </select>
      <span class="log-toolbar-hint">{{ taskState.logLevel === 'DEBUG' ? 'DEBUG 会输出外部 API 调用的 URL 与参数' : '' }}</span>
      <button class="btn btn-ghost btn-sm log-toolbar-clear" type="button" @click="clearLogs()">
        清空
      </button>
    </div>
    <div
      ref="box"
      class="log-console"
      :style="{ height: `${props.height}px` }"
      @scroll="onScroll"
    >
      <div v-if="props.logs.length === 0" class="log-empty">
        暂无日志 · 启动任务后这里会实时输出处理过程
      </div>
      <div v-else-if="visible.length === 0" class="log-empty">当前级别下暂无日志</div>
      <div v-for="(line, idx) in visible" :key="idx" class="log-line">
        <span class="log-time">{{ line.time }}</span>
        <span class="log-level" :class="`log-level-${line.level}`">{{ line.level }}</span>
        <span class="log-msg">{{ line.message }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.log-console-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.log-toolbar {
  align-items: center;
  display: flex;
  gap: 8px;
  padding: 6px 14px;
}
.log-toolbar-label {
  color: var(--text-3);
  font-size: 12px;
}
.log-level-select {
  height: 24px;
  padding: 0 8px;
  width: 88px;
  font-size: 12px;
}
.log-toolbar-hint {
  color: var(--text-3);
  font-size: 11px;
}
.log-toolbar-clear {
  margin-left: auto;
}
.log-empty {
  color: var(--text-3);
  font-family: inherit;
  padding: 12px 4px;
  text-align: center;
}
.log-line {
  align-items: flex-start;
}
</style>
