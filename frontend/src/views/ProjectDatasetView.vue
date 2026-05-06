<template>
  <section class="page">
    <h2>Dataset</h2>
    <div class="toolbar ag-card">
      <div class="filter-item">
        <AgSelect v-model="split" :options="splitOptions" size="small" button-label="Split" @change="reload" />
      </div>
      <div class="filter-item">
        <AgSelect v-model="sortBy" :options="sortOptions" size="small" button-label="Sort" @change="reload" />
      </div>
      <div ref="classesRoot" class="filter-item classes-filter">
        <button type="button" class="classes-btn" @click="classesOpen = !classesOpen">Classes ▾</button>
        <div v-if="classesOpen" class="classes-menu">
          <div class="classes-head">
            <button type="button" class="mini-link" @click="toggleAllClasses">Toggle All</button>
            <button type="button" class="mini-link" @click="clearAllClasses">Clear All</button>
          </div>
          <div class="classes-row" v-for="it in classItems" :key="it.id">
            <button type="button" class="mark-btn" :class="{ on: it.mode === 'present' }" @click="setClassMode(it.id, it.mode === 'present' ? 'any' : 'present')">✓</button>
            <button type="button" class="mark-btn x" :class="{ on: it.mode === 'absent' }" @click="setClassMode(it.id, it.mode === 'absent' ? 'any' : 'absent')">✕</button>
            <span class="cls-name">{{ it.label }}</span>
          </div>
        </div>
      </div>
      <input
        v-model.trim="q"
        class="ag-input search-input"
        :class="{ compact: hasSelectionMode }"
        type="search"
        placeholder="Search filename"
        @input="reload"
      />
      <button
        v-if="hasSelectionMode"
        type="button"
        class="ag-btn ag-btn-ghost danger-btn"
        :disabled="deleting || selectedCount === 0"
        @click="deleteSelected"
      >
        {{ deleting ? 'Deleting...' : `Delete (${selectedCount})` }}
      </button>
    </div>
    <div class="thumb-grid-wrap">
      <div v-if="isLoading" class="loading-overlay">
        <span>Loading ...</span>
      </div>
      <div class="thumb-grid">
      <router-link
        v-for="(im, ix) in images"
        :key="im.id"
        class="thumb ag-card"
        @mouseenter="hoveredImageId = im.id"
        @mouseleave="hoveredImageId = ''"
        :to="{
          path: `/annotate/${im.id}`,
          query: {
            from: route.fullPath,
            ids: imageIdsParam,
            ix: String(ix),
            wid: String(route.params.wid || ''),
            pid: String(route.params.pid || ''),
          },
        }"
      >
        <button
          v-if="showImageCheckbox(im.id)"
          type="button"
          class="pick-check"
          :class="{ on: isSelected(im.id) }"
          @click.prevent.stop="toggleSelected(im.id)"
        >
          <span v-if="isSelected(im.id)">✓</span>
        </button>
        <div class="ph">
          <DatasetThumb :image-id="im.id" :boxes="im.boxes || []" />
        </div>
        <p class="stem">{{ im.stem }}</p>
      </router-link>
      </div>
    </div>
    <div class="pager">
      <label class="per-page">
        <span class="muted">Images per page:</span>
        <AgSelect v-model="perPage" :options="perPageOptions" size="small" class="per-select" @change="onPerPageChange" />
      </label>
      <div class="center-nav">
        <button class="sq" type="button" :disabled="isLoading || page <= 1" @click="prev">←</button>
        <span class="muted">{{ rangeStart }} - {{ rangeEnd }} of {{ total }}</span>
        <button class="sq" type="button" :disabled="isLoading || page >= totalPages" @click="next">→</button>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'
import DatasetThumb from '@/components/DatasetThumb.vue'
import AgSelect from '@/components/AgSelect.vue'

const api = useApi()
const route = useRoute()
const split = ref('all')
const q = ref('')
const page = ref(1)
const totalPages = ref(1)
const perPage = ref(50)
const total = ref(0)
const images = ref([])
const imageIdsParam = ref('')
const sortBy = ref('newest')
const classItems = ref([])
const classesOpen = ref(false)
const classesRoot = ref(null)
const isLoading = ref(true)
const selectedIds = ref([])
const hoveredImageId = ref('')
const deleting = ref(false)
let activeController = null
let reloadSeq = 0
const splitOptions = [
  { value: 'all', label: 'all' },
  { value: 'train', label: 'train' },
  { value: 'valid', label: 'valid' },
  { value: 'test', label: 'test' },
]
const sortOptions = [
  { value: 'newest', label: 'newest' },
  { value: 'oldest', label: 'oldest' },
  { value: 'objects_desc', label: 'objects: high to low' },
  { value: 'objects_asc', label: 'objects: low to high' },
]
const perPageOptions = [
  { value: 50, label: '50' },
  { value: 100, label: '100' },
  { value: 150, label: '150' },
  { value: 200, label: '200' },
]

