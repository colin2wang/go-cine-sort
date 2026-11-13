<script setup lang="ts">
import { computed } from 'vue'
import type { TaskProgress } from '@/types'

const props = defineProps<{ progress: TaskProgress }>()

const percent = computed(() => Math.min(100, Math.max(0, props.progress.percent || 0)))
const counters = computed(() => [
  { label: '总计', value: props.progress.total, tone: '' },
  { label: '已完成', value: props.progress.completed, tone: 'text-sky' },
  { label: '成功', value: props.progress.success, tone: 'text-ok' },
  { label: '失败', value: props.progress.failed, tone: 'text-err' },
])
</script>

<template>
  <section class="glass-card card-pad progress-card fade-in-up">
    <div class="row-between head">
      <h2 class="section-title">处理进度</h2>
      <div class="stage-chip">
        <span class="spinner" v-if="props.progress.stage !== '已完成' && props.progress.stage !== '空闲'" />
        {{ props.progress.stage }}
      </div>
    </div>

    <div class="bar-row">
      <div class="progress-track">
        <div class="progress-fill" :style="{ width: `${percent}%` }" />
      </div>
      <span class="pct mono">{{ percent.toFixed(0) }}%</span>
    </div>

    <div class="current ellipsis" :title="props.progress.current">
      {{ props.progress.current || '等待任务开始…' }}
    </div>

    <div class="counters">
      <div v-for="c in counters" :key="c.label" class="counter">
        <span class="counter-label">{{ c.label }}</span>
        <span class="counter-value mono" :class="c.tone">{{ c.value }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.progress-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.head {
  margin-bottom: 2px;
}
.bar-row {
  display: flex;
  align-items: center;
  gap: 14px;
}
.pct {
  width: 48px;
  text-align: right;
  font-size: 13px;
  color: var(--gold-1);
  flex: none;
}
.current {
  font-size: 12px;
  color: var(--text-2);
  min-height: 18px;
}
.counters {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid var(--line);
}
.counter {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.counter-label {
  font-size: 11px;
  color: var(--text-3);
}
.counter-value {
  font-size: 19px;
  font-weight: 600;
}
.text-sky {
  color: var(--sky);
}
.text-ok {
  color: var(--ok);
}
.text-err {
  color: var(--err);
}
</style>
