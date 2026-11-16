<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppSidebar from '@/components/AppSidebar.vue'
import AppHeader from '@/components/AppHeader.vue'
import AppLogPanel from '@/components/AppLogPanel.vue'
import AppStatusBar from '@/components/AppStatusBar.vue'
import AppToast from '@/components/AppToast.vue'
import { installTaskEvents } from '@/store/task'
import { api } from '@/api'

const collapsed = ref(false)

onMounted(() => {
  installTaskEvents()
  void api.ping().catch(() => {})
})
</script>

<template>
  <div class="app-shell" :class="{ 'is-collapsed': collapsed }">
    <div class="app-sidebar">
      <AppSidebar :collapsed="collapsed" @toggle="collapsed = !collapsed" />
    </div>
    <div class="app-header">
      <AppHeader />
    </div>
    <main class="app-main">
      <RouterView v-slot="{ Component }">
        <Transition name="route-fade" mode="out-in">
          <KeepAlive>
            <component :is="Component" />
          </KeepAlive>
        </Transition>
      </RouterView>
    </main>
    <div class="app-logs">
      <AppLogPanel />
    </div>
    <div class="app-status">
      <AppStatusBar />
    </div>
    <AppToast />
  </div>
</template>
