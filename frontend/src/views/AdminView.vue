<template>
  <section class="page">
    <div v-if="adminToken" class="top">
      <h2>Admin Panel</h2>
      <div class="top-actions">
        <button type="button" class="ag-btn" :disabled="overlayLoading" @click="refreshOverview">Refresh</button>
        <button type="button" class="ag-btn" @click="logout">Logout</button>
      </div>
    </div>

    <div v-if="!adminToken" class="login-center">
      <article class="ag-card login-card">
      <h3>Admin Login</h3>
      <p v-if="err" class="err">{{ err }}</p>
      <div class="form">
        <label>
          <span class="muted small">Login</span>
          <input v-model.trim="login" class="ag-input" type="text" autocomplete="username" />
        </label>
        <label>
          <span class="muted small">Password</span>
          <input v-model="password" class="ag-input" type="password" autocomplete="current-password" @keydown.enter.prevent="doLogin" />
        </label>
        <button type="button" class="ag-btn ag-btn-primary" :disabled="loading" @click="doLogin">
          {{ loading ? 'Signing in...' : 'Sign in' }}
        </button>
      </div>
    </article>
    </div>

    <template v-else>
      <p v-if="err" class="err">{{ err }}</p>
      <p v-if="lastUpdatedAt" class="muted small upd">Last updated: {{ fmtTime(lastUpdatedAt) }}</p>

      <div v-if="overview" class="grid">
        <article class="ag-card tile">
          <p class="muted small">Disk Used</p>
          <p class="n">{{ fmtGB(overview.disk?.used_gb) }} GB</p>
          <p class="muted small">{{ fmtPct(overview.disk?.used_pct) }} of {{ fmtGB(overview.disk?.total_gb) }} GB</p>
        </article>
        <article class="ag-card tile">
          <p class="muted small">Total Images</p>
          <p class="n">{{ overview.totals?.images || 0 }}</p>
        </article>
        <article class="ag-card tile">
          <p class="muted small">Total Users</p>
          <p class="n">{{ overview.totals?.users || 0 }}</p>
        </article>
      </div>

      <article v-if="overview" class="ag-card section">
        <h3>Users</h3>
        <div class="table-wrap">
          <table class="tbl">
            <thead>
              <tr>
                <th>User</th>
                <th>Workspaces</th>
                <th>Projects</th>
                <th>Images</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in overview.users || []" :key="u.id">
                <td>{{ userTitle(u) }}</td>
                <td>{{ u.workspace_count || 0 }}</td>
                <td>{{ u.project_count || 0 }}</td>
                <td>{{ u.image_count || 0 }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

      <div v-if="overview" class="split">
        <article class="ag-card section">
          <h3>Workspaces</h3>
          <div class="list">
            <div v-for="w in overview.workspaces || []" :key="w.id" class="list-row">
              <div>
                <p class="title">{{ w.name }}</p>
                <p class="muted small">{{ w.owner_email || 'no owner' }} · projects {{ w.project_count || 0 }} · images {{ w.image_count || 0 }}</p>
              </div>
              <button type="button" class="ag-btn danger" :disabled="loading" @click="deleteWorkspace(w.id, w.name)">Delete</button>
            </div>
          </div>
        </article>

        <article class="ag-card section">
          <h3>Projects</h3>
          <div class="list">
            <div v-for="p in overview.projects || []" :key="p.id" class="list-row">
              <div>
                <p class="title">{{ p.name }}</p>
                <p class="muted small">{{ p.workspace_name || p.workspace_id }} · images {{ p.image_count || 0 }}</p>
              </div>
              <button type="button" class="ag-btn danger" :disabled="loading" @click="deleteProject(p.id, p.name)">Delete</button>
            </div>
          </div>
        </article>
      </div>
    </template>
    <div v-if="overlayLoading" class="loading-overlay">
      <span>Loading ...</span>
    </div>
  </section>
</template>

<script setup>
import { onMounted, ref } from 'vue'

const ADMIN_TOKEN_KEY = 'ag_admin_token'
const ADMIN_OVERVIEW_CACHE_KEY = 'ag_admin_overview_cache'

const adminToken = ref(localStorage.getItem(ADMIN_TOKEN_KEY) || '')
const login = ref('admin')
const password = ref('123456')
const overview = ref(null)
const loading = ref(false)
const overlayLoading = ref(false)
const err = ref('')
const lastUpdatedAt = ref(0)

async function adminRequest(path, opts = {}) {
  const headers = { ...(opts.headers || {}) }
  if (adminToken.value) headers.Authorization = `Bearer ${adminToken.value}`
  if (opts.body && !(opts.body instanceof FormData) && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(`/api${path}`, { ...opts, headers })
  const text = await res.text()
  let data = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = { raw: text }
  }
  if (!res.ok) {
    const e = new Error((data && data.error) || 'request failed')
    e.status = res.status
    throw e
  }
  return data
}

function fmtGB(v) {
  const n = Number(v || 0)
  return Number.isFinite(n) ? n.toFixed(2) : '0.00'
}
function fmtPct(v) {
  const n = Number(v || 0)
  return Number.isFinite(n) ? `${n.toFixed(2)}%` : '0.00%'
}
function userTitle(u) {
  const fn = String(u.first_name || '').trim()
  const ln = String(u.last_name || '').trim()
  const full = `${fn} ${ln}`.trim()
  return full ? `${full} (${u.email})` : String(u.email || '')
}

function fmtTime(ts) {
  const n = Number(ts || 0)
  if (!Number.isFinite(n) || n <= 0) return '-'
  return new Date(n).toLocaleString()
}

function saveOverviewCache(payload) {
  try {
    localStorage.setItem(ADMIN_OVERVIEW_CACHE_KEY, JSON.stringify(payload))
  } catch {
    // ignore storage quota issues
  }
}

function restoreOverviewCache() {
  try {
    const raw = localStorage.getItem(ADMIN_OVERVIEW_CACHE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') return
    if (parsed.data) overview.value = parsed.data
    if (parsed.ts) lastUpdatedAt.value = Number(parsed.ts) || 0
  } catch {
    // ignore parse issues
  }
}

async function loadOverview(showOverlay = true) {
  if (!adminToken.value) return
  overlayLoading.value = showOverlay
  err.value = ''
  try {
    const data = await adminRequest('/admin/overview')
    overview.value = data
    lastUpdatedAt.value = Date.now()
    saveOverviewCache({ ts: lastUpdatedAt.value, data })
  } catch (e) {
    err.value = e.message || 'Failed to load admin overview'
    if (e.status === 401) logout()
  } finally {
    overlayLoading.value = false
  }
}

async function refreshOverview() {
  await loadOverview(true)
}

async function doLogin() {
  loading.value = true
  err.value = ''
  try {
    const js = await adminRequest('/admin/login', {
      method: 'POST',
      body: JSON.stringify({ login: login.value, password: password.value }),
    })
    adminToken.value = String(js.token || '')
    localStorage.setItem(ADMIN_TOKEN_KEY, adminToken.value)
    await loadOverview(true)
  } catch (e) {
    err.value = e.message || 'Login failed'
  } finally {
    loading.value = false
  }
}

function logout() {
  localStorage.removeItem(ADMIN_TOKEN_KEY)
  adminToken.value = ''
  overview.value = null
  overlayLoading.value = false
  lastUpdatedAt.value = 0
}

async function deleteWorkspace(id, name) {
  if (!id) return
  overlayLoading.value = true
  err.value = ''
  try {
    await adminRequest(`/admin/workspaces/${id}`, { method: 'DELETE' })
    await loadOverview(false)
  } catch (e) {
    err.value = e.message || 'Failed to delete workspace'
    overlayLoading.value = false
  }
}

async function deleteProject(id, name) {
  if (!id) return
  overlayLoading.value = true
  err.value = ''
  try {
    await adminRequest(`/admin/projects/${id}`, { method: 'DELETE' })
    await loadOverview(false)
  } catch (e) {
    err.value = e.message || 'Failed to delete project'
    overlayLoading.value = false
  }
}

onMounted(async () => {
  if (!adminToken.value) return
  restoreOverviewCache()
  await loadOverview(!overview.value)
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.top-actions { display: inline-flex; gap: 8px; }
.login-center {
  min-height: calc(100vh - 140px);
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-card { max-width: 420px; padding: 14px; }
.form { display: grid; gap: 10px; }
.grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; margin-bottom: 12px; }
.tile { padding: 12px; }
.n { margin: 6px 0 0; font-size: 24px; font-weight: 700; }
.section { padding: 12px; margin-bottom: 12px; }
.upd { margin: 0 0 10px; }
.table-wrap { overflow: auto; }
.tbl { width: 100%; border-collapse: collapse; font-size: 13px; }
.tbl th, .tbl td { text-align: left; padding: 8px; border-bottom: 1px solid var(--ag-border); white-space: nowrap; }
.split { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.list { display: grid; gap: 8px; }
.list-row { border: 1px solid var(--ag-border); border-radius: 10px; padding: 10px; display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.title { margin: 0 0 4px; font-weight: 600; }
.danger { border-color: rgba(255, 123, 123, 0.55); color: #ff9a9a; }
.muted { color: var(--ag-muted); }
.small { font-size: 12px; }
.err { color: #ff7b7b; margin: 0 0 10px; }
.loading-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 20, 30, 0.42);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  color: #e6ebff;
  z-index: 1000;
}
@media (max-width: 980px) {
  .grid { grid-template-columns: 1fr; }
  .split { grid-template-columns: 1fr; }
}
</style>

