<template>
  <div class="showcase" aria-hidden="true">
    <p class="showcase-hint">{{ hintText }}</p>
    <div class="tiles">
      <figure v-for="item in displayItems" :key="item.id" class="tile ag-card">
        <div class="frame">
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
          <div v-for="(b, bi) in item.boxes" :key="bi" class="bbox" :style="boxStyle(b)">
            <span class="label" :style="{ borderColor: b.color, color: b.color }">{{ b.label }}</span>
          </div>
        </div>
        <figcaption class="cap">{{ item.caption }}</figcaption>
      </figure>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'

const PALETTE = ['#4d6bfe', '#5eead4', '#fbbf24', '#a78bfa', '#fb7185', '#4ade80', '#38bdf8', '#94a3b8']

function colorForClass(classId) {
  const i = typeof classId === 'number' && Number.isFinite(classId) ? Math.abs(classId) : 0
  return PALETTE[i % PALETTE.length]
}

const samples = ref([])
const loadState = ref('loading') // 'loading' | 'ok' | 'empty' | 'error'
const brokenIds = ref(new Set())

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

const displayItems = computed(() => {
  return samples.value
    .filter((s) => !brokenIds.value.has(s.image_id))
    .map((s) => {
      const route = s.file_route || `/api/images/${s.image_id}/file`
      const boxes = (s.boxes || []).map((b) => ({
        top: b.top_pct,
        left: b.left_pct,
        w: b.width_pct,
        h: b.height_pct,
        label: String(b.name || `class_${b.class_id}`),
        color: colorForClass(b.class_id),
      }))
      return {
        id: s.image_id,
        rawId: s.image_id,
        src: route,
        alt: s.project ? `Sample from ${s.project}` : 'Dataset sample',
        caption: s.caption || s.project || 'Public dataset',
        boxes,
      }
    })
})

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
