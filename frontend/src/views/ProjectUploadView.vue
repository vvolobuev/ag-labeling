<template>
  <section class="page">
    <h2>Upload</h2>
    <label class="batch">
      <span class="muted">Batch Name</span>
      <input v-model.trim="batchName" class="ag-input" />
    </label>

    <div class="ag-card uploader" @dragover.prevent @drop.prevent="onDrop">
      <p class="drop-title">Drag and drop file(s) to upload, or:</p>
      <div class="btns">
        <button class="ag-btn ag-btn-primary" type="button" @click="pickFolder">Select Folder</button>
        <button class="ag-btn ag-btn-ghost" type="button" @click="pickFiles">Select File(s)</button>
      </div>
      <input ref="folderRef" type="file" class="hid" webkitdirectory multiple @change="onFolderPicked" />
      <input ref="filesRef" type="file" class="hid" multiple accept=".jpg,.jpeg,.png,.bmp,.webp,.avif" @change="onFilesPicked" />

      <div class="support">
        <p class="muted st">Supported Formats</p>
        <div class="rows">
          <p><strong>Images</strong> .jpg, .png, .bmp, .webp, .avif</p>
        </div>
        <p class="muted note">*Max size of 20MB and 16,400 × 10,900 pixels.</p>
      </div>
    </div>

    <div v-if="pendingFiles.length" class="ag-card picked">
      <p class="muted">Selected: {{ pendingFiles.length }} file(s)</p>
      <button class="ag-btn ag-btn-primary" type="button" :disabled="loading" @click="uploadBatch">
        {{ loading ? 'Uploading...' : 'Upload Batch' }}
      </button>
      <div class="upload-stats">
        <p class="small muted">Total images: {{ selectedImageCount }}</p>
        <p class="small muted">Processed: {{ loading ? processedEstimate : selectedImageCount }} / {{ selectedImageCount }}</p>
        <p class="small muted">Duplicates: {{ loading ? '...' : uploadStats.duplicates }}</p>
        <p class="small muted">Added to batch: {{ loading ? '...' : uploadStats.imported }}</p>
      </div>
      <div v-if="loading" class="progress-wrap">
        <div class="stage">
          <div class="stage-head">
            <span>Loading images</span>
            <span>{{ uploadPercent }}%</span>
          </div>
          <div class="progress-line">
            <div class="progress-fill" :style="{ width: `${uploadPercent}%` }" />
          </div>
        </div>
        <div class="stage">
          <div class="stage-head">
            <span>Adding to DB</span>
            <span>{{ dbPercent }}%</span>
          </div>
          <div class="progress-line">
            <div class="progress-fill db" :style="{ width: `${dbPercent}%` }" />
          </div>
        </div>
        <div class="stage">
          <div class="stage-head">
            <span>Finishing</span>
            <span>{{ finalizePercent }}%</span>
          </div>
          <div class="progress-line">
            <div class="progress-fill fin" :style="{ width: `${finalizePercent}%` }" />
          </div>
        </div>
      </div>
    </div>

    <div v-if="canStartAnnotating" class="start-row">
      <button class="ag-btn ag-btn-primary" type="button" @click="startAnnotating">Start Annotating</button>
    </div>

    <div v-if="!loading && uploadStats.totalImages > 0" class="ag-card upload-summary">
      <div class="summary-head">
        <h3>Upload Summary</h3>
        <span class="muted small">Batch: {{ lastBatch || batchName }}</span>
      </div>
      <div class="summary-grid">
        <div class="summary-item">
          <span class="summary-label">Total images</span>
          <strong class="summary-value">{{ uploadStats.totalImages }}</strong>
        </div>
        <div class="summary-item good">
          <span class="summary-label">Added to batch</span>
          <strong class="summary-value">{{ uploadStats.imported }}</strong>
        </div>
        <div class="summary-item warn">
          <span class="summary-label">Duplicates skipped</span>
          <strong class="summary-value">{{ uploadStats.duplicates }}</strong>
        </div>
        <div class="summary-item">
          <span class="summary-label">Unsupported files</span>
          <strong class="summary-value">{{ uploadStats.skippedUnsupported }}</strong>
        </div>
        <div class="summary-item danger">
          <span class="summary-label">Failed</span>
          <strong class="summary-value">{{ uploadStats.failed }}</strong>
        </div>
      </div>
    </div>

    <p v-if="err" class="err">{{ err }}</p>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getToken, useApi } from '@/composables/useApi'
