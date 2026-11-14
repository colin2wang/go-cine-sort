<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { api } from '@/api'
import { logPanelCollapsed, taskState } from '@/store/task'

const configPath = ref('')
const ready = ref(false)
onMounted(async () => {
  try {
    const cfg = await api.getConfig()
    configPath.value = cfg.configPath
    ready.value = true
  } catch {
    ready.value = false
  }
})
const stage = computed(() => taskState.progress.stage)
</script>

<template>
  <footer class="status">
    <div class="left">
      <button
        class="log-toggle"
        type="button"
        :title="logPanelCollapsed ? '展开日志面板' : '折叠日志面板'"
        @click="logPanelCollapsed = !logPanelCollapsed"
      >
        <span class="log-toggle-label">日志</span>
        <span v-if="taskState.logs.length" class="log-toggle-count">{{ taskState.logs.length }}</span>
        <span class="log-toggle-arrow" :class="{ up: !logPanelCollapsed }">▴</span>
      </button>
      <span class="sep">/</span>
      <span class="conn" :class="ready ? 'ok' : 'bad'" />
      <span>{{ ready ? '后端已连接' : '后端未就绪' }}</span>
      <span class="sep">/</span>
      <span class="dim">{{ stage }}</span>
    </div>
    <div class="right ellipsis" :title="configPath">
      <span class="dim">配置</span>
      <span class="path">{{ configPath || '加载中…' }}</span>
    </div>
  </footer>
</template>

<style scoped>
.status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  height: 100%;
  padding: 0 20px;
  font-size: 11px;
  color: var(--text-2);
  border-top: 1px solid var(--line);
  background: rgba(11, 15, 26, 0.6);
}
.left,
.right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.conn {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex: none;
}
.conn.ok {
  background: var(--ok);
  box-shadow: 0 0 8px rgba(34, 197, 94, 0.6);
}
.conn.bad {
  background: var(--err);
}
.sep {
  color: var(--text-3);
  opacity: 0.5;
}
.log-toggle {
  align-items: center;
  background: transparent;
  border: 0;
  color: var(--text-2);
  cursor: pointer;
  display: flex;
  font-size: 11px;
  gap: 6px;
  padding: 2px 6px;
}
.log-toggle:hover {
  color: var(--text-1);
}
.log-toggle-count {
  background: rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  font-size: 10px;
  padding: 0 5px;
}
.log-toggle-arrow {
  font-size: 10px;
  opacity: 0.7;
  transition: transform 0.2s var(--ease);
}
.log-toggle-arrow.up {
  transform: rotate(180deg);
}
.path {
  color: var(--text-3);
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 10px;
}
</style>
