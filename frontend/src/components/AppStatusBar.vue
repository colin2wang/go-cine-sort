<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { api } from '@/api'
import { taskState } from '@/store/task'

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
.path {
  color: var(--text-3);
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 10px;
}
</style>
