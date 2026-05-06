<template>
  <div class="page">
    <div class="shell" v-if="meta">
      <button type="button" class="back-btn back-btn-fixed" @click="goBackToDataset">←</button>
      <div v-if="canEdit" class="tool-rail">
        <button
          type="button"
          class="tool-btn"
          :class="{ active: interactionMode === 'hand' }"
          title="Hand mode"
          @click="interactionMode = 'hand'"
        >
          ✋
        </button>
        <button
          type="button"
          class="tool-btn"
          :class="{ active: interactionMode === 'box' }"
          title="Box mode"
          @click="interactionMode = 'box'"
        >
          ▭
        </button>
      </div>
      <div class="topbar">
        <div class="pager-center" v-if="hasPageNav">
          <button type="button" class="step-btn" :disabled="currentIdx <= 0" @click="goPrevImage">←</button>
          <span class="pager-label">{{ currentIdx + 1 }} / {{ navIds.length }}</span>
          <button type="button" class="step-btn" :disabled="currentIdx >= navIds.length - 1" @click="goNextImage">→</button>
        </div>
        <div v-else class="pager-center muted small">1 / 1</div>
        <div class="topbar-right">
          <button v-if="canEdit" type="button" class="delete-image-btn" :disabled="saving" @click="deleteCurrentImage">
            Delete
          </button>
          <label class="split-badge">
          <AgSelect v-if="canEdit" v-model="selectedSplit" :options="splitOptions" size="small" class="split-select" @change="changeSplit" />
          <span v-else class="split-static">{{ selectedSplit }}</span>
          </label>
        </div>
      </div>

      <div class="grid">
        <aside class="left-pane ag-card">
          <div class="left-head">Annotations <span class="ag-pill sm">{{ bboxCount }}</span></div>
          <div class="left-tabs">
            <button
              type="button"
              class="left-tab-btn"
              :class="{ active: leftTab === 'classes' }"
              @click="leftTab = 'classes'"
            >
              Classes
            </button>
            <button
              type="button"
              class="left-tab-btn"
              :class="{ active: leftTab === 'layers' }"
              @click="leftTab = 'layers'"
            >
              Layers
            </button>
          </div>
          <div v-if="leftTab === 'classes'" class="left-content">
            <div class="class-list">
              <button
                v-for="row in usedClassStats"
                :key="`used-${row.classIdx}`"
                type="button"
                class="class-row"
                @mouseenter="hoveredClassIdx = row.classIdx"
                @mouseleave="hoveredClassIdx = null"
              >
                <span class="class-dot" :style="{ background: row.color }" />
                <span class="class-name">{{ row.name }}</span>
                <span class="class-count">{{ row.count }}</span>
              </button>
            </div>
            <div class="unused-title">Unused Classes</div>
            <div class="unused-list">
              <div v-for="row in unusedClassStats" :key="`unused-${row.classIdx}`" class="unused-row">
                <span class="class-dot" :style="{ background: row.color }" />
                <span class="class-name">{{ row.name }}</span>
              </div>
            </div>
          </div>
          <div v-else class="left-content">
            <div class="layer-list">
              <button
                v-for="row in layerRows"
                :key="`layer-${row.segIdx}`"
                type="button"
                class="layer-row"
                :class="{ active: selectedSegIdx === row.segIdx }"
                @click="selectLayer(row.segIdx)"
              >
                <span class="layer-dot" :style="{ background: row.color }" />
                <span class="layer-name">{{ row.name }}</span>
                <span class="layer-count">1</span>
              </button>
            </div>
          </div>
        </aside>

        <div class="pane ag-card image-pane">
          <AnnotateViewport
            v-model="segments"
            :image-id="meta.id"
            :default-class-index="defaultClassIdx"
            :interaction-mode="interactionMode"
            :hover-class-index="hoveredClassIdx"
            :selected-index="selectedSegIdx"
            :read-only="!canEdit"
            @select="onViewportSelect"
          />
          <div v-if="canEdit && selectedSegIdx !== null && selectedBBox" class="class-popup ag-card">
            <p class="pop-title">Class</p>
            <div v-if="renameClassActive" class="rename-row">
              <input
                v-model.trim="renameClassDraft"
                class="ag-input"
                type="text"
                placeholder="New class name"
                @keydown.enter.prevent="createClassFromRename"
                @blur="renameClassActive = false"
              />
            </div>
            <AgSelect
              v-if="!renameClassActive"
              v-model="selectedClassValue"
              :options="classSelectOptions"
              size="small"
              class="class-select"
              title="Double click to rename current class"
              @dblclick="beginRenameClass"
              @change="onClassSelect"
            />
          </div>
          <div class="info">
            <span class="muted small">{{ meta.stem }}</span>
            <span class="muted small" v-if="meta.width">{{ meta.width }}×{{ meta.height }}</span>
          </div>
        </div>

        <div v-if="bboxCount > 0" class="pane ag-card text-pane">
          <div class="source-wrap" :style="editorStyle">
            <div
              v-if="selectedLineIdx >= 0"
              class="line-highlight"
              :style="{ top: `${highlightTop}px`, height: `${lineHeightPx}px` }"
            />
            <textarea
              ref="sourceRef"
              v-model="labelText"
              class="ag-input mono source-area"
              :rows="editorRows"
              :cols="editorCols"
              wrap="off"
              spellcheck="false"
              :readonly="!canEdit"
              @blur="applyTextFromEditor"
              @scroll="onEditorScroll"
            />
          </div>
          <p v-if="err" class="err">{{ err }}</p>
        </div>
      </div>
    </div>
    <div v-else class="shell muted">Loading ...</div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AnnotateViewport from '@/components/AnnotateViewport.vue'
