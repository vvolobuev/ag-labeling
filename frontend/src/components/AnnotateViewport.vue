<template>
  <div ref="wrapRef" class="viewport">
    <div v-if="!blobUrl" class="ph ag-card"><span class="muted">…</span></div>
    <div v-show="blobUrl" class="stack">
      <img
        ref="imgRef"
        :src="blobUrl"
        class="photo"
        draggable="false"
        alt=""
        @load="onImgLoad"
      />
      <svg
        v-if="disp"
        ref="svgRef"
        class="ovl"
        :class="{ hand: interactionMode === 'hand', readonly: readOnly }"
        :viewBox="`0 0 ${vbW} ${vbH}`"
        tabindex="0"
        @pointerdown="onPointerDown"
      >
        <rect
          class="hit-bg"
          x="0"
          y="0"
          :width="vbW"
          :height="vbH"
          fill="transparent"
        />
        <rect
          v-if="hasClassHover"
          class="class-hover-mask"
          x="0"
          y="0"
          :width="vbW"
          :height="vbH"
        />
        <g v-for="bi in bboxRenderIndices" :key="'b-' + bi">
          <rect
            v-if="segments[bi].type === 'bbox'"
            class="bbox-rect"
            :class="{ selected: selectedIdx === bi, dimmed: hasClassHover && !isHoveredClass(bi) }"
            :x="rectPx(bi).x"
            :y="rectPx(bi).y"
            :width="rectPx(bi).w"
            :height="rectPx(bi).h"
            :stroke="strokeOf(bi)"
            :fill="fillOf(bi)"
            @pointerdown.stop="onBBoxPointerDown($event, bi)"
          />
        </g>
        <g
          v-if="interactionMode === 'hand' && selectedIdx !== null && segments[selectedIdx]?.type === 'bbox'"
          class="handles"
        >
          <rect
            v-for="h in handleSpecs"
            :key="h.k"
            class="handle"
            :x="handleRectPx(h.k).x"
            :y="handleRectPx(h.k).y"
            :width="handleRectPx(h.k).w"
            :height="handleRectPx(h.k).h"
            @pointerdown.stop="beginResize($event, h.k)"
          />
        </g>
        <rect
          v-if="draftDraw"
          class="draft"
          :x="draftRectPx.x"
          :y="draftRectPx.y"
          :width="draftRectPx.w"
          :height="draftRectPx.h"
          fill="none"
          stroke="var(--ag-accent)"
          stroke-width="2"
          stroke-dasharray="6 4"
        />
      </svg>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { fetchImageBlob } from '@/composables/useApi'
import {
  ltrbToYoloNorm,
  yoloNormToLTRB,
  hslForClass,
  hslStrokeForClass,
  clampYOLOBox,
} from '@/utils/yoloLabel'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  imageId: { type: String, required: true },
  /** Class for new box (index in names). */
  defaultClassIndex: { type: Number, default: 0 },
  interactionMode: { type: String, default: 'box' }, // 'box' | 'hand'
  hoverClassIndex: { type: Number, default: null },
  selectedIndex: { type: Number, default: null },
  readOnly: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'select'])

const segments = computed({
  get: () => props.modelValue || [],
  set: (v) => emit('update:modelValue', v),
})

const wrapRef = ref(null)
const imgRef = ref(null)
const svgRef = ref(null)
const blobUrl = ref('')
const vbW = ref(400)
const vbH = ref(300)
const disp = ref(null)

const selectedIdx = ref(null)
const grab = ref(null)
const draftDraw = ref(null)

const handleSpecs = [
  { k: 'nw' },
  { k: 'n' },
  { k: 'ne' },
  { k: 'e' },
  { k: 'se' },
  { k: 's' },
  { k: 'sw' },
  { k: 'w' },
]

const HANDLE = 11

const bboxIndices = computed(() => {
  const s = segments.value
  const ix = []
  for (let i = 0; i < s.length; i++) if (s[i].type === 'bbox') ix.push(i)
  return ix
})
const bboxRenderIndices = computed(() => {
  const list = [...bboxIndices.value]
  const si = selectedIdx.value
  if (si === null) return list
  const at = list.indexOf(si)
  if (at < 0) return list
  list.splice(at, 1)
  list.push(si)
  return list
})
const hasClassHover = computed(() => props.hoverClassIndex !== null && props.hoverClassIndex >= 0)

function isHoveredClass(i) {
  const seg = segments.value[i]
  if (seg?.type !== 'bbox') return false
  return Number(seg.cls) === Number(props.hoverClassIndex)
}

function fillOf(i) {
  const c = segments.value[i].cls
  return hslForClass(c)
}