async function reload() {
  const seq = ++reloadSeq
  if (activeController) activeController.abort()
  const controller = new AbortController()
  activeController = controller
  isLoading.value = true
  const qs = new URLSearchParams()
  qs.set('split', split.value)
  qs.set('page', String(page.value))
  qs.set('per_page', String(perPage.value))
  qs.set('sort', sortBy.value)
  const batch = String(route.query.batch || '').trim()
  if (batch) qs.set('batch', batch)
  if (q.value) qs.set('q', q.value)
  const present = classItems.value.filter((x) => x.mode === 'present').map((x) => x.id)
  const absent = classItems.value.filter((x) => x.mode === 'absent').map((x) => x.id)
  if (present.length) qs.set('class_present', present.join(','))
  if (absent.length) qs.set('class_absent', absent.join(','))
  try {
    const data = await api.request(`/projects/${route.params.pid}/images?${qs.toString()}`, { signal: controller.signal })
    if (seq !== reloadSeq) return
    images.value = data.images || []
    const onPage = new Set(images.value.map((x) => x.id))
    selectedIds.value = selectedIds.value.filter((id) => onPage.has(id))
    imageIdsParam.value = images.value.map((x) => x.id).join(',')
    total.value = Number(data.total || 0)
    page.value = Number(data.page || page.value || 1)
    perPage.value = Number(data.per_page || perPage.value || 50)
    totalPages.value = Math.max(1, data.total_pages || 1)
  } catch (err) {
    if (err?.name !== 'AbortError') throw err
  } finally {
    if (seq === reloadSeq) {
      isLoading.value = false
      if (activeController === controller) activeController = null
    }
  }
}

function prev() {
  if (page.value <= 1) return
  page.value -= 1
  reload()
}

function next() {
  if (page.value >= totalPages.value) return
  page.value += 1
  reload()
}

function onPerPageChange() {
  page.value = 1
  reload()
}

function setClassMode(id, mode) {
  const i = classItems.value.findIndex((x) => x.id === id)
  if (i < 0) return
  classItems.value[i] = { ...classItems.value[i], mode }
  page.value = 1
  reload()
}

function toggleAllClasses() {
  const allOn = classItems.value.every((x) => x.mode === 'present')
  classItems.value = classItems.value.map((x) => ({ ...x, mode: allOn ? 'any' : 'present' }))
  page.value = 1
  reload()
}

function clearAllClasses() {
  classItems.value = classItems.value.map((x) => ({ ...x, mode: 'any' }))
  page.value = 1
  reload()
}

function onDocPointer(e) {
  if (!classesRoot.value) return
  if (!classesRoot.value.contains(e.target)) classesOpen.value = false
}

const rangeStart = computed(() => {
  if (total.value === 0) return 0
  return (page.value - 1) * perPage.value + 1
})

const rangeEnd = computed(() => {
  if (total.value === 0) return 0
  return Math.min(total.value, rangeStart.value + images.value.length - 1)
})

const selectedCount = computed(() => selectedIds.value.length)
const hasSelectionMode = computed(() => selectedCount.value > 0)

function isSelected(id) {
  return selectedIds.value.includes(id)
}

function toggleSelected(id) {
  if (!id) return
  if (isSelected(id)) {
    selectedIds.value = selectedIds.value.filter((x) => x !== id)
    return
  }
  selectedIds.value = [...selectedIds.value, id]
}

function showImageCheckbox(id) {
  return hasSelectionMode.value || hoveredImageId.value === id
}

async function deleteSelected() {
  if (!selectedIds.value.length || deleting.value) return
  deleting.value = true
  try {
    await api.request(`/projects/${route.params.pid}/images/delete`, {
      method: 'POST',
      body: JSON.stringify({ ids: selectedIds.value }),
    })
    selectedIds.value = []
    await reload()
  } finally {
    deleting.value = false
  }
}