import AgSelect from '@/components/AgSelect.vue'
import { useApi } from '@/composables/useApi'
import {
  hslStrokeForClass,
  parseDatasetClassNames,
  parseLabelSegments,
  serializeLabelSegments,
} from '@/utils/yoloLabel'

const api = useApi()
const route = useRoute()
const router = useRouter()

const meta = ref(null)
const yamlText = ref('')
const labelText = ref('')
const segments = ref([])
const saving = ref(false)
const ok = ref(false)
const err = ref('')
const lastSavedLabelText = ref('')

const selectedSegIdx = ref(null)
const newClassName = ref('')
const addingClass = ref(false)
const yamlErr = ref('')
const interactionMode = ref('hand')
const sourceRef = ref(null)
const editorScrollTop = ref(0)
const selectedSplit = ref('train')
const lastChosenClass = ref(0)
const renameClassActive = ref(false)
const renameClassDraft = ref('')
const leftTab = ref('classes')
const hoveredClassIdx = ref(null)
const splitOptions = [
  { value: 'train', label: 'train' },
  { value: 'valid', label: 'valid' },
  { value: 'test', label: 'test' },
]
const navIds = computed(() => {
  const raw = String(route.query.ids || '').trim()
  if (!raw) return []
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
})
const currentIdx = computed(() => {
  const n = Number.parseInt(String(route.query.ix || '0'), 10)
  return Number.isNaN(n) ? 0 : Math.max(0, Math.min(n, Math.max(0, navIds.value.length - 1)))
})
const hasPageNav = computed(() => navIds.value.length > 1)
const isDirty = computed(() => labelText.value !== lastSavedLabelText.value)
const isVersionSource = computed(() => String(route.query.source || '').trim() === 'version')
const canEdit = computed(() => Boolean(meta.value?.can_edit) && !isVersionSource.value)
const labelLines = computed(() => labelText.value.split('\n'))

watch(
  segments,
  (segs) => {
    labelText.value = serializeLabelSegments(segs)
  },
  { deep: true },
)

const chipsQuick = computed(() => parseDatasetClassNames(yamlText.value))

