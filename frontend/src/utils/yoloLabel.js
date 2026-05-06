const BBOX_LINE = /^(\d+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s*$/

function fmtCoord(x) {
  const n = Number(x)
  let s = n.toFixed(6)
  s = s.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
  return s
}

export function clampYOLOBox(s) {
  let cx = Math.min(1, Math.max(0, s.cx))
  let cy = Math.min(1, Math.max(0, s.cy))
  let w = Math.max(s.w, 0.002)
  let h = Math.max(s.h, 0.002)
  cx = Math.min(cx, 1 - w / 2)
  cx = Math.max(cx, w / 2)
  cy = Math.min(cy, 1 - h / 2)
  cy = Math.max(cy, h / 2)
  w = Math.min(w, 2 * cx, 2 * (1 - cx))
  h = Math.min(h, 2 * cy, 2 * (1 - cy))
  return { cx, cy, w, h }
}

export function parseDatasetClassNames(txt) {
  if (!txt || !String(txt).trim()) return []
  const lines = txt.split(/\r?\n/)

  for (const line of lines) {
    const t = line.trim()
    const m = t.match(/^names:\s*\[(.*)\]\s*$/)
    if (m) {
      return m[1]
        .split(',')
        .map((x) => x.trim().replace(/^['"]|['"]$/g, ''))
        .filter(Boolean)
    }
  }

  const start = lines.findIndex((l) => /^\s*names:\s*$/.test(l))
  if (start >= 0) {
    const out = []
    for (let i = start + 1; i < lines.length; i++) {
      const l = lines[i]
      if (/^\s*[\w-]+:/.test(l) && !/^\s*\d+\s*:/.test(l)) break
      const dm = l.match(/^\s*-\s*(.+)\s*$/)
      if (dm) {
        out.push(dm[1].trim().replace(/^['"]|['"]$/g, ''))
        continue
      }
      const km = l.match(/^\s*(\d+)\s*:\s*(.+?)\s*$/)
      if (km) {
        const idx = parseInt(km[1], 10)
        const name = km[2].trim().replace(/^['"]|['"]$/g, '')
        while (out.length <= idx) out.push(`class_${out.length}`)
        out[idx] = name
      }
    }
    return out.filter(Boolean)
  }

  return []
}

export function parseLabelSegments(text) {
  const segments = []
  const lines = (text || '').split(/\r?\n/)
  for (const line of lines) {
    const tr = line.trim()
    if (tr === '') continue
    if (/^\s*#/.test(line)) {
      segments.push({ type: 'raw', text: line })
      continue
    }
    const m = tr.match(BBOX_LINE)
    if (m) {
      const cls = parseInt(m[1], 10)
      const cx = parseFloat(m[2])
      const cy = parseFloat(m[3])
      const w = parseFloat(m[4])
      const h = parseFloat(m[5])
      if (Number.isNaN(cls) || [cx, cy, w, h].some((n) => Number.isNaN(n))) {
        segments.push({ type: 'raw', text: line })
        continue
      }
      segments.push({ type: 'bbox', cls, ...clampYOLOBox({ cx, cy, w, h }) })
    } else {
      segments.push({ type: 'raw', text: line })
    }
  }
  return segments
}

export function serializeLabelSegments(segments) {
  return segments
    .map((seg) => {
      if (seg.type === 'bbox') {
        const b = clampYOLOBox(seg)
        return `${seg.cls} ${fmtCoord(b.cx)} ${fmtCoord(b.cy)} ${fmtCoord(b.w)} ${fmtCoord(b.h)}`
      }
      return seg.text
    })
    .join('\n')
}

export function yoloNormToLTRB(norm) {
  const b = clampYOLOBox(norm)
  return { x: b.cx - b.w / 2, y: b.cy - b.h / 2, w: b.w, h: b.h }
}

export function ltrbToYoloNorm(ltrb) {
  let { x, y, w, h } = ltrb
  x = Math.min(1, Math.max(0, x))
  y = Math.min(1, Math.max(0, y))
  w = Math.max(0.002, w)
  h = Math.max(0.002, h)
  if (x + w > 1) w = 1 - x
  if (y + h > 1) h = 1 - y
  return clampYOLOBox({ cx: x + w / 2, cy: y + h / 2, w, h })
}

export function hslForClass(cls) {
  const h = (cls * 47) % 360
  return `hsla(${h}, 78%, 55%, 0.95)`
}

export function hslStrokeForClass(cls) {
  const h = (cls * 47) % 360
  return `hsla(${h}, 85%, 42%, 1)`
}
