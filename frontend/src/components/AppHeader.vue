<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { taskState } from '@/store/task'

const route = useRoute()
const title = computed(() => (route.meta.title as string) || 'Cine Sort')
const subtitle = computed(() => (route.meta.subtitle as string) || '')
</script>

<template>
  <header class="header">
    <div class="header-titles">
      <h1 class="header-title">{{ title }}</h1>
      <span class="header-subtitle ellipsis">{{ subtitle }}</span>
    </div>
    <div class="header-right">
      <div class="status-pill" :class="taskState.running ? 'is-running' : 'is-idle'">
        <span class="dot" />
        {{ taskState.running ? '任务执行中' : '空闲' }}
      </div>
      <div class="header-glow" />
    </div>
  </header>
</template>

<style scoped>
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  height: 100%;
  padding: 0 30px;
  border-bottom: 1px solid var(--line);
  background: linear-gradient(90deg, rgba(20, 27, 45, 0.5), rgba(11, 15, 26, 0.2));
  backdrop-filter: blur(14px);
  position: relative;
  overflow: hidden;
}
.header-titles {
  display: flex;
  align-items: baseline;
  gap: 14px;
  min-width: 0;
}
.header-title {
  font-size: 17px;
  font-weight: 600;
  margin: 0;
  letter-spacing: 0.3px;
  white-space: nowrap;
}
.header-subtitle {
  font-size: 12px;
  color: var(--text-3);
  max-width: 460px;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
}
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 28px;
  padding: 0 12px;
  border-radius: 20px;
  font-size: 11px;
  border: 1px solid var(--line);
  white-space: nowrap;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.is-idle {
  color: var(--text-2);
  background: rgba(148, 163, 184, 0.07);
}
.is-running {
  color: var(--gold-1);
  background: rgba(245, 184, 65, 0.1);
  border-color: rgba(245, 184, 65, 0.3);
}
.is-running .dot {
  animation: pulse 1.3s ease-in-out infinite;
}
.header-glow {
  width: 90px;
  height: 2px;
  border-radius: 2px;
  background: linear-gradient(90deg, transparent, var(--gold-1), transparent);
  opacity: 0.55;
}
</style>
