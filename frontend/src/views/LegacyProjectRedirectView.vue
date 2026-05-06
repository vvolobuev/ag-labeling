<template>
  <section class="page">
    <p class="muted">Redirecting to project dataset...</p>
  </section>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApi } from '@/composables/useApi'

const route = useRoute()
const router = useRouter()
const api = useApi()

onMounted(async () => {
  const pid = String(route.params.pid || '')
  if (!pid) {
    await router.replace('/workspaces')
    return
  }
  try {
    const p = await api.request(`/projects/${pid}`)
    const wid = String(p.workspace_id || '')
    if (!wid) throw new Error('workspace missing')
    await router.replace(`/app/${wid}/projects/${pid}/dataset`)
  } catch {
    await router.replace('/workspaces')
  }
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.muted { color: var(--ag-muted); }
</style>
