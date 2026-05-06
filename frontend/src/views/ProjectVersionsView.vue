<template>
  <section class="page">
    <header class="top">
      <h2>Versions</h2>
      <button v-if="canEditProject" type="button" class="ag-btn ag-btn-primary" @click="toggleWizard">
        {{ wizardOpen ? 'Close Wizard' : 'Create New Version' }}
      </button>
    </header>

    <div class="layout">
      <aside class="ag-card sidebar">
        <div class="side-head">
          <h3 class="side-title">Versions</h3>
          <p class="muted small">{{ filteredVersions.length }} shown</p>
        </div>
        <div class="filters">
          <input v-model.trim="q" class="ag-input" type="search" placeholder="Search version name" />
        </div>
        <div class="vlist">
          <button
            v-for="v in filteredVersions"
            :key="v.id"
            type="button"
            class="vcard"
            :class="{ active: v.id === selectedId }"
            @click="selectedId = v.id"
          >
            <p class="vname">{{ v.name || '(no name)' }}</p>
            <p class="muted small">{{ ts(v.created_at) }}</p>
          </button>
          <p v-if="!filteredVersions.length" class="muted small empty">No versions yet</p>
        </div>
      </aside>

      <article v-if="wizardOpen" class="ag-card details wizard">
        <h3>Create New Version</h3>
        <div class="steps-head">
          <span v-for="(s, i) in stepNames" :key="s" class="step-pill" :class="{ active: step === i + 1, done: step > i + 1 }">
            {{ i + 1 }}. {{ s }}
          </span>
        </div>

        <div v-if="step === 1" class="step-block">
          <h4>Source Images</h4>
          <p class="muted">Upload images you want to include in your dataset.</p>
          <div class="stats-grid">
            <article class="stat"><p class="muted small">Images</p><p class="n">{{ sourceStats.total_images }}</p></article>
            <article class="stat"><p class="muted small">Classes</p><p class="n">{{ sourceStats.class_count }}</p></article>
            <article class="stat"><p class="muted small">Unannotated</p><p class="n">{{ sourceStats.unannotated_images }}</p></article>
          </div>
        </div>

        <div v-else-if="step === 2" class="step-block">
          <h4>Train/Test Split</h4>
          <div class="split-row"><label>Train {{ splitTrain }}%</label><input v-model.number="splitTrain" type="range" min="10" max="90" step="1" /></div>
          <div class="split-row"><label>Valid {{ splitValid }}%</label><input v-model.number="splitValid" type="range" min="5" max="40" step="1" /></div>
          <div class="split-row"><label>Test {{ splitTest }}%</label><input v-model.number="splitTest" type="range" min="5" max="40" step="1" /></div>
          <p class="muted small">Train: {{ splitCounts.train }} · Valid: {{ splitCounts.valid }} · Test: {{ splitCounts.test }}</p>
          <button type="button" class="ag-btn ag-btn-ghost" @click="rebalanceSplit">Rebalance</button>
        </div>

        <div v-else-if="step === 3" class="step-block">
          <h4>Preprocessing</h4>
          <p class="muted" v-if="keepOriginalSize">Resize: keep original image size</p>
          <p class="muted" v-else>Resize: {{ resizeTo }}x{{ resizeTo }}</p>
          <div class="row">
            <label class="keep-original">
              <input v-model="keepOriginalSize" type="checkbox" />
              <span class="muted small">Keep original image size</span>
            </label>
          </div>
          <div class="row" v-if="!keepOriginalSize">
            <label class="muted small">Resize</label>
            <input v-model.number="resizeTo" class="ag-input rs" type="number" min="64" step="32" />
          </div>
        </div>

        <div v-else class="step-block">
          <h4>Create</h4>
          <p class="muted">
            Review your selections then click "Create" to create a moment-in-time snapshot of your dataset with the applied preprocessing steps.
          </p>
          <div class="row">
            <label class="muted small">Version Name</label>
            <input v-model.trim="newName" class="ag-input" placeholder="2026-05-05 8:31pm" />
          </div>
          <p class="muted small">Maximum Version Size: {{ sourceStats.total_images }}</p>
          <label class="muted small">Version Notes</label>
          <textarea v-model="notes" class="ag-input notes" placeholder="Add any version notes here..." />

          <div v-if="creating" class="progress">
            <div v-for="(p, idx) in progressSteps" :key="p.key" class="pb">
              <div class="pb-top"><span>{{ p.label }}</span><span>{{ progressState(idx) }}</span></div>
              <div class="bar"><span :style="{ width: progressWidth(idx) }" /></div>
            </div>
          </div>
        </div>

        <div class="wizard-actions">
          <button type="button" class="ag-btn" @click="stepBack" :disabled="step === 1 || creating">Back</button>
          <button v-if="step < 4" type="button" class="ag-btn ag-btn-primary" @click="stepNext">Continue</button>
          <button v-else type="button" class="ag-btn ag-btn-primary" :disabled="creating" @click="createVersion">
            {{ creating ? 'Creating...' : 'Create' }}
          </button>
        </div>
        <p v-if="createErr" class="err">{{ createErr }}</p>
      </article>

      <article v-else class="ag-card details">
        <p v-if="loadingMeta" class="muted">Loading ...</p>
        <template v-else-if="selectedMeta">
          <div class="dtop">
            <div>
              <h3 class="dtitle">{{ selectedMeta.name }}</h3>
              <p class="muted small">Created: {{ ts(selectedVersionTs) }}</p>
            </div>
            <div class="actions">
              <button type="button" class="ag-btn" @click="downloadZip" :disabled="downloading || !canDownloadSelected">
                {{ downloading ? '...' : 'Download' }}
              </button>
              <button
                v-if="canEditProject"
                type="button"
                class="ag-btn ag-btn-ghost danger"
                :disabled="deletingVersion || !selectedId"
                @click="deleteSelectedVersion"
              >
                {{ deletingVersion ? 'Deleting...' : 'Delete' }}
              </button>
            </div>
          </div>
          <p v-if="downloadErr" class="err">{{ downloadErr }}</p>
          <p v-if="metaErr" class="err">{{ metaErr }}</p>
          <div class="stats">
            <div class="stat"><p class="muted small">Total Images</p><p class="n">{{ totalImages }}</p></div>
            <div class="stat"><p class="muted small">Train</p><p class="n">{{ splitStats.train }}</p></div>
            <div class="stat"><p class="muted small">Valid</p><p class="n">{{ splitStats.valid }}</p></div>
            <div class="stat"><p class="muted small">Test</p><p class="n">{{ splitStats.test }}</p></div>
          </div>

          <div v-if="!showAllImages">
            <div class="preview-head">
              <h4>Preview</h4>
              <button type="button" class="view-all-btn" @click="openAllImages">View All Images →</button>
            </div>
            <div class="preview-strip">
              <router-link
                v-for="(im, idx) in previewImages"
                :key="im.id"
                class="thumb mini ag-card"
                :to="annotateLinkFromVersion(im.id, idx, previewImages)"
              >
                <div class="ph"><DatasetThumb :image-id="im.id" :boxes="im.boxes || []" /></div>
              </router-link>
            </div>
            <p v-if="!previewImages.length" class="muted small">No preview images.</p>

            <div class="meta-block">
              <h4>Preprocessing</h4>
              <p class="muted small">Auto-Orient: Applied</p>
              <p class="muted small" v-if="resizeLabel === 'original size'">Resize: Keep original image size</p>
              <p class="muted small" v-else-if="resizeLabel === 'not applied'">Resize: Not applied</p>
              <p class="muted small" v-else>Resize: {{ resizeLabel }}</p>
              <h4>Augmentations</h4>
              <p class="muted small">No augmentations were applied</p>
            </div>
          </div>

          <div v-else class="all-images">
            <div class="all-tabs">
              <button type="button" class="tab-btn" :class="{ active: allSplit === 'train' }" @click="setAllSplit('train')">
                Train<br /><strong>{{ splitStats.train }}</strong>
              </button>
              <button type="button" class="tab-btn" :class="{ active: allSplit === 'valid' }" @click="setAllSplit('valid')">
                Valid<br /><strong>{{ splitStats.valid }}</strong>
              </button>
              <button type="button" class="tab-btn" :class="{ active: allSplit === 'test' }" @click="setAllSplit('test')">
                Test<br /><strong>{{ splitStats.test }}</strong>
              </button>
            </div>
            <div class="all-grid">
              <router-link
                v-for="(im, idx) in allImages"
                :key="im.id"
                class="thumb mini ag-card"
                :to="annotateLinkFromVersion(im.id, idx, allImages)"
              >
                <div class="ph"><DatasetThumb :image-id="im.id" :boxes="im.boxes || []" /></div>
              </router-link>
            </div>
            <div class="all-footer">
              <p class="muted small">{{ allImages.length }} / {{ allTotal }} images</p>
              <button type="button" class="ag-btn ag-btn-ghost" :disabled="allImages.length >= allTotal || loadingAll" @click="loadAllImages(false)">
                {{ loadingAll ? 'Loading...' : 'Load More...' }}
              </button>
            </div>
          </div>
        </template>
        <p v-else class="muted">Select a version on the left</p>
      </article>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import DatasetThumb from '@/components/DatasetThumb.vue'
