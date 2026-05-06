<template>
  <div class="page">
    <AgHeader />
    <div class="shell" v-if="meta">
      <div class="breadcrumbs row-flex">
        <router-link class="ag-link" :to="`/projects/${meta.project_id}`">← Project</router-link>
        <button type="button" class="ag-btn danger" @click="deleteVersion" :disabled="deleting">
          {{ deleting ? '…' : 'Delete version' }}
        </button>
        <button type="button" class="ag-btn dl" @click="downloadZip" :disabled="downloading">
          {{ downloading ? '…' : 'Download dataset (.zip)' }}
        </button>
      </div>
      <p v-if="downloadErr" class="err global-msg">{{ downloadErr }}</p>
      <p v-if="deleteErr" class="err global-msg">{{ deleteErr }}</p>

      <h1>{{ meta.name }}</h1>

      <div v-if="chips.length" class="chips">
        <span v-for="(c, i) in chips" :key="i" class="ag-pill sm">{{ c }}</span>
      </div>

      <div class="ag-card yaml-block">
        <div class="yaml-head row-flex">
          <h2 class="yaml-title">data.yaml</h2>
          <button
            type="button"
            class="icon-btn"
            :aria-pressed="yamlEditOpen"
            aria-label="Edit data.yaml"
            title="Edit data.yaml"
            @click="toggleYamlEdit"
          >
            <svg class="svg-ic" viewBox="0 0 24 24" aria-hidden="true">
              <path
                fill="currentColor"
                d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM21.71 7.04a1 1 0 0 0 0-1.41l-2.34-2.34a1 1 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"
              />
            </svg>
          </button>
        </div>
        <div v-show="yamlEditOpen" class="yaml-editor">
          <p v-if="yamlErr" class="err">{{ yamlErr }}</p>
          <textarea v-model="yamlDraft" class="ag-input mono yaml-edit" spellcheck="false" />
          <button type="button" class="ag-btn ag-btn-primary sm save-yaml" @click="saveYaml" :disabled="yamlSaving">
            {{ yamlSaving ? '…' : 'Save YAML' }}
          </button>
        </div>
      </div>

      <div class="tabs">
        <button
          type="button"
          v-for="s in splits"
          :key="s"
          class="tab"
          :class="{ active: split === s }"
          @click="setSplit(s)"
        >
          {{ s }}
        </button>
      </div>

      <div class="gallery-toolbar row-flex">
        <button
          type="button"
          class="icon-btn"
          :class="{ 'icon-btn-active': filtersOpen }"
          aria-label="Filters"
          title="Filters"
          @click="filtersOpen = !filtersOpen"
        >
          <svg class="svg-ic" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"
            />
          </svg>
        </button>
        <div class="spacer" />
        <label class="per-page lbl-inline">
          <span class="per-page-label">Per page</span>
          <select v-model.number="perPage" class="ag-input per-page-select" @change="onPerPageChange">
            <option :value="50">50</option>
            <option :value="100">100</option>
            <option :value="150">150</option>
            <option :value="200">200</option>
          </select>
        </label>
      </div>

      <div v-show="filtersOpen" class="ag-card filters">
        <div class="filter-grid">
          <label class="fl">
            <span class="flab">Filename</span>
            <input v-model="filt.q" class="ag-input" type="search" placeholder="search" />
          </label>
          <label class="fl">
            <span class="flab">Labels</span>
            <select v-model="filt.anno" class="ag-input">
              <option value="">all</option>
              <option value="yes">with objects</option>
              <option value="no">empty</option>
            </select>
          </label>
          <label class="fl">
            <span class="flab">Width</span>
            <div class="pair">
              <input v-model="filt.widthMin" class="ag-input" type="number" :placeholder="ph('width', 'min')" />
              <input v-model="filt.widthMax" class="ag-input" type="number" :placeholder="ph('width', 'max')" />
            </div>
          </label>
          <label class="fl">
            <span class="flab">Height</span>
            <div class="pair">
              <input v-model="filt.heightMin" class="ag-input" type="number" :placeholder="ph('height', 'min')" />
              <input v-model="filt.heightMax" class="ag-input" type="number" :placeholder="ph('height', 'max')" />
            </div>
          </label>
          <label class="fl">
            <span class="flab">Objects (bbox)</span>
            <div class="pair">
              <input v-model="filt.bboxMin" class="ag-input" type="number" :placeholder="ph('bbox_count', 'min')" />
              <input v-model="filt.bboxMax" class="ag-input" type="number" :placeholder="ph('bbox_count', 'max')" />
            </div>
          </label>
          <label class="fl">
            <span class="flab">Aspect ratio W/H</span>
            <div class="pair">
              <input v-model="filt.aspectMin" class="ag-input" type="number" step="0.01" placeholder="min" />
              <input v-model="filt.aspectMax" class="ag-input" type="number" step="0.01" placeholder="max" />
            </div>
          </label>
          <label class="fl">
            <span class="flab">Megapixels</span>
            <div class="pair">
              <input v-model="filt.mpMin" class="ag-input" type="number" step="0.01" placeholder="min" />
              <input v-model="filt.mpMax" class="ag-input" type="number" step="0.01" placeholder="max" />
            </div>
          </label>
        </div>
        <button type="button" class="ag-btn" @click="applyFiltersNow">Apply filters</button>
      </div>

      <div class="thumb-grid">
        <router-link
          v-for="(im, idx) in images"
          :key="im.id"
          class="thumb ag-card"
          :to="annotateLink(im.id, idx)"
        >
          <div class="ph">
            <DatasetThumb :image-id="im.id" :boxes="im.boxes || []" />
          </div>
          <p class="stem">{{ im.stem }}</p>
          <span v-if="im.bbox_count > 0" class="lbl">{{ im.bbox_count }}</span>
        </router-link>
      </div>
      <p v-if="loaded && !images.length" class="muted small">No images for current filters.</p>

      <nav class="pager pager-bottom" v-if="total > 0">
        <div class="pager-row row-flex">
          <button type="button" class="ag-btn" :disabled="page <= 1" @click="goPrev">Back</button>
          <div class="page-nums row-flex">
            <button
              v-for="p in pageNums"
              :key="p"
              type="button"
              class="pg"
              :class="{ active: p === page }"
              @click="goPage(p)"
            >
              {{ p }}
            </button>
          </div>
          <button type="button" class="ag-btn" :disabled="page >= lastPage" @click="goNext">Next</button>
        </div>
        <p class="muted small pager-meta">{{ page }} of {{ lastPage }} · total {{ total }} images</p>
      </nav>
    </div>
    <div v-else class="shell muted">Loading ...</div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AgHeader from '@/components/AgHeader.vue'
