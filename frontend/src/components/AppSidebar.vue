<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import logoUrl from '@/assets/logo.svg'

const props = defineProps<{ collapsed: boolean }>()
const emit = defineEmits<{ toggle: [] }>()
const route = useRoute()

const groups = [
  {
    label: '电影',
    items: [
      {
        path: '/workbench',
        label: '批量整理',
        icon: 'M3 3h7v7H3zM14 3h7v7h-7zM14 14h7v7h-7zM3 14h7v7H3z',
      },
      {
        path: '/single',
        label: '指定整理',
        icon: 'M3 4h18v16H3zM7 4v16M17 4v16M3 9h4M3 15h4M17 9h4M17 15h4',
      },
      {
        path: '/history',
        label: '历史记录',
        icon: 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zM12 6v6l4 2',
      },
      {
        path: '/cache',
        label: '响应缓存',
        icon: 'M4 6c0-1.7 3.6-3 8-3s8 1.3 8 3-3.6 3-8 3-8-1.3-8-3zM4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3',
      },
    ],
  },
  {
    label: '视频',
    items: [
      {
        path: '/tidy',
        label: '批量整理',
        icon: 'M4 4h16v16H4zM8 4v16M16 4v16M4 9h4M4 15h4M16 9h4M16 15h4',
      },
    ],
  },
  {
    label: 'YouTube',
    items: [
      {
        path: '/youtube',
        label: '归档整理',
        icon: 'M10 8.5v7l6-3.5-6-3.5zM12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20z',
      },
    ],
  },
  {
    label: '设置',
    items: [
      {
        path: '/settings/movie',
        label: '电影',
        icon: 'M4 7h16M4 12h16M4 17h16M9 5v4M15 10v4M8 15v4',
      },
      {
        path: '/settings/video',
        label: '视频',
        icon: 'M4 7h16M4 12h16M4 17h16M9 5v4M15 10v4M8 15v4',
      },
      {
        path: '/settings/youtube',
        label: 'YouTube',
        icon: 'M4 7h16M4 12h16M4 17h16M9 5v4M15 10v4M8 15v4',
      },
    ],
  },
]

const activePath = computed(() => route.path)
</script>

<template>
  <aside class="sidebar" :class="{ collapsed: props.collapsed }">
    <div class="brand">
      <img :src="logoUrl" alt="Cine Sort" class="brand-logo" />
      <div v-if="!props.collapsed" class="brand-text">
        <span class="brand-name">Cine Sort</span>
        <span class="brand-sub">电影整理工作台</span>
      </div>
    </div>

    <nav class="nav">
      <div v-for="group in groups" :key="group.label" class="nav-group">
        <div v-if="!props.collapsed" class="nav-caption">{{ group.label }}</div>
        <RouterLink
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: activePath === item.path, child: group.label === '设置' }"
          :title="item.label"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"
            stroke-linecap="round" stroke-linejoin="round" class="nav-icon">
            <path :d="item.icon" />
          </svg>
          <span v-if="!props.collapsed" class="nav-label">{{ item.label }}</span>
          <span v-if="activePath === item.path" class="nav-marker" />
        </RouterLink>
      </div>
    </nav>

    <div class="sidebar-foot">
      <button class="collapse-btn" type="button" :title="props.collapsed ? '展开侧栏' : '收起侧栏'"
        @click="emit('toggle')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"
          stroke-linecap="round" stroke-linejoin="round">
          <path v-if="props.collapsed" d="M9 18l6-6-6-6" />
          <path v-else d="M15 18l-6-6 6-6" />
        </svg>
        <span v-if="!props.collapsed">收起侧栏</span>
      </button>
      <div v-if="!props.collapsed" class="version">v1.0.0 · Wails 2.17.1</div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 18px 12px 14px;
  background: linear-gradient(180deg, rgba(20, 27, 45, 0.96), rgba(11, 15, 26, 0.96));
  border-right: 1px solid var(--line);
  backdrop-filter: blur(20px);
  overflow: hidden;
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 8px 22px;
  min-height: 62px;
}
.brand-logo {
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: 11px;
  box-shadow: 0 6px 18px rgba(245, 184, 65, 0.2);
  animation: fadeIn 0.5s var(--ease);
}
.brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  white-space: nowrap;
}
.brand-name {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.3px;
}
.brand-sub {
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.6px;
}
.nav {
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-right: 2px;
}
.nav-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 12px;
}
.nav-group:last-child {
  margin-bottom: 0;
}
.nav-caption {
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 1.4px;
  padding: 8px 12px 6px;
  text-transform: uppercase;
}
.nav-item.child {
  height: 36px;
  font-size: 12.5px;
  padding-left: 14px;
}
.nav-item.child .nav-icon {
  width: 16px;
  height: 16px;
}
.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  height: 42px;
  padding: 0 12px;
  border-radius: 10px;
  color: var(--text-2);
  text-decoration: none;
  font-size: 13px;
  white-space: nowrap;
  transition: background 0.22s var(--ease), color 0.22s, transform 0.22s var(--ease);
}
.nav-item:hover {
  background: rgba(148, 163, 184, 0.08);
  color: var(--text-1);
  transform: translateX(2px);
}
.nav-item.active {
  color: var(--gold-1);
  background: linear-gradient(90deg, rgba(245, 184, 65, 0.16), rgba(245, 184, 65, 0.03));
}
.nav-icon {
  width: 19px;
  height: 19px;
  flex: none;
}
.nav-label {
  flex: 1;
}
.nav-marker {
  position: absolute;
  right: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 3px;
  background: var(--gold-1);
  box-shadow: 0 0 12px var(--gold-1);
}
.collapsed .nav-item {
  justify-content: center;
  padding: 0;
}
.sidebar-foot {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 8px 0;
  border-top: 1px solid var(--line);
}
.collapse-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 34px;
  padding: 0 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-3);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.2s, background 0.2s;
}
.collapse-btn:hover {
  color: var(--text-1);
  background: rgba(148, 163, 184, 0.08);
}
.collapse-btn svg {
  width: 17px;
  height: 17px;
  flex: none;
}
.collapsed .collapse-btn {
  justify-content: center;
  padding: 0;
}
.version {
  font-size: 10px;
  color: var(--text-3);
  padding: 0 10px;
  letter-spacing: 0.3px;
}
</style>
