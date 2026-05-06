<template>
  <section class="page">
    <h2>Data</h2>
    <p class="muted">Project dataset overview.</p>
    <div class="grid">
      <article class="ag-card tile">
        <h3>Project</h3>
        <p class="muted">{{ project?.name || '-' }}</p>
      </article>
      <article class="ag-card tile">
        <h3>Versions</h3>
        <p class="muted">{{ versions.length }}</p>
      </article>
    </div>
    <div class="links">
      <router-link class="ag-link" :to="legacyProjectLink">Legacy detailed page</router-link>
      <router-link class="ag-link" :to="`/app/${route.params.wid}/projects/${route.params.pid}/upload`">Go to upload</router-link>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'
const route = useRoute()
const api = useApi()
const legacyProjectLink = computed(() => `/projects/${route.params.pid}`)
const project = ref(null)
const versions = ref([])

onMounted(async () => {
  project.value = await api.request(`/projects/${route.params.pid}`)
  const data = await api.request(`/projects/${route.params.pid}/versions`)
  versions.value = data.versions || []
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.muted { color: var(--ag-muted); }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 260px)); gap: 10px; margin-top: 10px; }
.tile h3 { margin: 0 0 8px; }
.tile p { margin: 0; }
.links { display: flex; gap: 16px; margin-top: 14px; }
</style>