import DatasetThumb from '@/components/DatasetThumb.vue'
import { useApi, getToken } from '@/composables/useApi'
import { parseDatasetClassNames } from '@/utils/yoloLabel'

const api = useApi()
const route = useRoute()
const router = useRouter()
const vid = computed(() => route.params.vid)

const meta = ref(null)
const yamlDraft = ref('')
const yamlErr = ref('')
const downloadErr = ref('')
const deleteErr = ref('')
const yamlSaving = ref(false)
const yamlEditOpen = ref(false)
const filtersOpen = ref(false)
const images = ref([])
const total = ref(0)
const page = ref(1)
const perPage = ref(50)
const split = ref('train')
const loaded = ref(false)
const splits = ['train', 'valid', 'test']
const stats = ref(null)
const downloading = ref(false)
const deleting = ref(false)
let listBoot = false

const filt = reactive({
  q: '',
  anno: '',
  widthMin: '',
  widthMax: '',
  heightMin: '',
  heightMax: '',
  bboxMin: '',
  bboxMax: '',
  aspectMin: '',
  aspectMax: '',
  mpMin: '',
  mpMax: '',
})

const chips = computed(() => parseDatasetClassNames(meta.value?.data_yaml || ''))

const lastPage = ref(1)

const pageNums = computed(() => {
  const tp = Math.max(1, lastPage.value)
  const cur = Math.min(Math.max(1, page.value), tp)
  const max = 9
  if (tp <= max) return Array.from({ length: tp }, (_, i) => i + 1)
  let start = cur - Math.floor(max / 2)
  let end = start + max - 1
  if (start < 1) {
    start = 1
    end = max
  }
  if (end > tp) {
    end = tp
    start = Math.max(1, end - max + 1)
  }
  return Array.from({ length: end - start + 1 }, (_, i) => start + i)
})

