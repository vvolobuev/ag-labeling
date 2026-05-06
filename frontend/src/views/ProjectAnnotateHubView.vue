<template>
  <section class="page">
    <h2>Annotate</h2>
    <div v-if="activeBatch" class="ag-card batch-workspace">
      <div class="batch-top">
        <button class="ag-btn ag-btn-ghost sm" type="button" @click="closeBatch">← Back to Batches</button>
        <p class="batch-title">{{ activeBatch }}</p>
        <button
          v-if="annotatedTotal > 0"
          class="ag-btn ag-btn-primary sm batch-add-btn"
          type="button"
          @click="addAnnotated(activeBatch)"
        >
          Add Annotated to Dataset
        </button>
      </div>
      <div class="tabs">
        <button
          type="button"
          class="tab-btn"
          :class="{ active: tab === 'unannotated' }"
          @click="tab = 'unannotated'"
        >
          Unannotated ({{ unannotatedTotal }})
        </button>
        <button
          type="button"
          class="tab-btn"
          :class="{ active: tab === 'annotated' }"
          @click="tab = 'annotated'"
        >
          Annotated ({{ annotatedTotal }})
        </button>
      </div>
      <div class="thumb-grid">
        <router-link
          v-for="(im, ix) in tabImages"
          :key="im.id"
          class="thumb ag-card"
          :to="{
            path: `/annotate/${im.id}`,
            query: {
              from: route.fullPath,
              ids: tabIds,
              ix: String(ix),
              wid: String(route.params.wid || ''),
              pid: String(route.params.pid || ''),
            },
          }"
        >
          <div class="ph">
            <DatasetThumb :image-id="im.id" :boxes="im.boxes || []" />
          </div>
          <p class="stem">{{ im.stem }}</p>
        </router-link>
      </div>
      <p v-if="!tabImages.length" class="muted mt">No images in this tab.</p>
    </div>
    <div v-if="!activeBatch" class="cols">
      <section class="col ag-card">
        <h3>Unassigned</h3>
        <p class="muted">{{ (board.unassigned || []).length }} Batches</p>
        <router-link class="ag-link" :to="uploadLink">Upload More Images</router-link>
        <article v-for="b in board.unassigned || []" :key="`u-${b.batch_name}-${b.uploaded_at}`" class="batch-card">
          <button class="link-batch" type="button" @click="startBatch(b.batch_name)">{{ b.batch_name }}</button>
          <p class="muted">{{ b.image_count }} Images</p>
          <div class="row-btns">
            <button class="ag-btn ag-btn-primary sm" type="button" @click="startBatch(b.batch_name)">Start Annotating</button>
            <button class="ag-btn ag-btn-ghost sm danger" type="button" @click="removeBatch(b.batch_name)">Delete Batch</button>
          </div>
        </article>
      </section>

      <section class="col ag-card">
        <h3>Annotating</h3>
        <p class="muted">{{ (board.annotating || []).length }} Jobs</p>
        <p class="muted">Upload and assign images to an annotator.</p>
        <article v-for="b in board.annotating || []" :key="`a-${b.batch_name}-${b.uploaded_at}`" class="batch-card">
          <p class="nm">{{ b.batch_name }}</p>
          <p class="muted">Annotated: {{ b.annotated || 0 }} · Unannotated: {{ b.unannotated || 0 }}</p>
          <div class="row-btns">
            <button class="ag-btn ag-btn-ghost sm" type="button" @click="startBatch(b.batch_name)">Continue</button>
            <button class="ag-btn ag-btn-primary sm" type="button" @click="addAnnotated(b.batch_name)">Add Annotated to Dataset</button>
            <button class="ag-btn ag-btn-ghost sm danger" type="button" @click="removeBatch(b.batch_name)">Delete Batch</button>
          </div>
        </article>
      </section>

      <section class="col ag-card">
        <h3>Dataset</h3>
        <p class="muted">{{ (board.dataset || []).length }} Jobs</p>
        <router-link class="ag-link" :to="datasetLink">See all images</router-link>
        <article v-for="b in board.dataset || []" :key="`d-${b.batch_name}-${b.uploaded_at}`" class="batch-card dataset-card">
          <p class="nm">{{ b.batch_name }}</p>
          <p class="muted">Added to Dataset: {{ b.image_count }} images</p>
          <p class="muted">Labeler: {{ b.labeler_email || '-' }}</p>
          <p class="muted">Uploaded {{ fmtLong(b.uploaded_at) }}</p>
        </article>
      </section>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApi } from '@/composables/useApi'
import DatasetThumb from '@/components/DatasetThumb.vue'
const route = useRoute()
const router = useRouter()
const api = useApi()
const uploadLink = computed(() => `/app/${route.params.wid}/projects/${route.params.pid}/upload`)
const datasetLink = computed(() => `/app/${route.params.wid}/projects/${route.params.pid}/dataset`)
const board = ref({ unassigned: [], annotating: [], dataset: [] })
const activeBatch = computed(() => String(route.query.batch || '').trim())
const tab = ref('unannotated')
const unannotated = ref([])
const annotated = ref([])
const unannotatedTotal = ref(0)
const annotatedTotal = ref(0)
const tabImages = computed(() => (tab.value === 'annotated' ? annotated.value : unannotated.value))
const tabIds = computed(() => tabImages.value.map((x) => x.id).join(','))

