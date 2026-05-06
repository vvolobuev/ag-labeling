import fs from 'node:fs'
import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

/** Reads SERVER_PORT from ../backend/.env so Vite proxy matches `go run` in backend/. */
function readBackendServerPort() {
  const envPath = fileURLToPath(new URL('../backend/.env', import.meta.url))
  try {
    const text = fs.readFileSync(envPath, 'utf8')
    const line = text.split('\n').find((l) => l.startsWith('SERVER_PORT='))
    const port = line?.slice('SERVER_PORT='.length).trim()
    if (port && /^\d+$/.test(port)) return port
  } catch {
    /* no local .env */
  }
  return '8080'
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    allowedHosts: true,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${readBackendServerPort()}`,
        changeOrigin: true,
        // Large ZIP / folder multipart can take minutes; avoid proxy closing the socket early.
        timeout: 900_000,
        proxyTimeout: 900_000,
      },
    },
  }
})
