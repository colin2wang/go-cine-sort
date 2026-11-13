<script setup lang="ts">
import { dismiss, toasts } from '@/store/toast'

const icons: Record<string, string> = {
  success: 'M20 6L9 17l-5-5',
  error: 'M18 6L6 18M6 6l12 12',
  info: 'M12 16v-4M12 8h.01M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20z',
  warning: 'M12 9v4M12 17h.01M10.3 3.9L1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z',
}
</script>

<template>
  <div class="toast-stack">
    <TransitionGroup name="toast">
      <div v-for="t in toasts" :key="t.id" class="toast" :class="`toast-${t.type}`">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round" class="toast-icon">
          <path :d="icons[t.type]" />
        </svg>
        <span class="toast-msg">{{ t.message }}</span>
        <button class="toast-close" type="button" @click="dismiss(t.id)">×</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  top: 78px;
  right: 26px;
  z-index: 999;
  display: flex;
  flex-direction: column;
  gap: 10px;
  pointer-events: none;
}
.toast {
  pointer-events: auto;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 300px;
  padding: 13px 14px;
  border-radius: 12px;
  border: 1px solid var(--line-strong);
  background: rgba(28, 36, 56, 0.96);
  box-shadow: var(--shadow);
  backdrop-filter: blur(16px);
  font-size: 12.5px;
  line-height: 1.55;
  color: var(--text-1);
}
.toast-icon {
  width: 16px;
  height: 16px;
  flex: none;
  margin-top: 1px;
}
.toast-msg {
  flex: 1;
  word-break: break-word;
}
.toast-close {
  border: none;
  background: transparent;
  color: var(--text-3);
  font-size: 17px;
  line-height: 1;
  cursor: pointer;
  padding: 0 2px;
}
.toast-close:hover {
  color: var(--text-1);
}
.toast-success {
  border-color: rgba(34, 197, 94, 0.4);
}
.toast-success .toast-icon {
  color: var(--ok);
}
.toast-error {
  border-color: rgba(239, 68, 68, 0.4);
}
.toast-error .toast-icon {
  color: var(--err);
}
.toast-warning {
  border-color: rgba(245, 158, 11, 0.4);
}
.toast-warning .toast-icon {
  color: var(--warn);
}
.toast-info .toast-icon {
  color: var(--sky);
}
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s var(--ease);
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(24px);
}
</style>