onMounted(async () => {
  try {
    const s = await api.request(`/projects/${route.params.pid}/class-stats`)
    classItems.value = (s?.classes || []).map((c) => ({
      id: String(c.class_id),
      label: c.name || 'class',
      mode: 'any',
    }))
  } catch {}
  document.addEventListener('pointerdown', onDocPointer, true)
  await reload()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointer, true)
  if (activeController) activeController.abort()
})
</script>

<style scoped>
.page {
  padding: 20px 22px;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.muted { color: var(--ag-muted); }
.toolbar {
  margin: 0 28px 12px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  position: relative;
  z-index: 200;
  font-size: 12px;
}
.filter-item {
  display: grid;
  min-width: 120px;
}
.search-input {
  flex: 1 1 280px;
  min-width: 220px;
  min-height: 30px;
  height: 30px;
  padding: 4px 10px;
  border-radius: 10px;
  font-size: 12px;
}
.search-input.compact {
  flex: 1 1 220px;
}
.danger-btn {
  min-height: 30px;
  height: 30px;
  font-size: 12px;
  padding: 0 12px;
}
.classes-filter { position: relative; }
.classes-btn {
  min-height: 34px;
  border: none;
  border-radius: 10px;
  background: rgba(255,255,255,.06);
  color: var(--ag-text);
  padding: 7px 10px;
  text-align: left;
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}
.classes-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  width: max-content;
  min-width: 100%;
  max-width: min(80vw, 760px);
  z-index: 9999;
  background: #242426;
  border: 1px solid var(--ag-border);
  border-radius: 12px;
  max-height: 280px;
  overflow: auto;
  padding: 6px;
}
.classes-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 4px 8px;
  border-bottom: 1px solid var(--ag-border);
  margin-bottom: 6px;
}
.mini-link {
  border: none;
  background: transparent;
  color: var(--ag-accent);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}
.classes-row {
  display: grid;
  grid-template-columns: 24px 24px 1fr;
  align-items: center;
  gap: 8px;
  padding: 4px;
}
.mark-btn {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: 1px solid var(--ag-border);
  background: rgba(255,255,255,.04);
  color: var(--ag-muted);
  cursor: pointer;
  font-size: 12px;
}
.mark-btn.on { background: rgba(77,107,254,.28); color: #fff; border-color: rgba(77,107,254,.45); }
.mark-btn.x.on { background: rgba(255,123,123,.22); border-color: rgba(255,123,123,.45); }
.cls-name {
  font-size: 12px;
  color: var(--ag-text);
  white-space: nowrap;
}
.thumb-grid-wrap {
  position: relative;
  padding: 0 28px;
  flex: 1 1 auto;
  min-height: clamp(360px, 62vh, 780px);
}
.thumb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
  position: relative;
  z-index: 1;
}
.loading-overlay {
  position: absolute;
  inset: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 17, 24, 0.45);
  border-radius: 12px;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.2px;
}
.thumb { text-decoration: none; color: inherit; position: relative; padding: 0; overflow: hidden; }
.pick-check {
  position: absolute;
  top: 8px;
  left: 8px;
  width: 18px;
  height: 18px;
  border-radius: 5px;
  border: 1px solid rgba(255, 255, 255, 0.8);
  background: rgba(10, 12, 20, 0.5);
  color: #fff;
  z-index: 4;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  cursor: pointer;
}
.pick-check.on {
  background: rgba(77, 107, 254, 0.9);
  border-color: rgba(77, 107, 254, 1);
}
.ph { aspect-ratio: 4/3; background: rgba(0,0,0,.35); display: flex; align-items: center; justify-content: center; }
.stem { margin: 0; padding: 8px 10px; font-size: 11px; color: var(--ag-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pager { margin-top: 14px; display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 10px; }
.per-page { display: inline-flex; align-items: center; gap: 7px; font-size: 12px; }
.per-select { width: 78px; }
.center-nav { justify-self: center; display: inline-flex; align-items: center; gap: 8px; font-size: 12px; }
.sq { width: 30px; height: 30px; border: 1px solid var(--ag-border); border-radius: 8px; background: rgba(255,255,255,.03); color: var(--ag-text); cursor: pointer; font-size: 13px; }
.sq:disabled { opacity: .45; cursor: default; }
@media (max-width: 1080px) {
  .search-input {
    flex: 1 1 100%;
    min-width: 0;
  }
}
@media (max-width: 760px) { .pager { grid-template-columns: 1fr; } .center-nav { justify-self: start; } }
</style>
