<template>
  <div class="showcase" aria-hidden="true">
    <p class="showcase-hint">
      Illustrative samples (CC-style imagery; not your workspace data).
    </p>
    <div class="tiles">
      <figure v-for="item in picks" :key="`${item.id}-${seed}`" class="tile ag-card">
        <div class="frame">
          <img
            class="img"
            :src="item.src"
            :alt="item.alt"
            width="640"
            height="400"
            loading="lazy"
            decoding="async"
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
import { onMounted, ref } from 'vue'

const seed = ref(0)
const picks = ref([])

const pool = [
  {
    id: 'city',
    src: 'https://images.unsplash.com/photo-1449824913935-59a10b8d2000?w=720&q=80&auto=format&fit=crop',
    alt: 'Urban street scene',
    caption: 'Detection & classification in dense scenes',
    boxes: [
      { top: 38, left: 22, w: 18, h: 28, label: 'vehicle', color: '#4d6bfe' },
      { top: 52, left: 48, w: 12, h: 22, label: 'person', color: '#5eead4' },
      { top: 44, left: 72, w: 14, h: 20, label: 'sign', color: '#fbbf24' },
    ],
  },
  {
    id: 'car',
    src: 'https://images.unsplash.com/photo-1503376780353-7e6692767b70?w=720&q=80&auto=format&fit=crop',
    alt: 'Sports car exterior',
    caption: 'Fine-grained object localization',
    boxes: [
      { top: 32, left: 18, w: 62, h: 48, label: 'car', color: '#4d6bfe' },
      { top: 48, left: 5, w: 22, h: 18, label: 'wheel', color: '#a78bfa' },
    ],
  },
  {
    id: 'workshop',
    src: 'https://images.unsplash.com/photo-1504917595217-df4695f5a21d?w=720&q=80&auto=format&fit=crop',
    alt: 'Industrial workspace',
    caption: 'Industrial QA and safety zones',
    boxes: [
      { top: 28, left: 30, w: 28, h: 42, label: 'worker', color: '#5eead4' },
      { top: 62, left: 55, w: 35, h: 22, label: 'equipment', color: '#fb7185' },
    ],
  },
  {
    id: 'logistics',
    src: 'https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?w=720&q=80&auto=format&fit=crop',
    alt: 'Warehouse logistics',
    caption: 'Logistics & inventory markers',
    boxes: [
      { top: 40, left: 20, w: 45, h: 38, label: 'pallet', color: '#4d6bfe' },
      { top: 22, left: 66, w: 20, h: 25, label: 'forklift', color: '#fbbf24' },
    ],
  },
  {
    id: 'nature',
    src: 'https://images.unsplash.com/photo-1472214103451-9374bd1c798e?w=720&q=80&auto=format&fit=crop',
    alt: 'Landscape with hills',
    caption: 'Semantic regions & masks',
    boxes: [
      { top: 48, left: 25, w: 50, h: 30, label: 'field', color: '#4ade80' },
      { top: 18, left: 35, w: 40, h: 28, label: 'sky', color: '#38bdf8' },
    ],
  },
  {
    id: 'tech',
    src: 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=720&q=80&auto=format&fit=crop',
    alt: 'Circuit board macro',
    caption: 'Defect boxes on high-resolution boards',
    boxes: [
      { top: 33, left: 40, w: 22, h: 18, label: 'chip', color: '#fbbf24' },
      { top: 58, left: 18, w: 30, h: 20, label: 'trace', color: '#94a3b8' },
      { top: 22, left: 12, w: 16, h: 14, label: 'solder', color: '#fb7185' },
    ],
  },
]

function boxStyle(b) {
  return {
    top: `${b.top}%`,
    left: `${b.left}%`,
    width: `${b.w}%`,
    height: `${b.h}%`,
    borderColor: b.color,
  }
}

function shuffle(arr) {
  const a = [...arr]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

onMounted(() => {
  seed.value = Date.now()
  picks.value = shuffle(pool).slice(0, 3)
})
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
