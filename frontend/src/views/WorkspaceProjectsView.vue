<template>
  <section class="page">
    <div class="top">
      <p class="ws-name">{{ workspaceName || 'Workspace' }}</p>
      <h1>Projects</h1>
    </div>

    <div class="controls">
      <input v-model.trim="q" class="ag-input search" type="search" placeholder="Search projects" @input="load" />
      <label class="sort-wrap">
        <span class="muted">Sort:</span>
        <AgSelect v-model="sortBy" :options="sortOptions" class="sort-select" />
      </label>
      <button class="ag-btn ag-btn-primary new-btn" type="button" @click="openCreateModal">New Project</button>
    </div>

    <p v-if="!projects.length" class="muted empty">Nothing found. Try a different query.</p>
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
              <span class="status">{{ p.is_public ? 'Public' : 'Private' }}</span>
              <span class="muted">{{ p.image_count || 0 }} images</span>
            </p>
          </div>
        </div>
        <div class="right">
          <button class="dots" type="button" @click="menuFor = menuFor === p.id ? '' : p.id">⋮</button>
          <div v-if="menuFor === p.id" class="menu ag-card">
            <router-link class="menu-item" :to="`/app/${wid}/projects/${p.id}/dataset`">Open</router-link>
            <button class="menu-item btn" type="button" @click="toggleVisibility(p)">
              {{ p.is_public ? 'Make private' : 'Make public' }}
            </button>
            <button class="menu-item btn danger" type="button" @click="deleteProject(p)">
              Delete dataset
            </button>
          </div>
        </div>
      </article>
    </div>

    <div v-if="newProjectOpen" class="overlay" @click.self="closeCreateModal">
      <div class="modal ag-card">
        <div class="modal-head">
          <h3>Let's create your project.</h3>
          <button class="close-btn" type="button" @click="closeCreateModal">✕</button>
        </div>
        <p class="muted ws-line">{{ workspaceName || 'Workspace' }}</p>

        <div class="form-grid">
          <label class="field">
            <span>Project Name</span>
            <input v-model.trim="newProjectName" class="ag-input" type="text" placeholder="My dataset project" />
          </label>
          <label class="field">
            <span>Visibility</span>
            <AgSelect v-model="newProjectVisibility" :options="visibilityOptions" />
          </label>
        </div>

        <div class="mode-grid">
          <button
            type="button"
            class="mode-card"
            :class="{ active: createMode === 'import' }"
            @click="createMode = 'import'"
          >
            <strong>Import Project</strong>
            <span>Upload ZIP or folder</span>
          </button>
          <button
            type="button"
            class="mode-card"
            :class="{ active: createMode === 'create' }"
            @click="createMode = 'create'"
          >
            <strong>Create New</strong>
            <span>Create empty project</span>
          </button>
        </div>

        <div v-if="createMode === 'import'" class="import-block">
          <div class="import-type">
            <label><input v-model="importType" type="radio" value="zip" /> ZIP archive</label>
            <label><input v-model="importType" type="radio" value="folder" /> Folder</label>
          </div>
          <input v-if="importType === 'zip'" ref="zipRef" type="file" class="file" accept=".zip,application/zip" />
          <input v-else ref="folderRef" type="file" class="file" webkitdirectory multiple />
          <div v-if="creating" class="import-progress">
            <div class="prog-line">
              <div class="prog-head">
                <span>Upload · {{ importUploadPercent }}%</span>
                <span class="time">{{ uploadElapsedLabel }} · {{ uploadRemainLabel }}</span>
              </div>
              <div class="bar"><div class="fill up" :style="{ width: `${importUploadPercent}%` }" /></div>
            </div>
            <div class="prog-line">
              <div class="prog-head">
                <span>Import · {{ importServerPercent }}%</span>
                <span class="time">{{ importElapsedLabel }} · {{ importRemainLabel }}</span>
              </div>
              <div class="bar"><div class="fill srv" :style="{ width: `${importServerPercent}%` }" /></div>
            </div>
            <p class="muted prog-detail">{{ importPhaseLabel }}<span v-if="importDetail"> · {{ importDetail }}</span></p>
          </div>
        </div>

        <p v-if="createErr" class="err">{{ createErr }}</p>

        <div class="actions">
          <button class="ag-btn ag-btn-ghost" type="button" @click="closeCreateModal">
            {{ creating ? 'Cancel Upload' : 'Cancel' }}
          </button>
          <button class="ag-btn ag-btn-primary" type="button" @click="submitCreateProject" :disabled="creating">
            {{ creating ? 'Working...' : createMode === 'import' ? 'Create & Import' : 'Create Project' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import DatasetThumb from '@/components/DatasetThumb.vue'
import AgSelect from '@/components/AgSelect.vue'
import { getToken, useApi } from '@/composables/useApi'
import { messageFromXHR, normalizeImportError, pollImportJob, postMultipartWithProgress } from '@/composables/uploadMultipart'

const api = useApi()
const route = useRoute()
const wid = computed(() => String(route.params.wid || ''))
const q = ref('')
const projects = ref([])
const menuFor = ref('')
const sortBy = ref('edited')
const workspaceName = ref('')
const newProjectOpen = ref(false)
const newProjectName = ref('')
const newProjectVisibility = ref('private')
const createMode = ref('create')
const importType = ref('zip')
const zipRef = ref(null)
const folderRef = ref(null)
const creating = ref(false)
const createErr = ref('')
const importUploadPercent = ref(0)
const importServerPercent = ref(0)
const importPhase = ref('')
const importDetail = ref('')
const cancelRequested = ref(false)
let currentImportXHR = null
let activeImportProjectId = ''
let activeImportJobId = ''
const nowTs = ref(Date.now())
let progressTimer = null
const uploadStartedAt = ref(0)
const importStartedAt = ref(0)
const uploadDoneAt = ref(0)
const importDoneAt = ref(0)

const importPhaseLabel = computed(() => {
  const p = String(importPhase.value || '').trim().toLowerCase()
  if (p === 'queued') return 'Queued'
  if (p === 'version') return 'Creating version'
  if (p === 'storage') return 'Preparing storage'
  if (p === 'import') return 'Adding images to DB'
  if (p === 'finalize') return 'Finalizing'
  if (p === 'done') return 'Done'
  if (p === 'error') return 'Error'
  if (!p) return 'Waiting for server'
  return importPhase.value
})

function formatDurationSec(sec) {
  const s = Math.max(0, Math.floor(sec || 0))
  const mm = Math.floor(s / 60)
  const ss = s % 60
  return `${String(mm).padStart(2, '0')}:${String(ss).padStart(2, '0')}`
}

function estimateRemain(elapsedSec, percent) {
  const p = Number(percent || 0)
  if (!(p > 0 && p < 100)) return 0
  const total = (elapsedSec * 100) / p
  return Math.max(0, Math.round(total - elapsedSec))
}

const uploadElapsedSec = computed(() => {
  if (!uploadStartedAt.value) return 0
  const end = uploadDoneAt.value || nowTs.value
  return Math.max(0, (end - uploadStartedAt.value) / 1000)
})
const importElapsedSec = computed(() => {
  if (!importStartedAt.value) return 0
  const end = importDoneAt.value || nowTs.value
  return Math.max(0, (end - importStartedAt.value) / 1000)
})
const uploadRemainSec = computed(() => estimateRemain(uploadElapsedSec.value, importUploadPercent.value))
const importRemainSec = computed(() => estimateRemain(importElapsedSec.value, importServerPercent.value))
const uploadElapsedLabel = computed(() => `elapsed ${formatDurationSec(uploadElapsedSec.value)}`)
const importElapsedLabel = computed(() => `elapsed ${formatDurationSec(importElapsedSec.value)}`)
const uploadRemainLabel = computed(() => `left ${formatDurationSec(uploadRemainSec.value)}`)
const importRemainLabel = computed(() => `left ${formatDurationSec(importRemainSec.value)}`)
const sortOptions = [
  { value: 'edited', label: 'Date Edited' },
  { value: 'created', label: 'Date Created' },
  { value: 'name', label: 'Project Name' },
]
const visibilityOptions = [
  { value: 'private', label: 'Private' },
  { value: 'public', label: 'Public' },
]

function startProgressTimer() {
  if (progressTimer) return
  progressTimer = setInterval(() => {
    nowTs.value = Date.now()
  }, 1000)
}

function stopProgressTimer() {
  if (!progressTimer) return
  clearInterval(progressTimer)
  progressTimer = null
}

function resetImportProgress() {
  importUploadPercent.value = 0
  importServerPercent.value = 0
  importPhase.value = ''
  importDetail.value = ''
  cancelRequested.value = false
  currentImportXHR = null
  activeImportProjectId = ''
  activeImportJobId = ''
  uploadStartedAt.value = 0
  importStartedAt.value = 0
  uploadDoneAt.value = 0
  importDoneAt.value = 0
  nowTs.value = Date.now()
  stopProgressTimer()
}

async function deleteProjectWithRetry(pid) {
  const id = String(pid || '').trim()
  if (!id) return false
  for (let i = 0; i < 30; i++) {
    try {
      await api.request(`/projects/${id}`, { method: 'DELETE' })
      projects.value = projects.value.filter((x) => String(x.id) !== id)
      return true
    } catch {
      await new Promise((r) => setTimeout(r, 1000))
    }
  }
  return false
}

function deriveServerPercent(status) {
  const base = Math.max(0, Math.min(100, Number(status?.percent ?? 0)))
  const phase = String(status?.phase || '').toLowerCase()
  if (phase !== 'import') return base
  const detail = String(status?.detail || '')
  const m = detail.match(/(\d+)\s+of\s+(\d+)/i)
  if (!m) return base
  const done = Number(m[1] || 0)
  const total = Number(m[2] || 0)
  if (!(done > 0 && total > 0)) return base
  const calc = 18 + (72 * done) / total
  const adj = Math.min(99, Math.max(19, Math.ceil(calc)))
  return Math.max(base, adj)
}

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

async function load() {
  const qq = q.value ? `?q=${encodeURIComponent(q.value)}` : ''
  const data = await api.request(`/workspaces/${wid.value}/projects${qq}`)
  projects.value = data.projects || []
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

async function toggleVisibility(p) {
  await api.request(`/projects/${p.id}`, {
    method: 'PATCH',
    body: JSON.stringify({ is_public: !p.is_public }),
  })
  p.is_public = !p.is_public
  menuFor.value = ''
}

async function deleteProject(p) {
  if (!p?.id) return
  await api.request(`/projects/${p.id}`, { method: 'DELETE' })
  projects.value = projects.value.filter((x) => String(x.id) !== String(p.id))
  menuFor.value = ''
}

function openCreateModal() {
  createErr.value = ''
  newProjectName.value = ''
  newProjectVisibility.value = 'private'
  createMode.value = 'create'
  importType.value = 'zip'
  resetImportProgress()
  newProjectOpen.value = true
}

async function closeCreateModal(opts = {}) {
  const shouldCancelImport = opts?.cancelImport !== false
  if (creating.value && shouldCancelImport) {
    cancelRequested.value = true
    try {
      currentImportXHR?.abort()
    } catch {}
    if (activeImportJobId) {
      try {
        await api.request(`/import-jobs/${activeImportJobId}/cancel`, { method: 'POST' })
      } catch {}
    }
    if (activeImportProjectId) {
      try {
        await api.request(`/projects/${activeImportProjectId}`, { method: 'DELETE' })
      } catch {}
    }
    creating.value = false
  }
  newProjectOpen.value = false
  createErr.value = ''
  resetImportProgress()
}

async function submitCreateProject() {
  createErr.value = ''
  const name = newProjectName.value.trim()
  if (!name) {
    createErr.value = 'Project Name is required.'
    return
  }
  creating.value = true
  resetImportProgress()
  startProgressTimer()
  uploadStartedAt.value = Date.now()
  uploadDoneAt.value = 0
  importDoneAt.value = 0
  let createdProjectId = ''
  try {
    const created = await api.request(`/workspaces/${wid.value}/projects`, {
      method: 'POST',
      body: JSON.stringify({ name }),
    })
    const pid = created?.id
    if (!pid) throw new Error('Project creation failed')
    createdProjectId = String(pid)
    activeImportProjectId = String(pid)

    if (newProjectVisibility.value === 'public') {
      await api.request(`/projects/${pid}`, {
        method: 'PATCH',
        body: JSON.stringify({ is_public: true }),
      })
    }

    if (createMode.value === 'import') {
      importPhase.value = 'queued'
      if (importType.value === 'zip') {
        const f = zipRef.value?.files?.[0]
        if (!f) throw new Error('Choose ZIP archive')
        const fd = new FormData()
        fd.append('file', f)
        const xhr = await postMultipartWithProgress(`/api/projects/${pid}/versions/import-zip`, fd, {
          headers: { Authorization: `Bearer ${getToken()}`, 'X-AlphaGuard-Import-Async': '1' },
          onRequestReady: (x) => {
            currentImportXHR = x
          },
          onUploadProgress: (p) => {
            if (typeof p?.percent === 'number') {
              importUploadPercent.value = p.percent
              if (importUploadPercent.value >= 100 && !uploadDoneAt.value) uploadDoneAt.value = Date.now()
            }
          },
          onUploadComplete: () => {
            importUploadPercent.value = 100
            if (!uploadDoneAt.value) uploadDoneAt.value = Date.now()
          },
        })
        if (cancelRequested.value) throw new Error('Import cancelled')
        if (xhr.status < 200 || xhr.status >= 300) throw new Error(messageFromXHR(xhr))
        importUploadPercent.value = 100
        uploadDoneAt.value = Date.now()
        const payload = JSON.parse(xhr.responseText || '{}')
        const jobID = String(payload?.job_id || '')
        if (!jobID) throw new Error('Import job was not created')
        if (jobID) {
          activeImportJobId = jobID
          importStartedAt.value = Date.now()
          importDoneAt.value = 0
          const status = await pollImportJob(jobID, {
            intervalMs: 550,
            token: getToken(),
            maxMs: 0,
            shouldStop: () => cancelRequested.value,
            onTick: (s) => {
              importPhase.value = s?.phase || ''
              importDetail.value = s?.detail || ''
              const p = deriveServerPercent(s)
              if (p >= 0) {
                importServerPercent.value = Math.min(100, p)
                if (importServerPercent.value >= 100 && !importDoneAt.value) importDoneAt.value = Date.now()
              }
            },
          })
          importServerPercent.value = 100
          importPhase.value = status?.phase || 'done'
          importDoneAt.value = Date.now()
        }
      } else {
        const files = folderRef.value?.files || []
        if (!files.length) throw new Error('Choose folder')
        const fd = new FormData()
        for (let i = 0; i < files.length; i++) {
          const f = files[i]
          fd.append('files', f, f.webkitRelativePath || f.name)
        }
        const xhr = await postMultipartWithProgress(`/api/projects/${pid}/versions/import-folder`, fd, {
          headers: { Authorization: `Bearer ${getToken()}`, 'X-AlphaGuard-Import-Async': '1' },
          onRequestReady: (x) => {
            currentImportXHR = x
          },
          onUploadProgress: (p) => {
            if (typeof p?.percent === 'number') {
              importUploadPercent.value = p.percent
              if (importUploadPercent.value >= 100 && !uploadDoneAt.value) uploadDoneAt.value = Date.now()
            }
          },
          onUploadComplete: () => {
            importUploadPercent.value = 100
            if (!uploadDoneAt.value) uploadDoneAt.value = Date.now()
          },
        })
        if (cancelRequested.value) throw new Error('Import cancelled')
        if (xhr.status < 200 || xhr.status >= 300) throw new Error(messageFromXHR(xhr))
        importUploadPercent.value = 100
        uploadDoneAt.value = Date.now()
        const payload = JSON.parse(xhr.responseText || '{}')
        const jobID = String(payload?.job_id || '')
        if (!jobID) throw new Error('Import job was not created')
        if (jobID) {
          activeImportJobId = jobID
          importStartedAt.value = Date.now()
          importDoneAt.value = 0
          const status = await pollImportJob(jobID, {
            intervalMs: 550,
            token: getToken(),
            maxMs: 0,
            shouldStop: () => cancelRequested.value,
            onTick: (s) => {
              importPhase.value = s?.phase || ''
              importDetail.value = s?.detail || ''
              const p = deriveServerPercent(s)
              if (p >= 0) {
                importServerPercent.value = Math.min(100, p)
                if (importServerPercent.value >= 100 && !importDoneAt.value) importDoneAt.value = Date.now()
              }
            },
          })
          importServerPercent.value = 100
          importPhase.value = status?.phase || 'done'
          importDoneAt.value = Date.now()
        }
      }
    }

    await load()
    creating.value = false
    await closeCreateModal({ cancelImport: false })
  } catch (e) {
    if (!cancelRequested.value) {
      createErr.value = normalizeImportError(e?.message || String(e))
    }
    if (createMode.value === 'import' && createdProjectId) {
      await deleteProjectWithRetry(createdProjectId)
    }
  } finally {
    creating.value = false
    currentImportXHR = null
    activeImportJobId = ''
    if (cancelRequested.value && activeImportProjectId) {
      await deleteProjectWithRetry(activeImportProjectId)
    }
    if (!cancelRequested.value) activeImportProjectId = ''
  }
}

onMounted(() => {
  api
    .request(`/workspaces/${wid.value}`)
    .then((d) => {
      workspaceName.value = d?.name || ''
    })
    .catch(() => {})
  load().catch(() => {})
})

onBeforeUnmount(() => {
  stopProgressTimer()
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
.new-btn { margin-left: auto; padding: 10px 14px; font-size: 13px; }
.grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; overflow: visible; }
.card { display: grid; grid-template-columns: 1fr auto; gap: 8px; padding: 10px; min-height: 112px; position: relative; overflow: visible; }
.left { display: grid; grid-template-columns: 104px 1fr; gap: 10px; min-width: 0; }
.cover { display: block; width: 104px; height: 84px; border-radius: 9px; overflow: hidden; background: rgba(255,255,255,.04); }
.title { color: #fff; text-decoration: none; font-weight: 650; font-size: 14px; }
.muted { color: var(--ag-muted); margin: 4px 0 0; }
.line { margin: 4px 0 0; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.row-meta { display: inline-flex; align-items: center; justify-content: flex-start; gap: 8px; }
.empty { margin: 10px 0 14px; }
.status { font-size: 12px; color: var(--ag-muted); }
.right { position: relative; z-index: 20; }
.dots { border: none; background: transparent; color: var(--ag-muted); width: 20px; height: 20px; cursor: pointer; font-size: 18px; line-height: 1; padding: 0; }
.dots:hover { color: var(--ag-text); }
.menu { position: absolute; right: 0; top: 34px; width: 180px; padding: 6px; z-index: 9999; box-shadow: 0 16px 36px rgba(0,0,0,.45); }
.menu-item { width: 100%; text-align: left; display: block; padding: 7px 8px; border-radius: 7px; text-decoration: none; color: var(--ag-text); font-size: 13px; }
.menu-item:hover { background: rgba(77, 107, 254, 0.14); }
.btn { border: none; background: transparent; font-family: inherit; cursor: pointer; }
.menu-item.danger { color: #ff9b9b; }
.menu-item.danger:hover { background: rgba(255, 123, 123, 0.14); color: #ffd2d2; }
.overlay { position: fixed; inset: 0; background: rgba(10, 10, 12, 0.66); display: grid; place-items: center; z-index: 200; padding: 20px; }
.modal { width: min(760px, 100%); }
.modal-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.modal-head h3 { margin: 0; font-size: 24px; }
.close-btn { border: 1px solid var(--ag-border); background: transparent; color: var(--ag-muted); border-radius: 8px; width: 34px; height: 34px; cursor: pointer; }
.ws-line { margin: 8px 0 16px; }
.form-grid { display: grid; grid-template-columns: 1fr 220px; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 8px; font-size: 13px; color: var(--ag-muted); }
.mode-grid { margin-top: 14px; display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.mode-card { min-height: 120px; border: 1px solid var(--ag-border); background: rgba(255,255,255,.03); border-radius: 12px; color: var(--ag-text); padding: 14px; text-align: left; cursor: pointer; display: flex; flex-direction: column; gap: 8px; font-family: inherit; }
.mode-card span { color: var(--ag-muted); font-size: 13px; }
.mode-card.active { border-color: var(--ag-accent); background: rgba(77, 107, 254, 0.14); }
.import-block { margin-top: 14px; padding: 12px; border: 1px dashed var(--ag-border); border-radius: 10px; }
.import-type { display: flex; gap: 18px; margin-bottom: 10px; color: var(--ag-muted); font-size: 13px; }
.file { color: var(--ag-muted); width: 100%; }
.import-progress { margin-top: 12px; display: grid; gap: 8px; }
.prog-line { display: grid; gap: 4px; }
.prog-head { display: flex; align-items: center; justify-content: space-between; font-size: 12px; color: var(--ag-muted); }
.time { opacity: 0.95; font-variant-numeric: tabular-nums; }
.bar { height: 8px; border-radius: 999px; border: 1px solid var(--ag-border); overflow: hidden; background: rgba(255,255,255,.03); }
.fill { height: 100%; transition: width .2s ease; }
.fill.up { background: linear-gradient(90deg, #4d6bfe, #6f88ff); }
.fill.srv { background: linear-gradient(90deg, #7189ff, #a8b7ff); }
.prog-detail { margin: 2px 0 0; font-size: 12px; }
.actions { margin-top: 14px; display: flex; justify-content: flex-end; gap: 10px; }
.err { color: #ff7b7b; margin: 10px 0 0; font-size: 13px; }
@media (max-width: 1220px) { .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 880px) {
  .grid { grid-template-columns: 1fr; }
  .controls { display: flex; flex-wrap: wrap; }
  .search { width: min(300px, 100%); }
  .new-btn { margin-left: 0; }
  .form-grid { grid-template-columns: 1fr; }
  .mode-grid { grid-template-columns: 1fr; }
}
</style>
