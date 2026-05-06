<template>
  <div class="showcase" aria-hidden="true">
    <p class="showcase-hint">{{ hintText }}</p>
    <div class="tiles">
      <figure v-for="item in baseItems" :key="item.id" class="tile ag-card">
        <div class="frame" :ref="(el) => observeFrame(el, item.id)">
          <img
            class="img"
            :src="item.src"
            :alt="item.alt"
            width="640"
            height="400"
            loading="lazy"
            decoding="async"
            @error="onImgError(item)"
          />
          <div v-for="(b, bi) in mappedBoxes(item)" :key="bi" class="bbox" :style="boxStyle(b)">
            <span class="label" :style="{ borderColor: b.color, color: b.color }">{{ b.label }}</span>
          </div>
        </div>
        <figcaption class="cap">{{ item.caption }}</figcaption>
      </figure>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const PALETTE = ['#4d6bfe', '#5eead4', '#fbbf24', '#a78bfa', '#fb7185', '#4ade80', '#38bdf8', '#94a3b8']

function colorForClass(classId) {
  const i = typeof classId === 'number' && Number.isFinite(classId) ? Math.abs(classId) : 0
  return PALETTE[i % PALETTE.length]
}

/** YOLO-style percents are relative to natural image; map to frame coords for object-fit: cover. */
function mapBoxesForCover(boxes, iw, ih, fw, fh) {
  if (!iw || !ih || !fw || !fh) {
    return boxes.map((b) => ({
      ...b,
      top: b.top_pct,
      left: b.left_pct,
      w: b.width_pct,
      h: b.height_pct,
    }))
  }
  const s = Math.max(fw / iw, fh / ih)
  const dispW = iw * s
  const dispH = ih * s
  const ox = (fw - dispW) / 2
  const oy = (fh - dispH) / 2

  return boxes.map((b) => {
    const lf = b.left_pct / 100
    const tf = b.top_pct / 100
    const wf = b.width_pct / 100
    const hf = b.height_pct / 100
    const x0 = ox + lf * iw * s
    const y0 = oy + tf * ih * s
    const w = wf * iw * s
    const h = hf * ih * s
    return {
      ...b,
      top: (y0 / fh) * 100,
      left: (x0 / fw) * 100,
      w: (w / fw) * 100,
      h: (h / fh) * 100,
    }
  })
}

const samples = ref([])
const loadState = ref('loading') // 'loading' | 'ok' | 'empty' | 'error'
const brokenIds = ref(new Set())
const frameSizes = ref({})
const observers = new Map()

const hintText = computed(() => {
  switch (loadState.value) {
    case 'loading':
      return 'Loading samples from public datasets…'
    case 'empty':
      return 'No public annotated samples yet — mark a project public with labeled draft images to fill this preview.'
    case 'error':
      return 'Could not load public dataset preview.'
    default:
      return 'Frames and class names come from real public projects on this server.'
  }
})

function onImgError(item) {
  if (item?.rawId) {
    brokenIds.value = new Set([...brokenIds.value, item.rawId])
  }
}

function observeFrame(el, id) {
  const oid = String(id)
  const old = observers.get(oid)
  if (old) {
    old.disconnect()
    observers.delete(oid)
  }
  if (!el) {
    const next = { ...frameSizes.value }
    delete next[oid]
    frameSizes.value = next
    return
  }
  const ro = new ResizeObserver((entries) => {
    const e = entries[0]
    if (!e) return
    const { width, height } = e.contentRect
    if (width <= 0 || height <= 0) return
    frameSizes.value = { ...frameSizes.value, [oid]: { w: width, h: height } }
  })
  ro.observe(el)
  observers.set(oid, ro)
}

onBeforeUnmount(() => {
  for (const ro of observers.values()) {
    ro.disconnect()
  }
  observers.clear()
})

async function loadSamples() {
  loadState.value = 'loading'
  try {
    const res = await fetch('/api/public/landing-samples', { credentials: 'same-origin' })
    if (!res.ok) {
      loadState.value = 'error'
      samples.value = []
      return
    }
    const data = await res.json()
    const list = Array.isArray(data.samples) ? data.samples : []
    samples.value = list.slice(0, 3)
    loadState.value = list.length === 0 ? 'empty' : 'ok'
  } catch {
    loadState.value = 'error'
    samples.value = []
  }
}

const baseItems = computed(() => {
  return samples.value
    .filter((s) => !brokenIds.value.has(s.image_id))
    .map((s) => {
      const route = s.file_route || `/api/images/${s.image_id}/file`
      const iw = Number(s.width) || 0
      const ih = Number(s.height) || 0
      const boxes = (s.boxes || []).map((b) => ({
        left_pct: b.left_pct,
        top_pct: b.top_pct,
        width_pct: b.width_pct,
        height_pct: b.height_pct,
        label: String(b.name || `class_${b.class_id}`),
        color: colorForClass(b.class_id),
      }))
      return {
        id: s.image_id,
        rawId: s.image_id,
        src: route,
        alt: s.project ? `Sample from ${s.project}` : 'Dataset sample',
        caption: s.caption || s.project || 'Public dataset',
        iw,
        ih,
        boxes,
      }
    })
})

function mappedBoxes(item) {
  const fs = frameSizes.value[String(item.id)]
  if (!fs) {
    return mapBoxesForCover(item.boxes, item.iw, item.ih, 0, 0)
  }
  return mapBoxesForCover(item.boxes, item.iw, item.ih, fs.w, fs.h)
}

function boxStyle(b) {
  return {
    top: `${b.top}%`,
    left: `${b.left}%`,
    width: `${b.w}%`,
    height: `${b.h}%`,
    borderColor: b.color,
  }
}

onMounted(loadSamples)
</script>

<style scoped>
.showcase {
  margin-top: 8px;
}

.showcase-hint {
  margin: 0 0 14px;
  font-size: 12px;
  color: var(--ag-muted);
  letter-spacing: 0.02em;
}

.tiles {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
}

@media (max-width: 1024px) {
  .tiles {
    grid-template-columns: 1fr;
  }
}

.tile {
  margin: 0;
  padding: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.frame {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  background: var(--ag-surface2);
  aspect-ratio: 16 / 10;
}

.img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  vertical-align: middle;
}

.bbox {
  position: absolute;
  box-sizing: border-box;
  border: 2px solid;
  border-radius: 4px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35);
  pointer-events: none;
}

.label {
  position: absolute;
  top: -1.5rem;
  left: -2px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(15, 15, 18, 0.92);
  border: 1px solid;
  white-space: nowrap;
}

.cap {
  margin: 10px 12px 14px;
  font-size: 12px;
  color: var(--ag-muted);
  line-height: 1.45;
}
</style>
