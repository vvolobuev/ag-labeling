<template>
  <div class="page">
    <AgHeader />
    <div class="shell">
      <div class="head">
        <h1>Your workspaces</h1>
        <router-link class="ag-btn ag-btn-ghost sm" to="/">Home</router-link>
      </div>

      <form class="create ag-card" @submit.prevent="createWs">
        <h2>New workspace</h2>
        <div class="row">
          <input v-model.trim="wsName" class="ag-input" placeholder="Team or lab name" />
          <button class="ag-btn ag-btn-primary" type="submit" :disabled="!wsName || pending">Create</button>
        </div>
        <p v-if="err" class="err">{{ err }}</p>
      </form>

      <div class="grid">
        <router-link
          v-for="w in list"
          :key="w.id"
          class="ag-card ws lift-hover"
          :to="`/workspaces/${w.id}`"
        >
          <div class="ws-top">
            <span class="ag-pill">{{ roleLabel(w.role) }}</span>
            <button
              v-if="w.role === 'owner'"
              type="button"
              class="del-ws"
              @click.prevent.stop="deleteWorkspace(w)"
            >
              Delete
            </button>
          </div>
          <h3>{{ w.name }}</h3>
          <p class="slug">/{{ w.slug }}</p>
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AgHeader from '@/components/AgHeader.vue'
import { useApi } from '@/composables/useApi'

const api = useApi()
const router = useRouter()
const list = ref([])
const wsName = ref('')
const pending = ref(false)
const err = ref('')

function roleLabel(r) {
  const m = { owner: 'Owner', admin: 'Admin', annotator: 'Annotator', viewer: 'Viewer' }
  return m[r] || r
}

async function load() {
  const data = await api.request('/workspaces')
  list.value = data.workspaces || []
}

async function createWs() {
  err.value = ''
  pending.value = true
  try {
    const created = await api.request('/workspaces', {
      method: 'POST',
      body: JSON.stringify({ name: wsName.value }),
    })
    wsName.value = ''
    localStorage.setItem('ag_last_workspace', created.id)
    await router.push(`/app/${created.id}/projects`)
  } catch (e) {
    err.value = e.message
  } finally {
    pending.value = false
  }
}

async function deleteWorkspace(w) {
  if (!w?.id) return
  await api.request(`/workspaces/${w.id}`, { method: 'DELETE' })
  list.value = list.value.filter((x) => String(x.id) !== String(w.id))
  const last = localStorage.getItem('ag_last_workspace') || ''
  if (last === String(w.id)) localStorage.removeItem('ag_last_workspace')
}

onMounted(() => {
  load().catch((e) => {
    err.value = e.message
  })
})
</script>

<style scoped>
.page {
  min-height: 100vh;
  padding-bottom: 60px;
}

.shell {
  max-width: 1100px;
  margin: 0 auto;
  padding: 24px 22px;
}

.head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 22px;
}

.head h1 {
  margin: 0;
  font-size: clamp(22px, 3vw, 30px);
}

.sm {
  padding: 8px 14px !important;
  font-size: 13px !important;
  text-decoration: none;
  align-self: center;
}

.create h2 {
  margin: 0 0 12px;
  font-size: 16px;
}

.row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.row .ag-input {
  flex: 1;
  min-width: 220px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
  margin-top: 22px;
}

.ws {
  text-decoration: none;
  color: inherit;
}

.ws-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.del-ws {
  border: 1px solid rgba(255, 123, 123, 0.35);
  color: #ff9b9b;
  background: rgba(255, 123, 123, 0.08);
  border-radius: 8px;
  font-size: 12px;
  padding: 4px 8px;
  cursor: pointer;
}

.ws h3 {
  margin: 8px 0 4px;
  font-size: 18px;
}

.slug {
  margin: 0;
  font-size: 13px;
  color: var(--ag-muted);
}

.err {
  color: #ff7b7b;
  margin-top: 10px;
}
</style>