import { getToken, useApi } from '@/composables/useApi'

const api = useApi()
const route = useRoute()
const versions = ref([])
const selectedId = ref('')
const selectedMeta = ref(null)
const selectedVersionTs = ref(0)
const splitStats = ref({ train: 0, valid: 0, test: 0 })
const previewImages = ref([])
const loadingMeta = ref(false)
const metaErr = ref('')
const downloadErr = ref('')
const downloading = ref(false)
const deletingVersion = ref(false)

const q = ref('')
const previewSplit = ref('train')
const showAllImages = ref(false)
const allSplit = ref('train')
const allImages = ref([])
const allTotal = ref(0)
const allPage = ref(1)
const loadingAll = ref(false)
const canEditProject = ref(false)

const wizardOpen = ref(false)
const step = ref(1)
const stepNames = ['Source Images', 'Train/Test Split', 'Preprocessing', 'Create']
const newName = ref('')
const notes = ref('')
const resizeTo = ref(640)
const keepOriginalSize = ref(false)
const splitTrain = ref(75)
const splitValid = ref(18)
const splitTest = ref(7)
const sourceStats = ref({ total_images: 0, class_count: 0, unannotated_images: 0, suggested: { train_pct: 75, valid_pct: 18, test_pct: 7 } })
const creating = ref(false)
const createErr = ref('')
const createTick = ref(0)
let createTimer = 0