import { messageFromXHR, normalizeImportError, postMultipartWithProgress } from '@/composables/uploadMultipart'
const route = useRoute()
const router = useRouter()
const api = useApi()
const pid = computed(() => String(route.params.pid || ''))
const folderRef = ref(null)
const filesRef = ref(null)
const pendingFiles = ref([])
const err = ref('')
const ok = ref('')
const loading = ref(false)
const uploadPercent = ref(0)
const dbPercent = ref(0)
const finalizePercent = ref(0)
const batchName = ref(`Uploaded on ${new Date().toLocaleString()}`)
const lastBatch = ref('')
const uploadStats = ref({
  totalImages: 0,
  imported: 0,
  duplicates: 0,
  skippedUnsupported: 0,
  failed: 0,
})
let dbTimer = null

function clearDbTimer() {
  if (dbTimer) {
    clearInterval(dbTimer)
    dbTimer = null
  }
}

function startDbProgress() {
  clearDbTimer()
  dbPercent.value = Math.max(dbPercent.value, 8)
  dbTimer = setInterval(() => {
    if (dbPercent.value >= 92) {
      clearDbTimer()
      return
    }
    dbPercent.value += 3
  }, 120)
}

function makeFileWithRelativePath(file, relPath) {
  try {
    const next = new File([file], file.name, { type: file.type, lastModified: file.lastModified })
    Object.defineProperty(next, 'webkitRelativePath', {
      value: String(relPath || ''),
      configurable: true,
    })
    return next
  } catch {
    return file
  }
}

async function walkDirHandle(dirHandle, prefix = '') {
  const out = []
  for await (const entry of dirHandle.values()) {
    if (entry.kind === 'file') {
      const f = await entry.getFile()
      const rel = prefix ? `${prefix}/${entry.name}` : entry.name
      out.push(makeFileWithRelativePath(f, rel))
      continue
    }
    if (entry.kind === 'directory') {
      const nestedPrefix = prefix ? `${prefix}/${entry.name}` : entry.name
      const nested = await walkDirHandle(entry, nestedPrefix)
      out.push(...nested)
    }
  }
  return out
}

async function pickFolder() {
  err.value = ''
  const picker = window.showDirectoryPicker
  if (typeof picker !== 'function') {
    folderRef.value?.click()
    return
  }
  try {
    const dir = await picker.call(window)
    const files = await walkDirHandle(dir)
    if (!files.length) {
      pendingFiles.value = []
      err.value = 'Selected folder is empty.'
      return
    }
    pendingFiles.value = files
  } catch (e) {
    if (e?.name === 'AbortError') return
    err.value = 'Unable to open folder picker in this browser.'
  }
}

async function pickFiles() {
  err.value = ''
  const picker = window.showOpenFilePicker
  if (typeof picker !== 'function') {
    filesRef.value?.click()
    return
  }
  try {
    const handles = await picker.call(window, {
      multiple: true,
      types: [
        {
          description: 'Images',
          accept: { 'image/*': ['.jpg', '.jpeg', '.png', '.bmp', '.webp', '.avif'] },
        },
      ],
      excludeAcceptAllOption: false,
    })
    const files = await Promise.all(handles.map((h) => h.getFile()))
    pendingFiles.value = files
  } catch (e) {
    if (e?.name === 'AbortError') return
    err.value = 'Unable to open file picker in this browser.'
  }
}

function setFilesFromList(list) {
  pendingFiles.value = Array.from(list || [])
}

function onFolderPicked() {
  const files = Array.from(folderRef.value?.files || [])
  if (!files.length) {
    pendingFiles.value = []
    err.value = 'Selected folder is empty.'
    return
  }
  const hasPlainFiles = files.some((f) => !String(f.webkitRelativePath || '').includes('/'))
  if (hasPlainFiles) {
    pendingFiles.value = []
    err.value = 'Please select a folder, not individual files.'
    return
  }
  err.value = ''
  pendingFiles.value = files
}

function onFilesPicked() {
  const files = Array.from(filesRef.value?.files || [])
  const hasFolderEntries = files.some((f) => String(f.webkitRelativePath || '').includes('/'))
  if (hasFolderEntries) {
    pendingFiles.value = []
    err.value = 'Please use Select Folder for directory upload.'
    return
  }
  err.value = ''
  pendingFiles.value = files
}

function onDrop(e) {
  setFilesFromList(e.dataTransfer?.files)
}

function isSupportedImageFile(file) {
  const name = String(file?.name || '').toLowerCase()
  return /\.(jpg|jpeg|png|bmp|webp|avif)$/.test(name)
}

const selectedImageCount = computed(() => pendingFiles.value.filter((f) => isSupportedImageFile(f)).length)
const canStartAnnotating = computed(() => !loading.value && !!lastBatch.value && Number(uploadStats.value.imported || 0) > 0)
const processedEstimate = computed(() => {
  const total = selectedImageCount.value
  if (total <= 0) return 0
  return Math.min(total, Math.round((uploadPercent.value / 100) * total))
})