const effectiveClassNames = computed(() => {
  const names = [...chipsQuick.value]
  let maxIx = -1
  for (const s of segments.value) {
    if (s.type === 'bbox' && typeof s.cls === 'number') maxIx = Math.max(maxIx, s.cls)
  }
  if (!names.length) {
    const n = Math.max(maxIx + 1, 1)
    return Array.from({ length: n }, (_, i) => `class_${i}`)
  }
  while (names.length <= maxIx) names.push(`class_${names.length}`)
  return names
})

const selectedBBox = computed(() => {
  const i = selectedSegIdx.value
  if (i === null) return null
  const s = segments.value[i]
  return s?.type === 'bbox' ? s : null
})
const selectedLineIdx = computed(() => (selectedSegIdx.value == null ? -1 : selectedSegIdx.value))
const bboxCount = computed(() => segments.value.filter((s) => s.type === 'bbox').length)
const classSelectOptions = computed(() =>
  effectiveClassNames.value.map((name, i) => ({ value: i, label: name || `class_${i}` })),
)
const usedClassStats = computed(() => {
  const counts = new Map()
  for (const seg of segments.value) {
    if (seg?.type !== 'bbox') continue
    const cls = Number(seg.cls)
    if (!Number.isFinite(cls)) continue
    counts.set(cls, (counts.get(cls) || 0) + 1)
  }
  return [...counts.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([classIdx, count]) => ({
      classIdx,
      count,
      name: effectiveClassNames.value[classIdx] || `class_${classIdx}`,
      color: hslStrokeForClass(classIdx),
    }))
})
const unusedClassStats = computed(() => {
  const used = new Set(usedClassStats.value.map((x) => x.classIdx))
  const out = []
  for (let i = 0; i < effectiveClassNames.value.length; i++) {
    if (used.has(i)) continue
    out.push({
      classIdx: i,
      name: effectiveClassNames.value[i] || `class_${i}`,
      color: hslStrokeForClass(i),
    })
  }
  return out
})
const layerRows = computed(() =>
  segments.value
    .map((seg, segIdx) => ({ seg, segIdx }))
    .filter((x) => x.seg?.type === 'bbox')
    .map(({ seg, segIdx }) => ({
      segIdx,
      cls: seg.cls,
      name: effectiveClassNames.value[seg.cls] || `class_${seg.cls}`,
      color: hslStrokeForClass(seg.cls),
    })),
)
const selectedClassValue = computed({
  get: () => {
    if (selectedSegIdx.value == null) return lastChosenClass.value
    const seg = segments.value[selectedSegIdx.value]
    return seg?.type === 'bbox' ? seg.cls : lastChosenClass.value
  },
  set: () => {},
})
const currentSelectedClassName = computed(() => {
  const idx = Number(selectedClassValue.value || 0)
  return effectiveClassNames.value[idx] || `class_${idx}`
})
const editorRows = computed(() => Math.max(1, labelLines.value.length))
const editorCols = computed(() => 44)
const lineHeightPx = ref(22)
const editorPadTopPx = ref(10)
const editorStyle = computed(() => ({
  width: `${editorCols.value}ch`,
  height: `${editorRows.value * lineHeightPx.value + editorPadTopPx.value * 2 + 4}px`,
}))
const highlightTop = computed(
  () => editorPadTopPx.value + selectedLineIdx.value * lineHeightPx.value - editorScrollTop.value - 1,
)

const defaultClassIdx = computed(() => {
  if (lastChosenClass.value >= 0) return lastChosenClass.value
  if (selectedBBox.value) return selectedBBox.value.cls
  return 0
})

function onViewportSelect(segIndex) {
  if (canEdit.value && segIndex === null && selectedSegIdx.value !== null && isDirty.value && !saving.value) {
    save()
  }
  selectedSegIdx.value = segIndex
  renameClassActive.value = false
  if (segIndex !== null && segments.value[segIndex]?.type === 'bbox') {
    lastChosenClass.value = segments.value[segIndex].cls
    nextTick(() => {
      focusLineInEditor(segIndex)
    })
  }
}