function strokeOf(i) {
  const c = segments.value[i].cls
  return hslStrokeForClass(c)
}

function rectPx(i) {
  const d = disp.value
  const seg = segments.value[i]
  if (!d || !seg || seg.type !== 'bbox') return { x: 0, y: 0, w: 0, h: 0 }
  const L = yoloNormToLTRB(seg)
  const x = d.offX + L.x * d.nw * d.scale
  const y = d.offY + L.y * d.nh * d.scale
  const w = L.w * d.nw * d.scale
  const h = L.h * d.nh * d.scale
  return { x, y, w, h }
}

function handleRectPx(kind) {
  const r = rectPx(selectedIdx.value)
  const q = HANDLE
  const hs = q
  switch (kind) {
    case 'nw':
      return { x: r.x - hs / 2, y: r.y - hs / 2, w: q, h: q }
    case 'n':
      return { x: r.x + r.w / 2 - hs / 2, y: r.y - hs / 2, w: q, h: q }
    case 'ne':
      return { x: r.x + r.w - hs / 2, y: r.y - hs / 2, w: q, h: q }
    case 'e':
      return { x: r.x + r.w - hs / 2, y: r.y + r.h / 2 - hs / 2, w: q, h: q }
    case 'se':
      return { x: r.x + r.w - hs / 2, y: r.y + r.h - hs / 2, w: q, h: q }
    case 's':
      return { x: r.x + r.w / 2 - hs / 2, y: r.y + r.h - hs / 2, w: q, h: q }
    case 'sw':
      return { x: r.x - hs / 2, y: r.y + r.h - hs / 2, w: q, h: q }
    case 'w':
      return { x: r.x - hs / 2, y: r.y + r.h / 2 - hs / 2, w: q, h: q }
    default:
      return { x: 0, y: 0, w: 0, h: 0 }
  }
}

function svgXY(e) {
  const svg = svgRef.value
  if (!svg) return { x: 0, y: 0 }
  const r = svg.getBoundingClientRect()
  const x = ((e.clientX - r.left) / r.width) * vbW.value
  const y = ((e.clientY - r.top) / r.height) * vbH.value
  return { x, y }
}

function hitHandle(px, py) {
  if (selectedIdx.value === null) return null
  if (segments.value[selectedIdx.value]?.type !== 'bbox') return null
  for (const h of handleSpecs) {
    const hr = handleRectPx(h.k)
    if (
      px >= hr.x &&
      px <= hr.x + hr.w &&
      py >= hr.y &&
      py <= hr.y + hr.h
    ) {
      return h.k
    }
  }
  return null
}

function pointInBBox(i, px, py) {
  const r = rectPx(i)
  return px >= r.x && px <= r.x + r.w && py >= r.y && py <= r.y + r.h
}

function hitBBoxIndex(px, py) {
  if (selectedIdx.value !== null && segments.value[selectedIdx.value]?.type === 'bbox') {
    if (pointInBBox(selectedIdx.value, px, py)) return selectedIdx.value
  }
  const order = [...bboxIndices.value].reverse()
  for (const i of order) {
    if (pointInBBox(i, px, py)) return i
  }
  return null
}

/** @typedef {{nw:{x:number,y:number}, cx:number, cy:number, w:number, h:number}} GrabMove */
/** norm delta from accumulated pixel delta */

function pxToNormDelta(dpx, dpy) {
  const d = disp.value
  if (!d) return { dnx: 0, dny: 0 }
  return { dnx: dpx / (d.nw * d.scale), dny: dpy / (d.nh * d.scale) }
}

function copySegList() {
  return segments.value.map((s) =>
    s.type === 'bbox' ? { ...s } : { ...s },
  )
}

function onImgLoad() {
  nextTick(() => layout())
}

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
  const dw = nw * scale
  const dh = nh * scale
  disp.value = {
    nw,
    nh,
    scale,
    offX: (cw - dw) / 2,
    offY: (ch - dh) / 2,
  }
}

let ro

async function loadBlob() {
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
  () => {
    selectedIdx.value = null
    draftDraw.value = null
    grab.value = null
    loadBlob()
  },
  { immediate: true },
)

onMounted(() => {
  ro = new ResizeObserver(() => layout())
  if (imgRef.value) ro.observe(imgRef.value)
  window.addEventListener('keydown', onKey)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('pointercancel', onPointerUp)
  window.addEventListener('pointermove', onPointerMove)
})

onBeforeUnmount(() => {
  if (ro && imgRef.value) ro.unobserve(imgRef.value)
  ro = null
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('pointercancel', onPointerUp)
  window.removeEventListener('pointermove', onPointerMove)
  if (blobUrl.value) URL.revokeObjectURL(blobUrl.value)
})