async function uploadBatch() {
  err.value = ''
  ok.value = ''
  if (!pendingFiles.value.length) {
    err.value = 'Select file(s) first.'
    return
  }
  loading.value = true
  uploadPercent.value = 0
  dbPercent.value = 0
  finalizePercent.value = 0
  uploadStats.value = {
    totalImages: selectedImageCount.value,
    imported: 0,
    duplicates: 0,
    skippedUnsupported: 0,
    failed: 0,
  }
  try {
    const fd = new FormData()
    fd.append('batch_name', batchName.value || '')
    for (const f of pendingFiles.value) fd.append('files', f, f.webkitRelativePath || f.name)
    const xhr = await postMultipartWithProgress(`/api/projects/${pid.value}/uploads/images`, fd, {
      headers: { Authorization: `Bearer ${getToken()}` },
      onUploadProgress: (p) => {
        if (typeof p?.percent === 'number') uploadPercent.value = p.percent
      },
      onUploadComplete: () => {
        uploadPercent.value = 100
        startDbProgress()
      },
    })
    if (xhr.status < 200 || xhr.status >= 300) throw new Error(messageFromXHR(xhr))
    uploadPercent.value = 100
    clearDbTimer()
    dbPercent.value = 100
    let uploadedBatch = batchName.value || ''
    try {
      const payload = JSON.parse(xhr.responseText || '{}')
      uploadedBatch = String(payload.batch_name || uploadedBatch).trim()
      uploadStats.value = {
        totalImages: Number(payload.total_images) || selectedImageCount.value,
        imported: Number(payload.imported) || 0,
        duplicates: Number(payload.duplicates) || 0,
        skippedUnsupported: Number(payload.skipped_unsupported) || 0,
        failed: Number(payload.failed) || 0,
      }
    } catch {}
    pendingFiles.value = []
    if (folderRef.value) folderRef.value.value = ''
    if (filesRef.value) filesRef.value.value = ''
    lastBatch.value = uploadStats.value.imported > 0 ? uploadedBatch : ''
    finalizePercent.value = 100
    ok.value = ''
  } catch (e) {
    err.value = normalizeImportError(e.message || 'Upload failed')
  } finally {
    clearDbTimer()
    loading.value = false
  }
}

async function startAnnotating() {
  if (!lastBatch.value) return
  await router.push({
    path: `/app/${route.params.wid}/projects/${pid.value}/annotate`,
    query: { batch: lastBatch.value },
  })
}

onBeforeUnmount(() => {
  clearDbTimer()
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.batch { display: flex; flex-direction: column; gap: 8px; max-width: 520px; margin-bottom: 12px; }
.muted { color: var(--ag-muted); }
.uploader { padding: 16px; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; }
.drop-title { margin: 0 0 12px; font-size: 14px; }
.btns { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 34px; justify-content: center; }
.hid { display: none; }
.support { border-top: 1px solid var(--ag-border); padding-top: 12px; width: 100%; max-width: 760px; }
.st { margin: 0 0 8px; font-size: 12px; }
.rows p { margin: 3px 0; font-size: 13px; }
.note { margin: 8px 0 0; font-size: 12px; }
.picked { margin-top: 12px; display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.upload-stats { margin-left: auto; min-width: 220px; }
.progress-wrap { width: 290px; margin-left: auto; display: grid; gap: 8px; }
.stage { display: grid; gap: 4px; }
.stage-head { display: flex; align-items: center; justify-content: space-between; color: var(--ag-muted); font-size: 12px; }
.progress-line { width: 100%; height: 8px; border-radius: 999px; border: 1px solid var(--ag-border); background: rgba(255,255,255,.03); overflow: hidden; }
.progress-fill { height: 100%; background: linear-gradient(90deg, var(--ag-accent), #6f8bff); transition: width .2s ease; }
.progress-fill.db { background: linear-gradient(90deg, #5b78ff, #88a0ff); }
.progress-fill.fin { background: linear-gradient(90deg, #738dff, #a9b9ff); }
.small { font-size: 12px; margin: 6px 0 0; text-align: right; }
.start-row { margin-top: 10px; }
.upload-summary { margin-top: 10px; padding: 12px; }
.summary-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 10px; }
.summary-head h3 { margin: 0; font-size: 14px; font-weight: 600; }
.summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 8px; }
.summary-item {
  border: 1px solid var(--ag-border);
  background: rgba(255, 255, 255, 0.02);
  border-radius: 10px;
  padding: 8px 10px;
  display: grid;
  gap: 3px;
}
.summary-label { font-size: 11px; color: var(--ag-muted); }
.summary-value { font-size: 18px; line-height: 1.1; color: var(--ag-text); }
.summary-item.good { border-color: rgba(120, 214, 149, 0.35); }
.summary-item.warn { border-color: rgba(241, 191, 89, 0.35); }
.summary-item.danger { border-color: rgba(255, 123, 123, 0.35); }
.err { color: #ff7b7b; }
</style>
