/** Older API builds returned {"error":"upload"}; sometimes a plain "upload" without JSON. */
export function normalizeImportError(msg) {
  const t = String(msg ?? '').trim()
  if (/^upload$/i.test(t) || /^upload:\s*upload$/i.test(t)) {
    return 'Upload error: wrong backend responded (old build or another service on the port). Stop foreign process on this port, restart API from alpha-guard-ai (go run / docker) and npm run dev; port is SERVER_PORT in backend/.env.'
  }
  return t
}

/**
 * POST multipart/form-data via XMLHttpRequest (fetch does not expose upload progress).
 */
export function postMultipartWithProgress(url, formData, opts = {}) {
  const { headers = {}, timeoutMs = 900_000, onUploadProgress, onUploadComplete, onRequestReady } = opts

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', url)
    xhr.timeout = timeoutMs
    onRequestReady?.(xhr)

    xhr.upload.addEventListener('progress', (e) => {
      if (!onUploadProgress) return
      if (!e.lengthComputable || e.total <= 0) {
        onUploadProgress({ lengthComputable: false, loaded: e.loaded, percent: null })
        return
      }
      const percent = Math.min(100, Math.round((100 * e.loaded) / e.total))
      onUploadProgress({ lengthComputable: true, loaded: e.loaded, total: e.total, percent })
    })

    xhr.upload.addEventListener('load', () => {
      onUploadComplete?.()
    })

    xhr.addEventListener('load', () => resolve(xhr))
    xhr.addEventListener('error', () => reject(new Error('Failed to upload file (network error)')))
    xhr.addEventListener('abort', () => reject(new Error('Loading aborted')))
    xhr.addEventListener('timeout', () =>
      reject(
        new Error(
          'Upload timeout. Reduce archive size or increase proxy timeout (nginx / Vite).',
        ),
      ),
    )

    for (const [k, v] of Object.entries(headers)) {
      xhr.setRequestHeader(k, v)
    }
    xhr.send(formData)
  })
}

/** Human-readable failure from finished XHR (JSON error or HTML/proxy pitfalls). */
export function messageFromXHR(xhr) {
  const txt = xhr.responseText || ''
  try {
    const j = JSON.parse(txt)
    if (j && j.error != null && String(j.error).trim()) {
      let m = String(j.error)
      if (j.hint != null && String(j.hint).trim()) m += ' ' + String(j.hint).trim()
      return normalizeImportError(m)
    }
  } catch {
    /* not JSON */
  }
  const s = xhr.status
  const st = (xhr.statusText || '').trim()
  if (s === 413) {
    return normalizeImportError(
      'File is too large (HTTP 413). Increase nginx client_max_body_size or use a smaller ZIP.',
    )
  }
  if (s === 502 || s === 503 || s === 504) {
    return normalizeImportError(
      `Server/proxy response ${s} ${st}. Archive import may have timed out - check nginx proxy_*_timeout.`,
    )
  }

  const head = txt.trim().slice(0, 480)
  if (!head) return normalizeImportError(st ? `${st} (${s})` : `HTTP ${s || '?'}`)

  const low = head.toLowerCase()
  if (low.includes('<html') || low.includes('<!doctype'))
    return normalizeImportError(`HTML returned instead of JSON (often nginx/proxy). HTTP ${s} ${st}`.trim())

  const short = head.replace(/\s+/g, ' ')
  return normalizeImportError(st ? `${st}: ${short}` : short)
}

/**
 * Poll import job after multipart upload.
 * @returns {Promise<object>} final status object with result on success
 */
export async function pollImportJob(jobId, opts = {}) {
  const token = opts.token ?? null
  const intervalMs = opts.intervalMs ?? 420
  const maxMs = opts.maxMs ?? 900_000
  const onTick = opts.onTick
  const shouldStop = opts.shouldStop
  const t0 = Date.now()
  const headers = {}
  if (token) headers.Authorization = `Bearer ${token}`

  while (maxMs <= 0 || Date.now() - t0 < maxMs) {
    if (shouldStop?.()) throw new Error('Import cancelled')
    const r = await fetch(`/api/import-jobs/${encodeURIComponent(jobId)}`, { headers })
    const txt = await r.text()
    if (!r.ok) {
      const shim = { status: r.status, statusText: r.statusText || '', responseText: txt || '' }
      let m = messageFromXHR(shim)
      try {
        const j = JSON.parse(txt || '{}')
        if (j?.error != null && String(j.error).trim()) m = normalizeImportError(String(j.error))
      } catch {
        /* noop */
      }
      throw new Error(m)
    }
    let status = {}
    try {
      status = txt ? JSON.parse(txt) : {}
    } catch {
      throw new Error(normalizeImportError('Invalid import status JSON'))
    }
    onTick?.(status)
    if (status.done) {
      if (status.phase === 'error') {
        let m = normalizeImportError(status.error || 'Import error')
        const body = status.error_body
        if (body && typeof body === 'object') {
          if (body.hint != null && String(body.hint).trim())
            m = `${m} ${String(body.hint).trim()}`
        }
        const err = new Error(m)
        err.importStatus = status
        throw err
      }
      return status
    }
    if (shouldStop?.()) throw new Error('Import cancelled')
    await new Promise((resolve) => setTimeout(resolve, intervalMs))
  }
  throw new Error(
    normalizeImportError(
      'Server import timeout exceeded. Check backend logs and nginx proxy_*_timeout.',
    ),
  )
}

/** Percent for second bar (server): -1 / NaN -> unknown, else 0-100. */
export function serverJobBarPercent(status) {
  if (status == null || typeof status !== 'object') return null
  const p = status.percent
  if (typeof p !== 'number' || p < 0) return null
  return Math.min(100, Math.round(p))
}