const progressSteps = [
  { key: 'source', label: 'Source images snapshot' },
  { key: 'split', label: 'Train/Valid/Test split' },
  { key: 'resize', label: 'Resize and preprocess' },
  { key: 'store', label: 'Store version metadata' },
]

const filteredVersions = computed(() => {
  const term = q.value.trim().toLowerCase()
  const list = versions.value.filter((v) => (v.name || '').toLowerCase().includes(term))
  return [...list].sort((a, b) => Number(b.created_at || 0) - Number(a.created_at || 0))
})

const totalImages = computed(() => splitStats.value.train + splitStats.value.valid + splitStats.value.test)
const canDownloadSelected = computed(() => !loadingMeta.value && !!selectedMeta.value && totalImages.value > 0)
const resizeLabel = computed(() => {
  const y = String(selectedMeta.value?.data_yaml || '')
  if (!/resize\s*:/i.test(y)) return 'not applied'
  if (/resize:\s*original/i.test(y)) return 'original size'
  const m = y.match(/resize:\s*(\d+)/i)
  const n = m ? Number(m[1]) : 0
  if (!n) return 'not applied'
  return `${n}x${n}`
})
const splitCounts = computed(() => {
  const total = Number(sourceStats.value.total_images || 0)
  const train = Math.round((total * splitTrain.value) / 100)
  const valid = Math.round((total * splitValid.value) / 100)
  const test = Math.max(0, total - train - valid)
  return { train, valid, test }
})

function ts(v) {
  return v ? new Date(Number(v) * 1000).toLocaleString() : '-'
}

function annotateLinkFromVersion(imageId, idx, listRef) {
  const items = Array.isArray(listRef?.value) ? listRef.value : listRef
  const ids = (Array.isArray(items) ? items : []).map((x) => x.id).filter(Boolean).join(',')
  return {
    path: `/annotate/${imageId}`,
    query: {
      from: route.fullPath,
      source: 'version',
      vid: String(selectedId.value || ''),
      ids,
      ix: String(idx),
    },
  }
}

