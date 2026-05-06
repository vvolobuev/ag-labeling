<template>
  <section class="page">
    <div class="top">
      <p class="ws-name">Explore</p>
      <h1>Public Projects</h1>
    </div>

    <div class="controls">
      <input v-model.trim="q" class="ag-input search" type="search" placeholder="Search projects" @input="load" />
      <label class="sort-wrap">
        <span class="muted">Sort:</span>
        <AgSelect v-model="sortBy" :options="sortOptions" class="sort-select" />
      </label>
    </div>

    <p v-if="!projects.length" class="muted empty">No public datasets found.</p>
    <div class="grid">
      <article v-for="p in sortedProjects" :key="p.id" class="card ag-card">
        <div class="left">
          <router-link class="cover" :to="`/app/${wid}/projects/${p.id}/dataset`">
            <DatasetThumb v-if="p.cover_image_id" :image-id="p.cover_image_id" :boxes="[]" />
          </router-link>
          <div class="meta">
            <router-link class="title" :to="`/app/${wid}/projects/${p.id}/dataset`">{{ p.name }}</router-link>
            <p class="muted line">Edited {{ ago(p.updated_at) }}</p>
            <p class="line row-meta">
              <span class="status">Public</span>
              <span class="muted">{{ p.image_count || 0 }} images</span>
              <span class="muted">{{ p.workspace_name }}</span>
            </p>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import DatasetThumb from '@/components/DatasetThumb.vue'
import AgSelect from '@/components/AgSelect.vue'
import { useApi } from '@/composables/useApi'

const api = useApi()
const route = useRoute()
const wid = computed(() => String(route.params.wid || ''))
const q = ref('')
const projects = ref([])
const sortBy = ref('edited')
const sortOptions = [
  { value: 'edited', label: 'Date Edited' },
  { value: 'created', label: 'Date Created' },
  { value: 'name', label: 'Project Name' },
]

function ago(v) {
  const ts = Number(v || 0)
  if (!ts) return 'just now'
  const sec = Math.max(1, Math.floor(Date.now() / 1000 - ts))
  if (sec < 60) return `${sec}s ago`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`
  const d = Math.floor(hr / 24)
  if (d < 30) return `${d}d ago`
  const mo = Math.floor(d / 30)
  if (mo < 12) return `${mo}mo ago`
  return `${Math.floor(mo / 12)}y ago`
}

const sortedProjects = computed(() => {
  const arr = [...projects.value]
  if (sortBy.value === 'name') {
    arr.sort((a, b) => String(a.name || '').localeCompare(String(b.name || ''), 'en'))
    return arr
  }
  if (sortBy.value === 'created') {
    arr.sort((a, b) => Number(b.created_at || 0) - Number(a.created_at || 0))
    return arr
  }
  arr.sort((a, b) => Number(b.updated_at || 0) - Number(a.updated_at || 0))
  return arr
})

async function load() {
  const qq = q.value ? `?q=${encodeURIComponent(q.value)}` : ''
  const data = await api.request(`/explore/projects${qq}`)
  projects.value = data.projects || []
}

onMounted(() => {
  load().catch(() => {})
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.top { margin-bottom: 10px; }
.ws-name { margin: 0; color: #fff; font-size: 21px; font-weight: 700; }
h1 { margin: 14px 0 0; font-size: 18px; font-weight: 600; color: var(--ag-muted); }
.controls { display: flex; align-items: center; gap: 10px; margin: 16px 0 12px; width: 100%; }
.search { width: 300px; }
.sort-wrap { display: inline-flex; align-items: center; gap: 6px; margin-left: 0; }
.sort-wrap .muted { font-size: 12px; }
.sort-select { width: 150px; font-size: 13px; padding-top: 9px; padding-bottom: 9px; }
.grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; overflow: visible; }
.card { display: grid; grid-template-columns: 1fr; gap: 8px; padding: 10px; min-height: 112px; position: relative; overflow: visible; }
.left { display: grid; grid-template-columns: 104px 1fr; gap: 10px; min-width: 0; }
.cover { display: block; width: 104px; height: 84px; border-radius: 9px; overflow: hidden; background: rgba(255,255,255,.04); }
.title { color: #fff; text-decoration: none; font-weight: 650; font-size: 14px; }
.muted { color: var(--ag-muted); margin: 4px 0 0; }
.line { margin: 4px 0 0; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.row-meta { display: inline-flex; align-items: center; justify-content: flex-start; gap: 8px; }
.empty { margin: 10px 0 14px; }
.status { font-size: 12px; color: var(--ag-muted); }
@media (max-width: 1220px) { .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 880px) {
  .grid { grid-template-columns: 1fr; }
  .controls { display: flex; flex-wrap: wrap; }
  .search { width: min(300px, 100%); }
}
</style>
