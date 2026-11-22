<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api'
import logoUrl from '@/assets/logo.svg'
import { notify, reportError } from '@/store/toast'
import type { AppConfig } from '@/types'

const props = defineProps<{ section: 'movie' | 'video' | 'youtube' }>()


const SECTION_META: Record<typeof props.section, { title: string; subtitle: string }> = {
  movie: { title: '设置 · 电影', subtitle: '目录、命名模板与整理行为的全局配置' },
  video: { title: '设置 · 视频', subtitle: '视频扫描扩展名与文件名清洗规则' },
  youtube: { title: '设置 · YouTube', subtitle: 'YouTube 归档整理的源目录与目标目录' },
}

const cfg = ref<AppConfig | null>(null)
const saving = ref(false)
const showRules = ref(false)

const boolFlags = computed(() => [
  { key: 'recursive' as const, label: '递归扫描子目录', hint: '关闭后只处理所选目录的第一层' },
  { key: 'preferSourceDir' as const, label: '目录跟随视频文件', hint: '开启后整理目录优先创建在原视频文件所在目录' },
  { key: 'downloadPosters' as const, label: '下载官方海报', hint: '筛选高清官方海报并保存到整理目录' },
  { key: 'fetchImdb' as const, label: '获取 IMDb 评分', hint: '生成 IMDB x.x.txt 评分文件' },
  { key: 'createRatingFile' as const, label: '生成评分文件', hint: '生成 {豆瓣评分}.txt' },
  { key: 'createSidFile' as const, label: '生成 SID 详情文件', hint: '写入影片完整元数据文本' },
])

const preview = computed(() => {
  const fmt = cfg.value?.directoryFormat ?? '{chinese_title}.{english_title}.{year}'
  return fmt
    .replaceAll('{chinese_title}', '肖申克的救赎')
    .replaceAll('{english_title}', 'The.Shawshank.Redemption')
    .replaceAll('{year}', '1994')
    .replace(/\.{2,}/g, '.')
    .replace(/\.+$/, '')
})

const ruleGroups = computed(() => [
  { label: '扩展名', items: cfg.value?.extensions ?? [] },
  { label: '清洗规则', items: cfg.value?.cleanupPatterns ?? [] },
  { label: '编码格式', items: cfg.value?.techPatterns ?? [] },
  { label: '语言标签', items: cfg.value?.languagePatterns ?? [] },
])

onMounted(async () => {
  try {
    cfg.value = await api.getConfig()
  } catch (e) {
    reportError(e, '读取配置失败')
  }
})

