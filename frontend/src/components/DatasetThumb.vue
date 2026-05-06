<template>
  <div ref="wrapRef" class="thumb-wrap">
    <div v-if="!blobUrl" class="ph"><span class="muted">…</span></div>
    <div v-show="blobUrl" class="stack">
      <img ref="imgRef" :src="blobUrl" class="img" draggable="false" alt="" @load="layout" />
      <svg v-if="disp && svgBoxes.length" class="ovl" :viewBox="`0 0 ${vbW} ${vbH}`">
        <rect
          v-for="(r, i) in svgBoxes"
          :key="i"
          :x="r.x"
          :y="r.y"
          :width="r.w"
          :height="r.h"
          :fill="r.stroke"
          fill-opacity="0.35"
          :stroke="r.stroke"
          stroke-width="2"
          vector-effect="non-scaling-stroke"
        />
      </svg>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { fetchImageBlob } from '@/composables/useApi'
import { yoloNormToLTRB, hslStrokeForClass } from '@/utils/yoloLabel'

const props = defineProps({
  imageId: { type: String, required: true },
  boxes: { type: Array, default: () => [] },
})

const wrapRef = ref(null)
const imgRef = ref(null)
const blobUrl = ref('')
const vbW = ref(80)
const vbH = ref(60)
const disp = ref(null)

const svgBoxes = computed(() => {
  const d = disp.value
  if (!d) return []
  const nw = d.nw
  const nh = d.nh
  if (nw <= 0 || nh <= 0) return []
  const out = []
  for (const b of props.boxes) {
    if (!Array.isArray(b) || b.length < 5) continue
    const cls = b[0]
    const L = yoloNormToLTRB({ cx: b[1], cy: b[2], w: b[3], h: b[4] })
    const x = d.offX + L.x * nw * d.scale
    const y = d.offY + L.y * nh * d.scale
    const w = Math.max(0.5, L.w * nw * d.scale)
    const h = Math.max(0.5, L.h * nh * d.scale)
    out.push({
      x,
      y,
      w,
      h,
      stroke: hslStrokeForClass(cls),
    })
  }
  return out
})

function layout() {
  const el = imgRef.value
  if (!el?.naturalWidth) {
    disp.value = null
    return
  }
  const nw = el.naturalWidth
  const nh = el.naturalHeight
  const cw = el.clientWidth
  const ch = el.clientHeight
  vbW.value = cw
  vbH.value = ch
  const scale = Math.min(cw / nw, ch / nh)
  disp.value = {
    nw,
    nh,
    scale,
    offX: (cw - nw * scale) / 2,
    offY: (ch - nh * scale) / 2,
  }
}

let ro

async function load() {
  if (blobUrl.value) URL.revokeObjectURL(blobUrl.value)
  blobUrl.value = ''
  if (!props.imageId) return
  try {
    const b = await fetchImageBlob(`/images/${props.imageId}/file`)
    blobUrl.value = URL.createObjectURL(b)
  } catch {
    blobUrl.value = ''
  }
}

watch(
  () => props.imageId,
  () => load(),
  { immediate: true },
)

onMounted(() => {
  ro = new ResizeObserver(() => layout())
  if (wrapRef.value) ro.observe(wrapRef.value)
})

onBeforeUnmount(() => {
  if (ro && wrapRef.value) ro.unobserve(wrapRef.value)
  ro = null
  if (blobUrl.value) URL.revokeObjectURL(blobUrl.value)
})

watch(blobUrl, () => nextTick(layout))

watch(() => props.boxes, layout, { deep: true })
</script>

<style scoped>
.thumb-wrap {
  width: 100%;
  height: 100%;
}
.stack {
  position: relative;
  width: 100%;
  height: 100%;
}
.img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.ovl {
  position: absolute;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}
.ph {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
.muted {
  color: var(--ag-muted);
  font-size: 11px;
}
</style>