let debTimer = 0

function toggleYamlEdit() {
  yamlEditOpen.value = !yamlEditOpen.value
  yamlErr.value = ''
  if (yamlEditOpen.value && meta.value) {
    yamlDraft.value = meta.value.data_yaml || ''
  }
}

function ph(dim, kind) {
  const s = stats.value
  if (!s || !s[dim]) return ''
  const v = s[dim][kind === 'min' ? 'min' : 'max']
  return v !== undefined && v !== null ? String(v) : ''
}

function buildQuery() {
  const u = new URLSearchParams()
  u.set('split', split.value)
  u.set('page', String(page.value))
  u.set('per_page', String(perPage.value))
  const q = filt.q.trim()
  if (q) u.set('q', q)
  if (filt.anno) u.set('anno', filt.anno)
  const nmi = parseInt(String(filt.widthMin), 10)
  const nmx = parseInt(String(filt.widthMax), 10)
  const hmi = parseInt(String(filt.heightMin), 10)
  const hmx = parseInt(String(filt.heightMax), 10)
  const bmi = parseInt(String(filt.bboxMin), 10)
  const bmx = parseInt(String(filt.bboxMax), 10)
  if (!Number.isNaN(nmi) && filt.widthMin !== '') u.set('width_min', String(nmi))
  if (!Number.isNaN(nmx) && filt.widthMax !== '') u.set('width_max', String(nmx))
  if (!Number.isNaN(hmi) && filt.heightMin !== '') u.set('height_min', String(hmi))
  if (!Number.isNaN(hmx) && filt.heightMax !== '') u.set('height_max', String(hmx))
  if (!Number.isNaN(bmi) && filt.bboxMin !== '') u.set('bbox_min', String(bmi))
  if (!Number.isNaN(bmx) && filt.bboxMax !== '') u.set('bbox_max', String(bmx))
  const ami = parseFloat(String(filt.aspectMin))
  const amx = parseFloat(String(filt.aspectMax))
  const mmi = parseFloat(String(filt.mpMin))
  const mmx = parseFloat(String(filt.mpMax))
  if (!Number.isNaN(ami) && filt.aspectMin !== '') u.set('aspect_min', String(ami))
  if (!Number.isNaN(amx) && filt.aspectMax !== '') u.set('aspect_max', String(amx))
  if (!Number.isNaN(mmi) && filt.mpMin !== '') u.set('mp_min', String(mmi))
  if (!Number.isNaN(mmx) && filt.mpMax !== '') u.set('mp_max', String(mmx))
  return u.toString()
}

async function loadStats() {
  try {
    stats.value = await api.request(`/versions/${vid.value}/split-stats?split=${split.value}`)
  } catch {
    stats.value = null
  }
}

async function fetchPage() {
  loaded.value = false
  try {
    const res = await api.request(`/versions/${vid.value}/images?${buildQuery()}`)
    images.value = res.images || []
    const tot = Number(res.total)
    total.value = Number.isFinite(tot) ? tot : 0
    const ppUsed = typeof res.per_page === 'number' && res.per_page > 0 ? res.per_page : perPage.value
    lastPage.value = Math.max(
      1,
      typeof res.total_pages === 'number'
        ? Math.floor(res.total_pages)
        : Math.ceil(total.value / (ppUsed || 1)),
    )
    const srv = typeof res.page === 'number' ? Math.floor(res.page) : page.value
    page.value = Math.min(Math.max(1, srv), lastPage.value)
  } catch {
    images.value = []
    total.value = 0
    lastPage.value = 1
  } finally {
    loaded.value = true
  }
}

