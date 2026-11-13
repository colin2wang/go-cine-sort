<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

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
  <div
    ref="box"
    class="log-console"
    :style="{ height: `${props.height}px` }"
    @scroll="onScroll"
  >
    <div v-if="props.logs.length === 0" class="log-empty">
      暂无日志 · 启动任务后这里会实时输出处理过程
    </div>
    <div v-for="(line, idx) in props.logs" :key="idx" class="log-line">
      <span class="log-time">{{ line.time }}</span>
      <span class="log-level" :class="`log-level-${line.level}`">{{ line.level }}</span>
      <span class="log-msg">{{ line.message }}</span>
    </div>
  </div>
</template>

<style scoped>
.log-console {
  border-radius: var(--radius);
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