function selectLayer(segIndex) {
  if (segIndex == null || !segments.value[segIndex] || segments.value[segIndex].type !== 'bbox') return
  selectedSegIdx.value = segIndex
  renameClassActive.value = false
  lastChosenClass.value = segments.value[segIndex].cls
  nextTick(() => {
    focusLineInEditor(segIndex)
  })
}

function onGlobalPointerDown(e) {
  if (selectedSegIdx.value === null) return
  const target = e.target
  if (!(target instanceof Element)) return
  if (target.closest('.ovl')) return
  if (target.closest('.class-popup')) return
  if (isDirty.value && !saving.value) save()
  selectedSegIdx.value = null
  renameClassActive.value = false
}

async function onClassSelect(val) {
  if (!canEdit.value) return
  const cls = parseInt(String(val), 10)
  if (Number.isNaN(cls) || selectedSegIdx.value === null) return
  lastChosenClass.value = cls
  const i = selectedSegIdx.value
  const next = segments.value.map((s, j) => {
    if (j !== i || s.type !== 'bbox') return s
    return { ...s, cls }
  })
  segments.value = next
  await persist()
}

function beginRenameClass() {
  renameClassDraft.value = currentSelectedClassName.value
  renameClassActive.value = true
  nextTick(() => {
    const el = document.querySelector('.rename-row .ag-input')
    el?.focus?.()
  })
}

function removeSelected() {
  const i = selectedSegIdx.value
  if (i === null) return
  const next = segments.value.filter((_, j) => j !== i)
  segments.value = next
  selectedSegIdx.value = null
}

function addCenterBox(clsIdx) {
  const cls = Math.max(0, Math.floor(clsIdx))
  segments.value = [
    ...segments.value,
    {
      type: 'bbox',
      cls,
      cx: 0.5,
      cy: 0.5,
      w: Math.min(0.2, 0.98),
      h: Math.min(0.2, 0.98),
    },
  ]
}

function applyTextFromEditor() {
  segments.value = parseLabelSegments(labelText.value)
}

function namesBaseForYaml() {
  const fromYaml = chipsQuick.value
  if (fromYaml.length > 0) return [...fromYaml]
  let maxIx = -1
  for (const s of segments.value) {
    if (s.type === 'bbox' && typeof s.cls === 'number') maxIx = Math.max(maxIx, s.cls)
  }
  const n = Math.max(maxIx + 1, 1)
  return Array.from({ length: n }, (_, i) => `class_${i}`)
}

async function addDatasetClass() {
  const raw = newClassName.value.trim()
  const n = raw.replace(/\s+/g, ' ')
  if (!n || !meta.value) return
  yamlErr.value = ''
  addingClass.value = true
  try {
    const base = namesBaseForYaml()
    const nextNames = [...base, n]
    const js = await api.request(`/versions/${meta.value.version_id}/names`, {
      method: 'PATCH',
      body: JSON.stringify({ names: nextNames }),
    })
    yamlText.value = js.data_yaml || ''
    newClassName.value = ''
    const newIx = nextNames.length - 1
    if (selectedSegIdx.value !== null && segments.value[selectedSegIdx.value]?.type === 'bbox') {
      onClassSelect(String(newIx))
    }
  } catch (e) {
    yamlErr.value = e.message || 'failed to update data.yaml'
  } finally {
    addingClass.value = false
  }
}

async function createClassByName(rawName) {
  const n = String(rawName || '').trim().replace(/\s+/g, ' ')
  if (!n || !meta.value) return
  const base = namesBaseForYaml()
  const already = base.findIndex((x) => String(x).trim().toLowerCase() === n.toLowerCase())
  if (already >= 0) {
    await onClassSelect(String(already))
    return
  }
  yamlErr.value = ''
  addingClass.value = true
  try {
    const nextNames = [...base, n]
    const js = await api.request(`/versions/${meta.value.version_id}/names`, {
      method: 'PATCH',
      body: JSON.stringify({ names: nextNames }),
    })
    yamlText.value = js.data_yaml || ''
    const newIx = nextNames.length - 1
    await onClassSelect(String(newIx))
  } catch (e) {
    yamlErr.value = e.message || 'failed to update data.yaml'
  } finally {
    addingClass.value = false
  }
}