async function loadBoard() {
  board.value = await api.request(`/projects/${route.params.pid}/batches`)
}

function fmtShort(ts) {
  if (!ts) return '-'
  return new Date(Number(ts) * 1000).toLocaleString()
}

function fmtLong(ts) {
  if (!ts) return '-'
  return new Date(Number(ts) * 1000).toLocaleString()
}

async function startBatch(batchName) {
  await router.push({
    path: `/app/${route.params.wid}/projects/${route.params.pid}/annotate`,
    query: { batch: batchName },
  })
}

async function addAnnotated(batchName) {
  const res = await api.request(`/projects/${route.params.pid}/batches/${encodeURIComponent(batchName)}/add-annotated`, {
    method: 'POST',
  })
  await loadBoard()
  if (activeBatch.value && String(activeBatch.value) === String(batchName)) {
    await loadBatchImages()
  }
  if (Number(res?.moved || 0) === 0) {
    alert('No annotated images were moved. Save labels first.')
  }
}

async function removeBatch(batchName) {
  const name = String(batchName || '').trim()
  if (!name) return
  await api.request(`/projects/${route.params.pid}/batches/${encodeURIComponent(name)}`, { method: 'DELETE' })
  if (activeBatch.value && String(activeBatch.value) === name) {
    await closeBatch()
  } else {
    await loadBoard()
  }
}

onMounted(async () => {
  await loadBoard()
  await loadBatchImages()
})

async function loadBatchImages() {
  if (!activeBatch.value) {
    unannotated.value = []
    annotated.value = []
    unannotatedTotal.value = 0
    annotatedTotal.value = 0
    return
  }
  const [u, a] = await Promise.all([
    api.request(`/projects/${route.params.pid}/batches/${encodeURIComponent(activeBatch.value)}/images?anno=no`),
    api.request(`/projects/${route.params.pid}/batches/${encodeURIComponent(activeBatch.value)}/images?anno=yes`),
  ])
  unannotated.value = u.images || []
  annotated.value = a.images || []
  unannotatedTotal.value = Number(u.total || 0)
  annotatedTotal.value = Number(a.total || 0)
}

async function closeBatch() {
  await router.push({ path: `/app/${route.params.wid}/projects/${route.params.pid}/annotate` })
  await loadBoard()
}

watch(
  () => route.query.batch,
  async () => {
    tab.value = 'unannotated'
    await loadBatchImages()
    if (!activeBatch.value) await loadBoard()
  },
  { immediate: false },
)
</script>

<style scoped>
.page { padding: 20px 22px; }
.batch-workspace { margin-bottom: 12px; padding: 12px; }
.batch-top { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.batch-title { margin: 0; font-weight: 600; }
.batch-add-btn { margin-left: auto; }
.tabs { display: inline-flex; gap: 6px; margin-bottom: 12px; }
.tab-btn { border: 1px solid var(--ag-border); background: rgba(255,255,255,.03); color: var(--ag-muted); border-radius: 8px; padding: 6px 10px; cursor: pointer; font-size: 12px; }
.tab-btn.active { color: var(--ag-text); border-color: var(--ag-accent); background: rgba(77,107,254,.14); }
.thumb-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px; }
.thumb { text-decoration: none; color: inherit; position: relative; padding: 0; overflow: hidden; }
.ph { aspect-ratio: 4/3; background: rgba(0,0,0,.35); display: flex; align-items: center; justify-content: center; }
.stem { margin: 0; padding: 8px 10px; font-size: 11px; color: var(--ag-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.mt { margin-top: 10px; }
.cols { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.col { min-height: 420px; }
h3 { margin: 0 0 6px; }
.muted { color: var(--ag-muted); }
.batch-card { margin-top: 10px; padding: 10px; border: 1px solid var(--ag-border); border-radius: 10px; background: rgba(255,255,255,.02); }
.dataset-card { margin-top: 10px; }
.dataset-card .nm { font-size: 14px; margin-bottom: 4px; }
.dataset-card .muted { font-size: 13px; line-height: 1.4; }
.nm { margin: 0 0 4px; font-weight: 600; }
.link-batch { border: none; background: transparent; color: var(--ag-text); font-weight: 600; padding: 0; cursor: pointer; }
.link-batch:hover { color: var(--ag-accent); }
.sm.ag-btn { margin-top: 8px; padding: 7px 10px; font-size: 12px; }
.row-btns { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.danger { border-color: rgba(255,123,123,.55); color: #ff9a9a; }
@media (max-width: 1100px) { .cols { grid-template-columns: 1fr; } }
</style>