function normalizeSplits() {
  const t = Math.max(10, Math.min(90, Number(splitTrain.value) || 75))
  const v = Math.max(5, Math.min(40, Number(splitValid.value) || 18))
  splitTrain.value = t
  splitValid.value = v
  splitTest.value = Math.max(5, 100 - t - v)
}

function rebalanceSplit() {
  splitTrain.value = Number(sourceStats.value?.suggested?.train_pct || 75)
  splitValid.value = Number(sourceStats.value?.suggested?.valid_pct || 18)
  splitTest.value = Number(sourceStats.value?.suggested?.test_pct || 7)
  normalizeSplits()
}

function stepBack() {
  if (step.value > 1) step.value -= 1
}

function stepNext() {
  if (step.value < 4) step.value += 1
}

function progressState(i) {
  if (!creating.value) return 'waiting'
  return createTick.value > i ? 'done' : 'running'
}

function progressWidth(i) {
  if (!creating.value) return '0%'
  if (createTick.value > i) return '100%'
  if (createTick.value === i) return '45%'
  return '0%'
}

async function loadVersions() {
  const data = await api.request(`/projects/${route.params.pid}/versions`)
  versions.value = data.versions || []
}

async function loadProjectMeta() {
  try {
    const p = await api.request(`/projects/${route.params.pid}`)
    canEditProject.value = Boolean(p?.can_edit)
    if (!canEditProject.value) wizardOpen.value = false
  } catch {
    canEditProject.value = false
    wizardOpen.value = false
  }
}

async function loadSourceStats() {
  try {
    const s = await api.request(`/projects/${route.params.pid}/versions/source-stats`)
    sourceStats.value = s || sourceStats.value
    splitTrain.value = Number(s?.suggested?.train_pct || 75)
    splitValid.value = Number(s?.suggested?.valid_pct || 18)
    splitTest.value = Number(s?.suggested?.test_pct || 7)
    normalizeSplits()
  } catch {
    sourceStats.value = { total_images: 0, class_count: 0, unannotated_images: 0, suggested: { train_pct: 75, valid_pct: 18, test_pct: 7 } }
  }
}

async function toggleWizard() {
  if (!canEditProject.value) return
  wizardOpen.value = !wizardOpen.value
  if (wizardOpen.value) {
    step.value = 1
    createErr.value = ''
    await loadSourceStats()
  }
}

async function loadPreview() {
  if (!selectedId.value) {
    previewImages.value = []
    return
  }
  const u = new URLSearchParams()
  u.set('split', previewSplit.value)
  u.set('page', '1')
  u.set('per_page', '10')
  try {
    const res = await api.request(`/versions/${selectedId.value}/images?${u.toString()}`)
    previewImages.value = res.images || []
  } catch {
    previewImages.value = []
  }
}

async function loadAllImages(reset = false) {
  if (!selectedId.value) return
  if (reset) {
    allImages.value = []
    allPage.value = 1
  }
  loadingAll.value = true
  try {
    const u = new URLSearchParams()
    u.set('split', allSplit.value)
    u.set('page', String(allPage.value))
    u.set('per_page', '50')
    const res = await api.request(`/versions/${selectedId.value}/images?${u.toString()}`)
    const incoming = res.images || []
    allTotal.value = Number(res.total || 0)
    allImages.value = reset ? incoming : [...allImages.value, ...incoming]
    allPage.value += 1
  } finally {
    loadingAll.value = false
  }
}

function openAllImages() {
  showAllImages.value = true
  allSplit.value = 'train'
  loadAllImages(true)
}

function setAllSplit(split) {
  allSplit.value = split
  loadAllImages(true)
}

async function loadSelectedVersion(id) {
  if (!id) {
    selectedMeta.value = null
    return
  }
  loadingMeta.value = true
  metaErr.value = ''
  downloadErr.value = ''
  try {
    const [meta, tr, va, te] = await Promise.all([
      api.request(`/versions/${id}`),
      api.request(`/versions/${id}/split-stats?split=train`).catch(() => ({ total: 0 })),
      api.request(`/versions/${id}/split-stats?split=valid`).catch(() => ({ total: 0 })),
      api.request(`/versions/${id}/split-stats?split=test`).catch(() => ({ total: 0 })),
    ])
    selectedMeta.value = meta
    selectedVersionTs.value = (versions.value.find((v) => v.id === id) || {}).created_at || 0
    splitStats.value = { train: Number(tr.total || 0), valid: Number(va.total || 0), test: Number(te.total || 0) }
    showAllImages.value = false
    allImages.value = []
    allTotal.value = 0
    allPage.value = 1
    await loadPreview()
  } catch (e) {
    selectedMeta.value = null
    splitStats.value = { train: 0, valid: 0, test: 0 }
    previewImages.value = []
    metaErr.value = e.message || 'Failed to load version'
  } finally {
    loadingMeta.value = false
  }
}

