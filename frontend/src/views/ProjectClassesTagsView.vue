<template>
  <section class="page">
    <h2>Classes & Tags</h2>
    <div v-if="loading" class="loading-wrap">
      <p class="muted loading-text">Loading ...</p>
    </div>
    <div class="grid" v-if="stats">
      <article class="ag-card tile">
        <p class="muted small">Total Images</p>
        <p class="n">{{ stats.summary?.total_images || 0 }}</p>
      </article>
      <article class="ag-card tile">
        <p class="muted small">Total Classes</p>
        <p class="n">{{ stats.summary?.total_classes || 0 }}</p>
      </article>
      <article class="ag-card tile">
        <p class="muted small">Unannotated Images</p>
        <p class="n">{{ stats.summary?.unannotated_images || 0 }}</p>
      </article>
      <article class="ag-card tile">
        <p class="muted small">Avg Image Aspect (W/H)</p>
        <p class="n">{{ fmt(stats.image_aspect_ratio?.avg) }}</p>
      </article>
    </div>

    <div class="ag-card stats" v-if="stats">
      <h3>Split Distribution</h3>
      <div class="split-grid">
        <div>
          <p class="muted small">Train</p>
          <p>Images: {{ stats.split_distribution?.images?.train || 0 }}</p>
          <p>Objects: {{ stats.split_distribution?.objects?.train || 0 }}</p>
        </div>
        <div>
          <p class="muted small">Valid</p>
          <p>Images: {{ stats.split_distribution?.images?.valid || 0 }}</p>
          <p>Objects: {{ stats.split_distribution?.objects?.valid || 0 }}</p>
        </div>
        <div>
          <p class="muted small">Test</p>
          <p>Images: {{ stats.split_distribution?.images?.test || 0 }}</p>
          <p>Objects: {{ stats.split_distribution?.objects?.test || 0 }}</p>
        </div>
      </div>
    </div>

    <div class="ag-card stats" v-if="stats">
      <h3>Image Size Statistics</h3>
      <p class="muted small">Width: min {{ stats.image_size?.width?.min || 0 }}, avg {{ fmt(stats.image_size?.width?.avg) }}, max {{ stats.image_size?.width?.max || 0 }}</p>
      <p class="muted small">Height: min {{ stats.image_size?.height?.min || 0 }}, avg {{ fmt(stats.image_size?.height?.avg) }}, max {{ stats.image_size?.height?.max || 0 }}</p>
    </div>

    <div class="ag-card stats" v-if="stats">
      <h3>Class Summary</h3>
      <p v-if="saveErr" class="err">{{ saveErr }}</p>
      <div class="table-wrap">
        <table class="tbl">
          <thead>
            <tr>
              <th>Class</th>
              <th>Objects</th>
              <th>Train</th>
              <th>Valid</th>
              <th>Test</th>
              <th>Avg bbox ratio (W/H)</th>
              <th>Avg bbox area %</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in stats.classes || []" :key="c.class_id">
              <td>
                <span
                  v-if="editingClassId !== c.class_id"
                  class="class-name"
                  @dblclick="startEdit(c.class_id, c.name || '')"
                >
                  {{ c.name || 'class' }}
                </span>
                <div v-else class="edit-row">
                  <input v-model.trim="editingName" class="ag-input cls-input" @keydown.enter.prevent="saveClassName(c.class_id)" />
                  <button type="button" class="ag-btn ag-btn-primary sm" :disabled="saving" @click="saveClassName(c.class_id)">Save</button>
                </div>
              </td>
              <td>{{ c.count }}</td>
              <td>{{ c.by_split?.train || 0 }}</td>
              <td>{{ c.by_split?.valid || 0 }}</td>
              <td>{{ c.by_split?.test || 0 }}</td>
              <td>{{ fmt(c.avg_bbox_aspect_ratio) }}</td>
              <td>{{ fmt(c.avg_bbox_area_pct) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'

const api = useApi()
const route = useRoute()
const stats = ref(null)
const loading = ref(false)
const latestVersionId = ref('')
const editingClassId = ref(null)
const editingName = ref('')
const saving = ref(false)
const saveErr = ref('')
function fmt(v) { return Number.isFinite(Number(v)) ? Number(v).toFixed(3).replace(/\.?0+$/, '') : '0' }

async function loadStats() {
  loading.value = true
  try {
    stats.value = await api.request(`/projects/${route.params.pid}/class-stats`)
    const vv = await api.request(`/projects/${route.params.pid}/versions`)
    latestVersionId.value = String(vv?.versions?.[0]?.id || '')
  } finally {
    loading.value = false
  }
}

function startEdit(classId, currentName) {
  editingClassId.value = classId
  editingName.value = String(currentName || '')
  saveErr.value = ''
}

async function saveClassName(classId) {
  if (!latestVersionId.value) {
    saveErr.value = 'No version available for class rename'
    return
  }
  const list = [...(stats.value?.classes || [])].sort((a, b) => Number(a.class_id) - Number(b.class_id))
  if (!list.length) return
  const maxId = Number(list[list.length - 1].class_id || 0)
  const names = Array.from({ length: maxId + 1 }, (_, i) => {
    const row = list.find((x) => Number(x.class_id) === i)
    if (!row) return `class_${i}`
    if (i === Number(classId)) return editingName.value.trim() || row.name || `class_${i}`
    return String(row.name || `class_${i}`).trim()
  })
  saving.value = true
  saveErr.value = ''
  try {
    await api.request(`/versions/${latestVersionId.value}/names`, {
      method: 'PATCH',
      body: JSON.stringify({ names }),
    })
    editingClassId.value = null
    editingName.value = ''
    await loadStats()
  } catch (e) {
    saveErr.value = e.message || 'Failed to save class name'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadStats()
})
</script>

<style scoped>
.page { padding: 20px 22px; }
.muted { color: var(--ag-muted); }
.small { font-size: 12px; }
.stats { margin-top: 10px; padding: 12px; }
.grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-top: 10px; }
.tile { padding: 12px; }
.n { margin: 6px 0 0; font-size: 24px; font-weight: 700; }
.loading-wrap {
  min-height: calc(100vh - 220px);
  display: flex;
  align-items: center;
  justify-content: center;
}
.loading-text {
  font-size: 18px;
}
.split-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.table-wrap { overflow: auto; }
.tbl { width: 100%; border-collapse: collapse; font-size: 13px; }
.tbl th, .tbl td { text-align: left; padding: 8px; border-bottom: 1px solid var(--ag-border); white-space: nowrap; }
.class-name { cursor: text; }
.edit-row { display: flex; align-items: center; gap: 8px; }
.cls-input { min-width: 180px; height: 30px; font-size: 12px; }
.err { color: #ff7b7b; margin: 0 0 8px; }
@media (max-width: 960px) {
  .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .split-grid { grid-template-columns: 1fr; }
}
</style>