function scheduleLoad() {
  if (!listBoot) return
  clearTimeout(debTimer)
  debTimer = setTimeout(() => {
    page.value = 1
    fetchPage()
  }, 280)
}

function applyFiltersNow() {
  clearTimeout(debTimer)
  page.value = 1
  fetchPage()
}

function onPerPageChange() {
  page.value = 1
  fetchPage()
}

function goPrev() {
  if (page.value <= 1) return
  page.value -= 1
  fetchPage()
}

function goNext() {
  if (page.value >= lastPage.value) return
  page.value += 1
  fetchPage()
}

function goPage(p) {
  const n = Number(p)
  if (!Number.isFinite(n)) return
  const t = Math.floor(n)
  if (t < 1 || t > lastPage.value) return
  if (page.value === t) return
  page.value = t
  fetchPage()
}

function annotateLink(imageId, idx) {
  const ids = images.value.map((x) => x.id).filter(Boolean).join(',')
  return {
    path: `/annotate/${imageId}`,
    query: {
      from: route.fullPath,
      ids,
      ix: String(idx),
      source: 'version',
      vid: String(vid.value || ''),
    },
  }
}

async function setSplit(s) {
  split.value = s
  page.value = 1
  await loadStats()
  await fetchPage()
}

async function saveYaml() {
  yamlErr.value = ''
  yamlSaving.value = true
  try {
    const r = await api.request(`/versions/${vid.value}/data-yaml`, {
      method: 'PUT',
      body: JSON.stringify({ data_yaml: yamlDraft.value }),
    })
    if (meta.value) meta.value.data_yaml = r.data_yaml ?? yamlDraft.value
  } catch (e) {
    yamlErr.value = e.message || 'error'
  } finally {
    yamlSaving.value = false
  }
}