async function createClassFromRename() {
  if (!canEdit.value) {
    renameClassActive.value = false
    return
  }
  const draft = renameClassDraft.value.trim()
  if (!draft) {
    renameClassActive.value = false
    return
  }
  await createClassByName(draft)
  renameClassActive.value = false
}

async function load() {
  const id = route.params.imgid
  const js = await api.request(`/images/${id}/json`)
  meta.value = js
  segments.value = parseLabelSegments(js.label_text || '')
  labelText.value = serializeLabelSegments(segments.value)
  lastSavedLabelText.value = labelText.value
  ok.value = false
  err.value = ''
  selectedSegIdx.value = null
  selectedSplit.value = String(js.split || 'train')
  try {
    const v = await api.request(`/versions/${js.version_id}`)
    yamlText.value = v.data_yaml || ''
  } catch {
    yamlText.value = ''
  }
}

async function persist(splitOverride = '') {
  if (!canEdit.value) return
  if (saving.value || !meta.value) return
  saving.value = true
  ok.value = false
  err.value = ''
  try {
    labelText.value = serializeLabelSegments(segments.value)
    await api.request(`/images/${meta.value.id}/label`, {
      method: 'PUT',
      body: JSON.stringify({ label_text: labelText.value, split: splitOverride || selectedSplit.value }),
    })
    lastSavedLabelText.value = labelText.value
    ok.value = true
  } catch (e) {
    err.value = e.message
  } finally {
    saving.value = false
  }
}

async function save() {
  await persist()
}

async function changeSplit() {
  if (!canEdit.value) return
  await persist(selectedSplit.value)
}

function lineRangeByIndex(text, lineIdx) {
  if (lineIdx < 0) return null
  let start = 0
  let cur = 0
  while (cur < lineIdx) {
    const nl = text.indexOf('\n', start)
    if (nl < 0) return null
    start = nl + 1
    cur++
  }
  const endNl = text.indexOf('\n', start)
  const end = endNl < 0 ? text.length : endNl
  return { start, end }
}

function focusLineInEditor(lineIdx) {
  const ta = sourceRef.value
  if (!ta) return
  const r = lineRangeByIndex(labelText.value, lineIdx)
  if (!r) return
  ta.focus()
  ta.setSelectionRange(r.start, r.end)
  ta.scrollTop = Math.max(0, lineIdx * lineHeightPx.value)
  editorScrollTop.value = ta.scrollTop
}

function onEditorScroll(e) {
  editorScrollTop.value = e.target?.scrollTop || 0
}

function syncEditorMetrics() {
  const ta = sourceRef.value
  if (!ta) return
  const st = window.getComputedStyle(ta)
  const lh = parseFloat(st.lineHeight)
  const pt = parseFloat(st.paddingTop)
  if (Number.isFinite(lh) && lh > 0) lineHeightPx.value = lh
  if (Number.isFinite(pt) && pt >= 0) editorPadTopPx.value = pt
}

async function goBackToDataset() {
  const from = String(route.query.from || '').trim()
  if (from) {
    await router.push(from)
    return
  }
  const vid = String(route.query.vid || '').trim()
  if (vid) {
    await router.push(`/versions/${vid}`)
    return
  }
  const wid = String(route.query.wid || '')
  const pid = String(route.query.pid || '')
  if (wid && pid) {
    await router.push(`/app/${wid}/projects/${pid}/dataset`)
    return
  }
  if (meta.value?.version_id) {
    await router.push(`/versions/${meta.value.version_id}`)
    return
  }
  await router.push('/workspaces')
}