async function chooseDir() {
  if (!cfg.value) return
  try {
    const picked = await api.selectDirectory(cfg.value.movieFolder)
    if (picked) cfg.value.movieFolder = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

async function chooseYouTubeSourceDir() {
  if (!cfg.value) return
  try {
    const picked = await api.selectDirectory(cfg.value.youtubeSourceDir)
    if (picked) cfg.value.youtubeSourceDir = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

async function chooseYouTubeTargetDir() {
  if (!cfg.value) return
  try {
    const picked = await api.selectDirectory(cfg.value.youtubeTargetDir)
    if (picked) cfg.value.youtubeTargetDir = picked
  } catch (e) {
    reportError(e, '选择目录失败')
  }
}

async function save() {
  if (!cfg.value) return
  saving.value = true
  try {
    cfg.value = await api.saveConfig(cfg.value)
    notify('配置已保存并立即生效', 'success')
  } catch (e) {
    reportError(e, '保存失败')
  } finally {
    saving.value = false
  }
}

async function restore() {
  try {
    cfg.value = await api.getDefaults()
    notify('已载入内置默认值，点击「保存配置」后生效', 'info', 4200)
  } catch (e) {
    reportError(e, '恢复默认失败')
  }
}

function toggle(key: 'recursive' | 'preferSourceDir' | 'downloadPosters' | 'fetchImdb' | 'createRatingFile' | 'createSidFile' | 'archiveMove') {
  if (!cfg.value) return
  cfg.value = { ...cfg.value, [key]: !cfg.value[key] }
}
</script>

<template>
  <div class="page-body fade-in-up">
    <div class="page-head">
      <div>
        <h1 class="page-title">{{ SECTION_META[props.section].title }}</h1>
        <p class="page-subtitle">{{ SECTION_META[props.section].subtitle }}</p>
      </div>
      <span class="badge-soft">YAML · 用户配置目录</span>
    </div>

    <template v-if="cfg">
      <section class="glass-card card-pad notice">
        <div class="notice-icon">i</div>
        <div>
          <div class="notice-title">配置持久化目录（configs）</div>
          <div class="mono notice-path">{{ cfg.configPath }}</div>
          <div class="field-hint">电影 / 视频 / YouTube 分别存为该目录下的 movie.yml、video.yml、youtube.yml，也可直接编辑；修改后重启应用或在此页保存即可生效。</div>
        </div>
        <button class="btn btn-ghost btn-sm" type="button" @click="restore">恢复默认</button>
      </section>

      <section v-if="props.section === 'movie'" class="glass-card card-pad">
        <h2 class="section-title" style="margin-bottom: 18px">路径与命名</h2>
        <div class="field">
          <span class="field-label">电影目录</span>
          <div class="input-group">
            <input v-model="cfg.movieFolder" class="input" placeholder="选择存放视频文件的目录…" spellcheck="false" />
            <button class="btn" type="button" @click="chooseDir">浏览…</button>
          </div>
          <span class="field-hint">整理目录与归档后的视频文件都会生成在此路径下；留空或目录不存在且无源文件时，回退到程序目录下的 downloads。</span>
        </div>
        <div class="divider" style="margin: 18px 0" />
        <div class="field">
          <span class="field-label">目录名模板</span>
          <input v-model="cfg.directoryFormat" class="input mono" spellcheck="false" />
          <div class="var-chips">
            <span class="var-chip mono">{chinese_title}</span>
            <span class="var-chip mono">{english_title}</span>
            <span class="var-chip mono">{year}</span>
          </div>
          <div class="preview-box">
            <span class="dim" style="font-size: 11px">实时预览</span>
            <div class="preview-name mono">{{ preview || '（模板为空）' }}</div>
          </div>
        </div>
      </section>

      <section v-if="props.section === 'youtube'" class="glass-card card-pad">
        <h2 class="section-title" style="margin-bottom: 18px">YouTube 归档目录</h2>
        <div class="field">
          <span class="field-label">YouTube 归档源目录</span>
          <div class="input-group">
            <input v-model="cfg.youtubeSourceDir" class="input" placeholder="待整理的 YouTube 文件所在目录…" spellcheck="false" />
            <button class="btn" type="button" @click="chooseYouTubeSourceDir">浏览…</button>
          </div>
          <span class="field-hint">「YouTube 归档整理」页默认从这里扫描按 “上传者 - 标题” 命名的文件。</span>
        </div>
        <div class="field" style="margin-top: 14px">
          <span class="field-label">YouTube 归档目标目录</span>
          <div class="input-group">
            <input v-model="cfg.youtubeTargetDir" class="input" placeholder="按上传者分类存放的根目录…" spellcheck="false" />
            <button class="btn" type="button" @click="chooseYouTubeTargetDir">浏览…</button>
          </div>
          <span class="field-hint">文件会被移动到该目录下以上传者命名的子文件夹。</span>
        </div>
      </section>

      <div v-if="props.section === 'movie'" class="grid-2">
        <section class="glass-card card-pad">
          <h2 class="section-title" style="margin-bottom: 6px">整理行为</h2>
          <p class="field-hint" style="margin-bottom: 16px">这些开关同时影响电影批量整理与电影指定整理。</p>
          <div class="flag-list">
            <div v-for="f in boolFlags" :key="f.key" class="flag-row">
              <div>
                <div class="flag-label">{{ f.label }}</div>
                <div class="field-hint">{{ f.hint }}</div>
              </div>
              <div class="switch" :class="{ 'is-on': !!cfg[f.key] }" @click="toggle(f.key)">
                <span class="switch-track" />
              </div>
            </div>
          </div>
          <div class="field" style="margin-top: 18px">
            <span class="field-label">海报图片格式</span>
            <select v-model="cfg.posterFormat" class="select" :disabled="!cfg.downloadPosters">
              <option value="none">不转换（保留原始格式）</option>
              <option value="png">PNG</option>
              <option value="jpg">JPG</option>
            </select>
            <span class="field-hint">下载的海报会自动转换成所选格式保存；JPG 以 90% 质量重编码。</span>
          </div>
        </section>

        <section class="glass-card card-pad">
          <h2 class="section-title" style="margin-bottom: 6px">归档移动</h2>
          <p class="field-hint" style="margin-bottom: 16px">
            开启后，整理成功会把源视频文件移动进生成的目录；同盘直接移动，跨盘使用校验复制。
          </p>
          <div class="flag-row archive-switch">
            <div>
              <div class="flag-label">归档移动视频</div>
              <div class="field-hint">默认关闭，不会改动原视频文件</div>
            </div>
            <div class="switch" :class="{ 'is-on': cfg.archiveMove }" @click="toggle('archiveMove')">
              <span class="switch-track" />
            </div>
          </div>
          <div class="field" style="margin-top: 18px">
            <span class="field-label">同名冲突策略</span>
            <select v-model="cfg.conflictStrategy" class="select" :disabled="!cfg.archiveMove">
              <option value="rename">自动重命名（追加 _1、_2…）</option>
              <option value="skip">跳过，保留源文件</option>
              <option value="overwrite">覆盖目标文件</option>
            </select>
          </div>
          <div class="archive-note" :class="{ active: cfg.archiveMove }">
            <span class="dot" />
            {{ cfg.archiveMove ? '归档移动已开启，请确认目标磁盘空间充足' : '归档移动已关闭，仅生成目录与元数据文件' }}
          </div>
        </section>
      </div>

      <section v-if="props.section === 'video'" class="glass-card card-pad">
        <h2 class="section-title" style="margin-bottom: 12px">扫描规则</h2>
        <div class="ext-chips">
          <span v-for="ext in cfg.extensions" :key="ext" class="ext-chip mono">{{ ext }}</span>
        </div>
        <button class="link-more" type="button" @click="showRules = !showRules">
          {{ showRules ? '收起清洗规则' : `查看 ${cfg.cleanupPatterns.length + cfg.techPatterns.length + cfg.languagePatterns.length} 条清洗规则` }} →
        </button>
        <div v-if="showRules" class="rule-box">
          <div v-for="g in ruleGroups" :key="g.label" class="rule-group">
            <div class="rule-label">{{ g.label }}（{{ g.items.length }}）</div>
            <div v-for="(item, i) in g.items" :key="i" class="rule-item mono">{{ item }}</div>
          </div>
        </div>
      </section>

      <section class="glass-card card-pad about-card">
        <img :src="logoUrl" alt="Cine Sort" class="about-logo" />
        <div class="about-text">
          <div class="about-name">Cine Sort</div>
          <div class="field-hint">电影整理工作台 · Wails 2.17.1 + Vue 3 + TypeScript</div>
        </div>
        <div class="grow" />
        <span class="dim mono" style="font-size: 11px; margin-right: 16px">v1.0.0</span>
        <button class="btn btn-primary" type="button" :disabled="saving" @click="save">
          <span v-if="saving" class="spinner" />{{ saving ? '保存中…' : '保存配置' }}
        </button>
      </section>
    </template>

    <div v-else class="glass-card card-pad empty-state">
      <span class="spinner" /> 正在加载配置…
    </div>
  </div>
</template>

<style scoped>
.notice {
  display: flex;
  align-items: center;
  gap: 14px;
  background: linear-gradient(120deg, rgba(56, 189, 248, 0.09), rgba(20, 27, 45, 0.7));
}
.notice-icon {
  width: 28px;
  height: 28px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--sky);
  border: 1px solid rgba(56, 189, 248, 0.4);
  font-size: 14px;
  font-weight: 600;
}
.notice-title {
  font-size: 12px;
  color: var(--text-2);
}
.notice-path {
  font-size: 11px;
  color: var(--text-1);
  margin: 3px 0;
  word-break: break-all;
  user-select: text;
}
.var-chips {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  flex-wrap: wrap;
}
.var-chip {
  font-size: 11px;
  padding: 4px 9px;
  border-radius: 6px;
  color: var(--sky);
  background: var(--sky-dim);
  border: 1px solid rgba(56, 189, 248, 0.2);
}
.preview-box {
  margin-top: 14px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px dashed rgba(245, 184, 65, 0.35);
  background: rgba(245, 184, 65, 0.05);
}
.preview-name {
  margin-top: 7px;
  font-size: 14px;
  color: var(--gold-1);
  word-break: break-all;
}
.flag-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}
.flag-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.flag-label {
  font-size: 13px;
  margin-bottom: 3px;
}
.archive-note {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-top: 16px;
  padding: 11px 13px;
  border-radius: 9px;
  font-size: 11.5px;
  color: var(--text-2);
  background: rgba(11, 15, 26, 0.5);
  border: 1px solid var(--line);
}
.archive-note .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-3);
  flex: none;
}
.archive-note.active {
  color: var(--gold-1);
  border-color: rgba(245, 184, 65, 0.3);
  background: rgba(245, 184, 65, 0.07);
}
.archive-note.active .dot {
  background: var(--gold-1);
  box-shadow: 0 0 8px var(--gold-1);
}
.ext-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  margin-bottom: 12px;
}
.ext-chip {
  font-size: 11px;
  padding: 4px 9px;
  border-radius: 6px;
  background: rgba(148, 163, 184, 0.09);
  border: 1px solid var(--line);
  color: var(--text-2);
}
.link-more {
  border: none;
  background: transparent;
  padding: 0;
  font-size: 12px;
  font-family: inherit;
  color: var(--sky);
  cursor: pointer;
}
.link-more:hover {
  text-decoration: underline;
}
.rule-box {
  margin-top: 14px;
  max-height: 230px;
  overflow-y: auto;
  padding: 12px;
  border-radius: 9px;
  background: rgba(6, 10, 18, 0.6);
  border: 1px solid var(--line);
}
.rule-group {
  margin-bottom: 12px;
}
.rule-label {
  font-size: 11px;
  color: var(--gold-1);
  margin-bottom: 6px;
}
.rule-item {
  font-size: 10.5px;
  color: var(--text-2);
  line-height: 1.7;
  word-break: break-all;
  user-select: text;
}
.about-card {
  display: flex;
  align-items: center;
  gap: 16px;
}
.about-logo {
  width: 46px;
  height: 46px;
  border-radius: 12px;
  box-shadow: 0 6px 18px rgba(245, 184, 65, 0.2);
}
.about-name {
  font-size: 16px;
  font-weight: 600;
}
</style>
