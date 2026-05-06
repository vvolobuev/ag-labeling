const TOKEN_KEY = 'ag_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(t) {
  if (t) localStorage.setItem(TOKEN_KEY, t)
  else localStorage.removeItem(TOKEN_KEY)
}

export function useApi() {
  const base = '/api'

  async function request(path, opts = {}) {
    const headers = { ...(opts.headers || {}) }
    const tok = getToken()
    if (tok) headers.Authorization = `Bearer ${tok}`
    if (opts.body && !(opts.body instanceof FormData) && !headers['Content-Type']) {
      headers['Content-Type'] = 'application/json'
    }
    const res = await fetch(base + path, { ...opts, headers })
    const text = await res.text()
    let data
    try {
      data = text ? JSON.parse(text) : null
    } catch {
      data = { raw: text }
    }
    if (!res.ok) {
      const msg = (data && data.error) || res.statusText || 'request failed'
      const err = new Error(msg)
      err.status = res.status
      err.data = data
      throw err
    }
    return data
  }

  return { request, getToken, setToken }
}

export async function fetchImageBlob(path) {
  const headers = {}
  const tok = getToken()
  if (tok) headers.Authorization = `Bearer ${tok}`
  const res = await fetch('/api' + path, { headers })
  if (!res.ok) throw new Error('image load failed')
  return res.blob()
}