async function createVersion() {
  createErr.value = ''
  creating.value = true
  createTick.value = 0
  clearInterval(createTimer)
  createTimer = setInterval(() => {
    createTick.value = Math.min(progressSteps.length, createTick.value + 1)
  }, 700)
  try {
    const res = await api.request(`/projects/${route.params.pid}/versions/create-from-dataset`, {
      method: 'POST',
      body: JSON.stringify({
        name: newName.value,
        resize: keepOriginalSize.value ? 0 : resizeTo.value,
        keep_original_size: keepOriginalSize.value,
        train_pct: splitTrain.value,
        valid_pct: splitValid.value,
        test_pct: splitTest.value,
        rebalance: true,
        notes: notes.value,
      }),
    })
    newName.value = ''
    notes.value = ''
    keepOriginalSize.value = false
    wizardOpen.value = false
    step.value = 1
    await loadVersions()
    selectedId.value = res.id || selectedId.value
  } catch (e) {
    createErr.value = e.message || 'Failed to create version.'
  } finally {
    clearInterval(createTimer)
    creating.value = false
  }
}

async function downloadZip() {
  if (!selectedId.value || !canDownloadSelected.value) return
  downloadErr.value = ''
  downloading.value = true
  try {
    const tok = getToken()
    const res = await fetch(`/api/versions/${selectedId.value}/dataset.zip`, {
      headers: tok ? { Authorization: `Bearer ${tok}` } : {},
    })
    if (!res.ok) throw new Error('export failed')
    const b = await res.blob()
    const cd = res.headers.get('Content-Disposition')
    let name = 'dataset.zip'
    const m = cd && cd.match(/filename="?([^";]+)"?/i)
    if (m) name = m[1]
    const a = document.createElement('a')
    a.href = URL.createObjectURL(b)
    a.download = name
    a.click()
    URL.revokeObjectURL(a.href)
  } catch (e) {
    downloadErr.value = e.message || 'Download error'
  } finally {
    downloading.value = false
  }
}

async function deleteSelectedVersion() {
  if (!canEditProject.value || !selectedId.value) return
  deletingVersion.value = true
  metaErr.value = ''
  try {
    await api.request(`/versions/${selectedId.value}`, { method: 'DELETE' })
    const removed = selectedId.value
    await loadVersions()
    if (!versions.value.length) {
      selectedId.value = ''
      selectedMeta.value = null
      splitStats.value = { train: 0, valid: 0, test: 0 }
      previewImages.value = []
      return
    }
    if (selectedId.value === removed || !versions.value.some((v) => v.id === selectedId.value)) {
      selectedId.value = versions.value[0].id
    }
  } catch (e) {
    metaErr.value = e.message || 'Failed to delete version'
  } finally {
    deletingVersion.value = false
  }
}

watch([splitTrain, splitValid], () => normalizeSplits())

watch(filteredVersions, (list) => {
  if (!list.length) {
    selectedId.value = ''
    return
  }
  if (!selectedId.value || !list.some((v) => v.id === selectedId.value)) {
    selectedId.value = list[0].id
  }
})

watch(selectedId, async (id) => {
  await loadSelectedVersion(id)
})
watch(previewSplit, async () => {
  await loadPreview()
})

