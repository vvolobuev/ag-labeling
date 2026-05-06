<template>
  <div class="project-layout" v-if="project">
    <aside class="secondary">
      <router-link class="back-workspace" :to="`/app/${wid}/projects`">← {{ workspaceName }}</router-link>
      <router-link class="hero-wrap" :to="`/app/${wid}/home`" title="Home">
        <div class="hero-image">
          <DatasetThumb v-if="coverImageId" :image-id="coverImageId" :boxes="[]" />
          <div v-else class="logo">{{ project.name.slice(0, 1).toUpperCase() }}</div>
        </div>
      </router-link>
      <p class="project-name">{{ project.name }}</p>
      <p class="data-label">Data</p>
      <nav class="subnav">
        <router-link v-if="project.can_edit" :to="base + '/upload'" class="subitem"><span class="ic">⤴</span>Upload Data</router-link>
        <router-link v-if="project.can_edit" :to="base + '/annotate'" class="subitem"><span class="ic">✎</span>Annotate</router-link>
        <router-link :to="base + '/dataset'" class="subitem"><span class="ic">▦</span>Dataset</router-link>
        <router-link :to="base + '/versions'" class="subitem"><span class="ic">⎘</span>Versions</router-link>
        <router-link :to="base + '/classes-tags'" class="subitem"><span class="ic">◔</span>Classes & Tags</router-link>
      </nav>
    </aside>
    <div class="project-content">
      <router-view />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApi } from '@/composables/useApi'
import DatasetThumb from '@/components/DatasetThumb.vue'

const api = useApi()
const route = useRoute()
const router = useRouter()
const wid = computed(() => String(route.params.wid || ''))
const pid = computed(() => String(route.params.pid || ''))
const base = computed(() => `/app/${wid.value}/projects/${pid.value}`)
const project = ref(null)
const coverImageId = ref('')
const workspaceName = ref('Workspace')

onMounted(async () => {
  project.value = await api.request(`/projects/${pid.value}`)
  if (!project.value?.can_edit && (route.path.endsWith('/upload') || route.path.endsWith('/annotate'))) {
    await router.replace(`${base.value}/dataset`)
  }
  const ws = await api.request(`/workspaces/${wid.value}`)
  workspaceName.value = ws?.name || 'Workspace'
  const list = await api.request(`/workspaces/${wid.value}/projects`)
  const row = (list.projects || []).find((x) => String(x.id) === pid.value)
  coverImageId.value = row?.cover_image_id || ''
})
</script>

<style scoped>
.project-layout { display: grid; grid-template-columns: 190px 1fr; min-height: 100vh; }
.secondary { border-right: 1px solid var(--ag-border); background: linear-gradient(160deg, var(--ag-surface), var(--ag-surface2)); padding: 12px 0 16px; position: sticky; top: 0; height: 100vh; overflow: hidden; }
.back-workspace { display: inline-block; margin: 0 12px 10px; color: var(--ag-text); text-decoration: none; font-size: 12px; }
.hero-wrap { width: 100%; padding: 0 20px; box-sizing: border-box; display: block; text-decoration: none; }
.hero-image { width: 100%; height: 130px; overflow: hidden; background: rgba(255,255,255,.04); border: 1px solid var(--ag-border); border-radius: 12px; }
.hero-image :deep(img) { width: 100%; height: 100%; object-fit: cover; display: block; }
.logo { width: 100%; height: 100%; background: rgba(77,107,254,.2); display: grid; place-items: center; font-weight: 700; font-size: 26px; border-radius: 12px; }
.project-name { margin: 10px 12px 0; font-weight: 700; font-size: 14px; }
.data-label { margin: 14px 12px 6px; font-size: 15px; font-weight: 500; color: var(--ag-text); }
.subnav { margin-top: 0; display: flex; flex-direction: column; gap: 3px; padding: 0 8px; }
.subitem { color: var(--ag-muted); text-decoration: none; padding: 8px 8px; border-radius: 8px; font-size: 12px; display: flex; align-items: center; gap: 8px; }
.ic { width: 16px; display: inline-flex; justify-content: center; opacity: .95; }
.subitem.router-link-active { color: var(--ag-text); background: rgba(77,107,254,.16); }
.project-content { min-width: 0; }
</style>
