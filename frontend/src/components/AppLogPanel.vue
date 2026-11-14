<script setup lang="ts">
import LogConsole from '@/components/LogConsole.vue'
import { taskState } from '@/store/task'
</script>

<template>
  <!-- 参考风格：全宽扁平底部区域，背景与左侧侧边栏同款，无边框、无内缩。 -->
  <section class="app-log-panel">
    <LogConsole :logs="taskState.logs" :height="200" />
  </section>
</template>

<style scoped>
.app-log-panel {
  background: linear-gradient(180deg, rgba(20, 27, 45, 0.96), rgba(11, 15, 26, 0.96));
  border: 0;
}
</style>