onMounted(async () => {
  await loadProjectMeta()
  await loadVersions()
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.top { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.wizard { margin-bottom: 12px; }
.steps-head { display: flex; flex-wrap: wrap; gap: 8px; margin: 10px 0 14px; }
.step-pill { border: 1px solid var(--ag-border); border-radius: 999px; padding: 6px 10px; font-size: 12px; color: var(--ag-muted); }
.step-pill.active { border-color: var(--ag-accent); color: var(--ag-text); }
.step-pill.done { border-color: rgba(130, 255, 190, 0.45); color: #9ff0b8; }
.step-block { margin-bottom: 12px; }
.row { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.keep-original { display: inline-flex; align-items: center; gap: 8px; cursor: pointer; }
.rs { max-width: 120px; }
.notes { min-height: 96px; width: 100%; }
.split-row { display: grid; grid-template-columns: 140px 1fr; gap: 10px; align-items: center; margin-bottom: 8px; }
.wizard-actions { display: flex; justify-content: space-between; gap: 8px; margin-top: 10px; }
.progress { margin-top: 12px; display: grid; gap: 8px; }
.pb-top { display: flex; justify-content: space-between; font-size: 12px; color: var(--ag-muted); margin-bottom: 4px; }
.bar { height: 8px; border-radius: 999px; background: rgba(255, 255, 255, 0.08); overflow: hidden; }
.bar > span { display: block; height: 100%; background: linear-gradient(90deg, #5e72ff, #8f9bff); transition: width 260ms ease; }
.layout { display: grid; grid-template-columns: 300px minmax(0, 1fr); gap: 12px; align-items: start; }
.sidebar { padding: 12px; position: sticky; top: 16px; max-height: calc(100vh - 140px); overflow: auto; }
.side-head { margin-bottom: 10px; }
.side-title { margin: 0 0 4px; font-size: 15px; }
.filters { display: grid; gap: 8px; margin-bottom: 10px; }
.vlist { display: grid; gap: 8px; }
.vcard { text-align: left; border: 1px solid var(--ag-border); background: rgba(255, 255, 255, 0.03); border-radius: 10px; padding: 10px; color: var(--ag-text); cursor: pointer; }
.vcard.active { border-color: var(--ag-accent); background: rgba(77, 107, 254, 0.14); }
.vname { margin: 0 0 4px; font-weight: 600; }
.details { padding: 14px; min-height: 520px; }
.dtop { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.badge { margin: 0; color: var(--ag-accent); font-size: 12px; }
.dtitle { margin: 2px 0 4px; font-size: 22px; }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
.stats-grid, .stats { display: grid; grid-template-columns: repeat(3, minmax(120px, 1fr)); gap: 8px; margin: 10px 0 16px; }
.stat { border: 1px solid var(--ag-border); border-radius: 10px; padding: 10px; background: rgba(255, 255, 255, 0.02); }
.n { margin: 4px 0 0; font-size: 20px; font-weight: 700; }
.preview-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 10px; }
.preview-head h4 { margin: 0; }
.view-all-btn { border: none; background: transparent; color: var(--ag-accent); cursor: pointer; font: inherit; }
.preview-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(86px, 1fr));
  gap: 8px;
  margin-bottom: 12px;
}
.all-grid { display: grid; grid-template-columns: repeat(10, minmax(0, 1fr)); gap: 8px; margin-bottom: 12px; }
.thumb.mini .ph { aspect-ratio: 4/3; }
.meta-block { border-top: 1px solid var(--ag-border); padding-top: 10px; }
.meta-block h4 { margin: 10px 0 4px; }
.all-images { margin-top: 6px; }
.all-tabs { display: flex; gap: 8px; margin-bottom: 10px; }
.tab-btn { border: 1px solid var(--ag-border); background: rgba(255,255,255,.03); color: var(--ag-text); border-radius: 8px; padding: 8px 10px; cursor: pointer; min-width: 88px; text-align: left; }
.tab-btn.active { border-color: var(--ag-accent); background: rgba(77,107,254,.14); }
.all-footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 8px; }
.thumb-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px; }
.thumb { text-decoration: none; color: inherit; position: relative; padding: 0; overflow: hidden; }
.ph { aspect-ratio: 4/3; background: rgba(0, 0, 0, 0.35); display: flex; align-items: center; justify-content: center; }
.stem { margin: 0; padding: 8px 10px; font-size: 11px; color: var(--ag-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.muted { color: var(--ag-muted); }
.small { font-size: 12px; }
.empty { padding: 8px 4px; }
.err { color: #ff7b7b; margin: 8px 0; }
@media (max-width: 1020px) {
  .layout { grid-template-columns: 1fr; }
  .sidebar { position: static; max-height: none; }
  .stats, .stats-grid { grid-template-columns: repeat(2, minmax(120px, 1fr)); }
  .all-grid { grid-template-columns: repeat(5, minmax(0, 1fr)); }
}
</style>
