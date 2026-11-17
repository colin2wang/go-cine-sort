import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/workbench' },
  {
    path: '/workbench',
    name: 'workbench',
    component: () => import('@/views/WorkbenchView.vue'),
    meta: { title: '批量整理', subtitle: '扫描目录、抓取元数据并生成整理结构' },
  },
  {
    path: '/single',
    name: 'single',
    component: () => import('@/views/SingleView.vue'),
    meta: { title: '单部处理', subtitle: '输入豆瓣 SID 直接整理一部影片' },
  },
  {
    path: '/history',
    name: 'history',
    component: () => import('@/views/HistoryView.vue'),
    meta: { title: '历史记录', subtitle: '查看已整理影片与海报归档' },
  },
  {
    path: '/youtube',
    name: 'youtube',
    component: () => import('@/views/YouTubeView.vue'),
    meta: { title: 'YouTube 归档', subtitle: '按上传者将 YouTube 视频文件移动到分类目录' },
  },
  {
    path: '/cache',
    name: 'cache',
    component: () => import('@/views/CacheView.vue'),
    meta: { title: '响应缓存', subtitle: '检查外部 API 与页面请求的原始响应' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: { title: '设置', subtitle: '目录、命名模板与各项处理开关' },
  },
  { path: '/:pathMatch(.*)*', redirect: '/workbench' },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