async function goToImageAt(idx) {
  if (idx < 0 || idx >= navIds.value.length) return
  const target = navIds.value[idx]
  if (!target) return
  await router.push({
    path: `/annotate/${target}`,
    query: {
      ...route.query,
      ix: String(idx),
    },
  })
}

async function deleteCurrentImage() {
  if (!canEdit.value || !meta.value?.id) return
  const pid = String(route.query.pid || '').trim()
  if (!pid) return
  const ids = [...navIds.value]
  const cur = currentIdx.value
  const remaining = ids.filter((id) => id !== meta.value.id)
  await api.request(`/projects/${pid}/images/delete`, {
    method: 'POST',
    body: JSON.stringify({ ids: [meta.value.id] }),
  })
  if (remaining.length === 0) {
    await goBackToDataset()
    return
  }
  const nextIdx = Math.min(cur, remaining.length - 1)
  await router.push({
    path: `/annotate/${remaining[nextIdx]}`,
    query: {
      ...route.query,
      ids: remaining.join(','),
      ix: String(nextIdx),
    },
  })
}

function goPrevImage() {
  goToImageAt(currentIdx.value - 1)
}

function goNextImage() {
  goToImageAt(currentIdx.value + 1)
}

onMounted(() => load().catch(() => {}))
onMounted(() => nextTick(() => syncEditorMetrics()))
onMounted(() => window.addEventListener('pointerdown', onGlobalPointerDown, true))
onBeforeUnmount(() => window.removeEventListener('pointerdown', onGlobalPointerDown, true))
watch(
  () => route.params.imgid,
  () => {
    load().catch(() => {})
  },
)
watch(selectedLineIdx, (idx) => {
  if (idx >= 0) nextTick(() => focusLineInEditor(idx))
})
watch(labelText, () => nextTick(() => syncEditorMetrics()))
</script>

<style scoped>
.page {
  min-height: 100vh;
}

.shell {
  max-width: 1680px;
  margin: 0 auto;
  padding: 14px 16px;
}

.topbar {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  margin-bottom: 48px;
}
.topbar-right {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.delete-image-btn {
  height: 30px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid rgba(255, 123, 123, 0.45);
  background: rgba(255, 123, 123, 0.12);
  color: #ffb0b0;
  cursor: pointer;
}

.back-btn {
  width: 36px;
  height: 36px;
  border-radius: 999px;
  border: 1px solid var(--ag-border);
  background: rgba(255, 255, 255, 0.04);
  color: var(--ag-text);
  cursor: pointer;
  font-size: 18px;
}
.back-btn-fixed {
  position: fixed;
  left: 12px;
  top: 12px;
  z-index: 45;
}

.pager-center {
  justify-self: center;
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.split-badge {
  justify-self: end;
  display: inline-flex;
  align-items: center;
  margin-right: 34px;
}
.split-select {
  width: 110px;
}
.split-static {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 78px;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--ag-border);
  border-radius: 8px;
  font-size: 12px;
  color: var(--ag-muted);
  text-transform: lowercase;
}

.step-btn {
  width: 30px;
  height: 30px;
  border-radius: 999px;
  border: 1px solid var(--ag-border);
  background: transparent;
  color: var(--ag-text);
  cursor: pointer;
}

.step-btn:disabled {
  opacity: 0.45;
  cursor: default;
}

.pager-label {
  font-size: 13px;
  color: var(--ag-muted);
}

.grid {
  display: grid;
  grid-template-columns: 206px 182px minmax(0, 1fr) auto;
  gap: 4px;
  align-items: start;
}

@media (max-width: 920px) {
  .grid {
    grid-template-columns: 1fr;
  }
}

.info {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  text-align: center;
  align-items: center;
  margin-top: 12px;
}

.left-pane {
  min-height: min(560px, 70vh);
  display: flex;
  flex-direction: column;
  padding: 6px;
  transform: translateX(-42px);
}
.image-pane {
  grid-column: 3;
}
.left-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 8px;
}
.left-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2px;
  border-bottom: 1px solid var(--ag-border);
  margin-bottom: 10px;
}
.left-tab-btn {
  border: 0;
  background: transparent;
  color: var(--ag-muted);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  padding: 6px 6px 8px;
  border-bottom: 2px solid transparent;
}
.left-tab-btn.active {
  color: var(--ag-accent);
  border-bottom-color: var(--ag-accent);
}
.left-content {
  min-height: 0;
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
}
.class-list {
  display: grid;
  gap: 6px;
}
.class-row {
  border: 0;
  background: transparent;
  color: var(--ag-text);
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 6px;
  padding: 2px 3px;
  border-radius: 6px;
  text-align: left;
  cursor: default;
}
.class-row:hover {
  background: rgba(255, 255, 255, 0.04);
}
.class-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex: 0 0 auto;
}
.class-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  font-size: 13px;
}
.class-count {
  color: var(--ag-muted);
  font-size: 11px;
  border-radius: 999px;
  padding: 1px 6px;
  background: rgba(255, 255, 255, 0.05);
}
.unused-title {
  margin-top: auto;
  padding-top: 10px;
  color: var(--ag-muted);
  font-weight: 600;
  font-size: 12px;
}
.unused-list {
  margin-top: 8px;
  max-height: 180px;
  overflow: auto;
  padding-right: 2px;
}
.unused-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 6px;
  padding: 2px 3px;
  color: var(--ag-muted);
  font-size: 13px;
}
.layer-list {
  display: grid;
  gap: 4px;
}
.layer-row {
  border: 0;
  background: transparent;
  color: var(--ag-text);
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 6px;
  padding: 2px 3px;
  border-radius: 6px;
  text-align: left;
  cursor: pointer;
}
.layer-row:hover {
  background: rgba(255, 255, 255, 0.04);
}
.layer-row.active {
  background: rgba(77, 107, 254, 0.18);
}
.layer-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
}
.layer-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  font-size: 13px;
}
.layer-count {
  color: var(--ag-muted);
  font-size: 11px;
}