function onKey(e) {
  if (e.target?.tagName === 'TEXTAREA' || e.target?.tagName === 'INPUT') return
  if (e.key !== 'Delete' && e.code !== 'Backspace') return
  if (selectedIdx.value === null) return
  const i = selectedIdx.value
  if (segments.value[i]?.type !== 'bbox') return
  e.preventDefault()
  const next = copySegList()
  next.splice(i, 1)
  segments.value = next
  selectedIdx.value = null
}

function selectAndMaybeMove(e, idx) {
  if (props.interactionMode !== 'hand') return
  selectedIdx.value = idx
  const { x, y } = svgXY(e)
  grab.value = {
    type: 'move',
    idx,
    cx0: segments.value[idx].cx,
    cy0: segments.value[idx].cy,
    px0: x,
    py0: y,
    client0: { x: e.clientX, y: e.clientY },
  }
  e.target.setPointerCapture?.(e.pointerId)
}

function onBBoxPointerDown(e, idx) {
  const { x, y } = svgXY(e)
  const selectedHit =
    selectedIdx.value !== null &&
    segments.value[selectedIdx.value]?.type === 'bbox' &&
    pointInBBox(selectedIdx.value, x, y)
      ? selectedIdx.value
      : null
  const targetIdx = selectedHit !== null ? selectedHit : idx
  if (props.readOnly) {
    selectedIdx.value = targetIdx
    return
  }
  if (props.interactionMode === 'box') {
    if (!disp.value) return
    selectedIdx.value = null
    grab.value = { type: 'draw', x0: x, y0: y, client0: { x: e.clientX, y: e.clientY } }
    draftDraw.value = { x0: x, y0: y, x1: x, y1: y }
    e.target.setPointerCapture?.(e.pointerId)
    return
  }
  selectAndMaybeMove(e, targetIdx)
}

function beginResize(e, handle) {
  if (props.interactionMode !== 'hand') return
  const i = selectedIdx.value
  if (i === null || segments.value[i]?.type !== 'bbox') return
  const L = yoloNormToLTRB(segments.value[i])
  grab.value = {
    type: 'resize',
    idx: i,
    handle,
    ltrb0: { ...L },
    client0: { x: e.clientX, y: e.clientY },
  }
  e.target.setPointerCapture?.(e.pointerId)
}

function onPointerDown(e) {
  if (!disp.value) return
  const { x, y } = svgXY(e)
  if (props.readOnly) {
    const hit = hitBBoxIndex(x, y)
    selectedIdx.value = hit
    return
  }
  if (props.interactionMode === 'box') {
    selectedIdx.value = null
    grab.value = { type: 'draw', x0: x, y0: y, client0: { x: e.clientX, y: e.clientY } }
    draftDraw.value = { x0: x, y0: y, x1: x, y1: y }
    e.currentTarget.setPointerCapture?.(e.pointerId)
    return
  }

  const h = hitHandle(x, y)
  if (h) {
    beginResize(e, h)
    return
  }

  const hit = hitBBoxIndex(x, y)
  if (hit !== null) {
    selectedIdx.value = hit
    if (props.interactionMode !== 'hand') return
    grab.value = {
      type: 'move',
      idx: hit,
      cx0: segments.value[hit].cx,
      cy0: segments.value[hit].cy,
      px0: x,
      py0: y,
      client0: { x: e.clientX, y: e.clientY },
    }
    e.currentTarget.setPointerCapture?.(e.pointerId)
    return
  }

  selectedIdx.value = null
}