async function downloadZip() {
  downloadErr.value = ''
  downloading.value = true
  try {
    const t = getToken()
    const res = await fetch(`/api/versions/${vid.value}/dataset.zip`, {
      headers: t ? { Authorization: `Bearer ${t}` } : {},
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
    downloadErr.value = e.message || 'download failed'
  } finally {
    downloading.value = false
  }
}

async function deleteVersion() {
  deleteErr.value = ''
  if (!meta.value?.project_id) return
  deleting.value = true
  try {
    await api.request(`/versions/${vid.value}`, { method: 'DELETE' })
    await router.push(`/projects/${meta.value.project_id}`)
  } catch (e) {
    deleteErr.value = e.message || 'failed to delete version'
  } finally {
    deleting.value = false
  }
}

watch(
  () => ({ ...filt }),
  () => scheduleLoad(),
  { deep: true },
)

watch(
  () => route.params.vid,
  async (newVid, oldVid) => {
    if (!listBoot || oldVid === undefined || !newVid) return
    if (String(newVid) === String(oldVid)) return
    yamlEditOpen.value = false
    yamlDraft.value = ''
    filtersOpen.value = false
    page.value = 1
    perPage.value = 50
    split.value = 'train'
    try {
      meta.value = await api.request(`/versions/${newVid}`)
      yamlDraft.value = meta.value?.data_yaml || ''
      await loadStats()
      await fetchPage()
    } catch (_) {
      loaded.value = true
    }
  },
)

onMounted(async () => {
  try {
    page.value = 1
    perPage.value = 50
    meta.value = await api.request(`/versions/${vid.value}`)
    yamlDraft.value = meta.value.data_yaml || ''
    await loadStats()
    await fetchPage()
  } catch {
    loaded.value = true
  } finally {
    listBoot = true
  }
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
  padding: 22px;
}

h1 {
  margin: 0 0 12px;
}

.row-flex {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.dl {
  margin-left: auto;
}

.danger {
  border-color: rgba(255, 123, 123, 0.55);
  color: #ff9a9a;
}

.danger:hover {
  border-color: #ff7b7b;
  color: #ffd0d0;
}

.global-msg {
  margin: 0 0 12px;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.sm.ag-pill {
  font-size: 11px;
}

.yaml-block {
  margin-bottom: 16px;
  padding: 12px 14px;
}

.yaml-title {
  margin: 0;
  flex: 1;
  font-size: 15px;
}

.yaml-head {
  margin-bottom: 0;
  align-items: center;
}

.yaml-editor {
  margin-top: 12px;
}

.save-yaml {
  margin-top: 10px;
}

.yaml-edit {
  width: 100%;
  min-height: 160px;
  font-size: 12px;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  padding: 0;
  border-radius: 10px;
  border: 1px solid var(--ag-border);
  background: rgba(255, 255, 255, 0.04);
  color: var(--ag-muted);
  cursor: pointer;
  transition:
    border-color var(--ag-duration),
    color var(--ag-duration),
    background var(--ag-duration);
}

.icon-btn:hover {
  border-color: var(--ag-accent);
  color: var(--ag-accent);
}

.icon-btn-active {
  border-color: var(--ag-accent);
  color: var(--ag-accent);
  background: rgba(77, 107, 254, 0.12);
}

.svg-ic {
  width: 22px;
  height: 22px;
  display: block;
}

.sm.ag-btn {
  padding: 6px 12px;
  font-size: 13px;
}

.tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.tab {
  font-family: inherit;
  cursor: pointer;
  border-radius: 10px;
  border: 1px solid var(--ag-border);
  background: transparent;
  color: var(--ag-muted);
  padding: 8px 16px;
  font-weight: 600;
  transition:
    border-color var(--ag-duration),
    color var(--ag-duration),
    background var(--ag-duration);
}

.tab.active {
  border-color: var(--ag-accent);
  color: var(--ag-text);
  background: rgba(77, 107, 254, 0.16);
}

.gallery-toolbar {
  margin-bottom: 12px;
  align-items: center;
}

.spacer {
  flex: 1;
  min-width: 8px;
}

.lbl-inline {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
}

.per-page-label {
  font-size: 13px;
  color: var(--ag-muted);
  white-space: nowrap;
}

.per-page-select {
  min-width: 88px;
  width: auto;
}

.filters {
  margin-bottom: 14px;
  padding: 14px;
}

.filter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.fl {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.flab {
  font-size: 11px;
  color: var(--ag-muted);
}

.pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}

.pager-bottom {
  margin-top: 20px;
}

.pager-row {
  justify-content: center;
  flex-wrap: wrap;
  gap: 10px;
}

.page-nums {
  gap: 4px;
  flex-wrap: wrap;
  justify-content: center;
}

.pg {
  min-width: 36px;
  height: 36px;
  padding: 0 8px;
  font-family: inherit;
  font-size: 13px;
  border-radius: 8px;
  border: 1px solid var(--ag-border);
  background: rgba(255, 255, 255, 0.04);
  color: var(--ag-muted);
  cursor: pointer;
  transition:
    border-color var(--ag-duration),
    color var(--ag-duration),
    background var(--ag-duration);
}

.pg:hover {
  border-color: var(--ag-accent);
  color: var(--ag-accent);
}

.pg.active {
  border-color: var(--ag-accent);
  color: var(--ag-text);
  background: rgba(77, 107, 254, 0.2);
}

.pager-meta {
  margin: 12px 0 0;
  text-align: center;
}

@media (max-width: 720px) {
  .pager-row {
    gap: 8px;
  }
}

.thumb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.thumb {
  text-decoration: none;
  color: inherit;
  position: relative;
  padding: 0;
  overflow: hidden;
}

.ph {
  aspect-ratio: 4/3;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
}

.stem {
  margin: 0;
  padding: 8px 10px;
  font-size: 11px;
  color: var(--ag-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.lbl {
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 11px;
  font-weight: 700;
  color: #4dffb5;
  text-shadow: 0 0 4px #000;
}

.err {
  color: #ff7b7b;
  font-size: 13px;
  margin: 0 0 8px;
}

.muted {
  color: var(--ag-muted);
}

.small {
  font-size: 13px;
}
</style>