.tool-rail {
  position: fixed;
  right: 14px;
  top: 46%;
  transform: translateY(-50%);
  z-index: 40;
  display: grid;
  gap: 6px;
}

.tool-btn {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  border: 1px solid var(--ag-border);
  background: rgba(255, 255, 255, 0.04);
  color: var(--ag-text);
  cursor: pointer;
}

.tool-btn.active {
  border-color: var(--ag-accent);
  background: rgba(77, 107, 254, 0.2);
}

.class-popup {
  position: absolute;
  left: -184px;
  top: 8px;
  z-index: 6;
  width: 178px;
  padding: 8px;
}

.text-pane { justify-self: end; width: auto; max-width: none; margin-left: 24px; }
.source-wrap {
  position: relative;
  max-width: calc(100vw - 120px);
  max-height: calc(100vh - 140px);
}
.line-highlight {
  position: absolute;
  left: 0;
  right: 0;
  background: rgba(77, 107, 254, 0.26);
  border-left: 2px solid rgba(77, 107, 254, 0.85);
  pointer-events: none;
  border-radius: 4px;
  z-index: 1;
}
.source-area {
  margin-top: 0;
  min-height: 0;
  resize: none;
  overflow: auto;
  position: relative;
  z-index: 2;
  line-height: 22px;
  padding: 10px 14px;
}

.pop-title {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--ag-muted);
}

.rename-row {
  display: block;
  margin-bottom: 8px;
}
.rename-row .ag-input { width: 100%; min-width: 0; }
.class-select { width: 100%; }
.class-select :deep(.ag-selectx-menu) {
  max-height: 340px; /* ~10 rows */
  overflow-y: auto;
}

.sm.ag-pill {
  font-size: 11px;
}

.small {
  font-size: 13px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  line-height: 1.45;
}

.err {
  color: #ff7b7b;
  font-size: 14px;
}

pre,
code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}

.muted {
  color: var(--ag-muted);
}
</style>