function onPointerMove(e) {
  const g = grab.value
  if (!g || !disp.value) return

  if (g.type === 'draw' && draftDraw.value) {
    const { x, y } = svgXY(e)
    draftDraw.value = { ...draftDraw.value, x1: x, y1: y }
    return
  }

  if (g.type === 'move') {
    const dpx = e.clientX - g.client0.x
    const dpy = e.clientY - g.client0.y
    const { dnx, dny } = pxToNormDelta(dpx, dpy)
    const next = copySegList()
    const seg = next[g.idx]
    if (seg?.type !== 'bbox') return
    const yo = clampYOLOBox({
      cls: seg.cls,
      cx: g.cx0 + dnx,
      cy: g.cy0 + dny,
      w: seg.w,
      h: seg.h,
    })
    seg.cx = yo.cx
    seg.cy = yo.cy
    seg.w = yo.w
    seg.h = yo.h
    segments.value = next
    return
  }

  if (g.type === 'resize') {
    const dpx = e.clientX - g.client0.x
    const dpy = e.clientY - g.client0.y
    const { dnx, dny } = pxToNormDelta(dpx, dpy)
    let { x, y, w, h } = g.ltrb0

    switch (g.handle) {
      case 'nw':
        x += dnx
        y += dny
        w -= dnx
        h -= dny
        break
      case 'n':
        y += dny
        h -= dny
        break
      case 'ne':
        y += dny
        w += dnx
        h -= dny
        break
      case 'e':
        w += dnx
        break
      case 'se':
        w += dnx
        h += dny
        break
      case 's':
        h += dny
        break
      case 'sw':
        x += dnx
        w -= dnx
        h += dny
        break
      case 'w':
        x += dnx
        w -= dnx
        break
      default:
        break
    }

    if (w < 0.0025) w = 0.0025
    if (h < 0.0025) h = 0.0025
    if (x < 0) {
      w += x
      x = 0
    }
    if (y < 0) {
      h += y
      y = 0
    }
    if (x + w > 1) w = 1 - x
    if (y + h > 1) h = 1 - y

    const yolo = ltrbToYoloNorm({ x, y, w, h })
    const next = copySegList()
    const seg = next[g.idx]
    if (seg?.type === 'bbox') {
      seg.cx = yolo.cx
      seg.cy = yolo.cy
      seg.w = yolo.w
      seg.h = yolo.h
      segments.value = next
    }
  }
}

const draftRectPx = computed(() => {
  const d = draftDraw.value
  if (!d) return { x: 0, y: 0, w: 0, h: 0 }
  const x1 = Math.min(d.x0, d.x1)
  const y1 = Math.min(d.y0, d.y1)
  const x2 = Math.max(d.x0, d.x1)
  const y2 = Math.max(d.y0, d.y1)
  return { x: x1, y: y1, w: Math.max(1, x2 - x1), h: Math.max(1, y2 - y1) }
})

function onPointerUp() {
  const g = grab.value
  grab.value = null

  if (g?.type === 'draw' && draftDraw.value && disp.value) {
    const d = draftDraw.value
    draftDraw.value = null
    const x1 = Math.min(d.x0, d.x1)
    const y1 = Math.min(d.y0, d.y1)
    const x2 = Math.max(d.x0, d.x1)
    const y2 = Math.max(d.y0, d.y1)
    const di = disp.value
    const nx = (x1 - di.offX) / (di.nw * di.scale)
    const ny = (y1 - di.offY) / (di.nh * di.scale)
    const nw = (x2 - x1) / (di.nw * di.scale)
    const nh = (y2 - y1) / (di.nh * di.scale)
    if (nw < 0.004 || nh < 0.004) return
    const yolo = ltrbToYoloNorm({ x: nx, y: ny, w: nw, h: nh })
    const next = copySegList()
    const defaultCls = Math.max(0, Math.floor(props.defaultClassIndex))
    next.push({ type: 'bbox', cls: defaultCls, ...yolo })
    segments.value = next
    selectedIdx.value = next.length - 1
  }
}

watch(blobUrl, () =>
  nextTick(() => {
    if (ro && imgRef.value) {
      ro.disconnect()
      ro.observe(imgRef.value)
    }
    layout()
  }),
)

watch(selectedIdx, (v) => emit('select', v))
watch(
  () => props.selectedIndex,
  (v) => {
    if (v === selectedIdx.value) return
    selectedIdx.value = v == null ? null : v
  },
)
</script>

<style scoped>
.viewport {
  width: 100%;
}

.stack {
  position: relative;
  display: block;
  width: 100%;
}

.photo {
  display: block;
  width: 100%;
  max-height: min(560px, 70vh);
  height: auto;
  object-fit: contain;
  margin: 0 auto;
  user-select: none;
}

.ovl {
  position: absolute;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  pointer-events: auto;
  outline: none;
}
.ovl.hand { cursor: grab; }

.hit-bg {
  pointer-events: all;
}

.bbox-rect {
  pointer-events: visiblePainted;
  stroke-width: 2;
  fill-opacity: 0.12;
  cursor: move;
}
.ovl:not(.hand) .bbox-rect { cursor: default; }
.ovl.readonly .bbox-rect { cursor: default; }

.bbox-rect.selected {
  stroke-width: 3;
  fill-opacity: 0.2;
}
.bbox-rect.dimmed {
  opacity: 0.18;
}
.class-hover-mask {
  fill: rgba(0, 0, 0, 0.45);
  pointer-events: none;
}

.handle {
  fill: var(--ag-bg, #1a1d24);
  stroke: var(--ag-accent);
  stroke-width: 2;
  cursor: nwse-resize;
  pointer-events: all;
}

.ph {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}

.muted {
  color: var(--ag-muted);
}

</style>
