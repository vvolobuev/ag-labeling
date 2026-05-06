<template>
  <div class="layout">
    <div class="shell">
      <aside class="primary" :class="{ collapsed: inProjectScope }">
        <div class="brand-block">
          <router-link class="brand" :to="`/app/${wid}/home`">
            ALPHA GUARD <span class="ai">AI</span>
          </router-link>
        </div>
        <button class="workspace-switch" type="button" @click="!inProjectScope && (menuOpen = !menuOpen)" :title="currentWorkspace?.name || 'Workspace'">
          <span v-if="!inProjectScope" class="workspace-name">{{ currentWorkspace?.name || 'Workspace' }}</span>
          <span v-else class="workspace-icon ag-mark">AG</span>
          <span v-if="!inProjectScope" class="workspace-caret">▾</span>
        </button>
        <div v-if="menuOpen && !inProjectScope" class="workspace-menu ag-card">
          <router-link v-for="w in workspaces" :key="w.id" class="workspace-row" :to="`/app/${w.id}/projects`">
            {{ w.name }}
          </router-link>
          <router-link class="workspace-row add" to="/workspaces">+ Add workspace</router-link>
        </div>
        <nav class="nav">
          <router-link class="nav-item" :to="`/app/${wid}/home`">
            <svg class="ic-svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 3l9 8h-3v9h-5v-5H11v5H6v-9H3z"/></svg>
            <span v-if="!inProjectScope">Home</span>
          </router-link>
          <router-link class="nav-item" :to="`/app/${wid}/projects`">
            <svg class="ic-svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 6a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v2H3z"/><path fill="currentColor" d="M3 10h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
            <span v-if="!inProjectScope">Projects</span>
          </router-link>
          <router-link class="nav-item" :to="`/app/${wid}/explore`">
            <svg class="ic-svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M10 2a8 8 0 1 1-5.293 13.999L2.354 18.35l1.414 1.414 2.353-2.353A8 8 0 0 1 10 2m0 2a6 6 0 1 0 .001 12.001A6 6 0 0 0 10 4"/></svg>
            <span v-if="!inProjectScope">Explore</span>
          </router-link>
          <router-link class="nav-item" :to="`/app/${wid}/settings`">
            <svg class="ic-svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8m9 4-.94-.34a7.8 7.8 0 0 0-.39-.94l.5-.86a1 1 0 0 0-.12-1.2l-1.41-1.41a1 1 0 0 0-1.2-.12l-.86.5c-.3-.15-.62-.28-.94-.39L16 3h-2l-.34.94c-.32.11-.64.24-.94.39l-.86-.5a1 1 0 0 0-1.2.12L9.25 5.36a1 1 0 0 0-.12 1.2l.5.86c-.15.3-.28.62-.39.94L8 9H6l-.34.94c-.32.11-.64.24-.94.39l-.86-.5a1 1 0 0 0-1.2.12L1.25 11.36a1 1 0 0 0-.12 1.2l.5.86c-.15.3-.28.62-.39.94L1 15v2l.94.34c.11.32.24.64.39.94l-.5.86a1 1 0 0 0 .12 1.2l1.41 1.41a1 1 0 0 0 1.2.12l.86-.5c.3.15.62.28.94.39L6 23h2l.34-.94c.32-.11.64-.24.94-.39l.86.5a1 1 0 0 0 1.2-.12l1.41-1.41a1 1 0 0 0 .12-1.2l-.5-.86c.15-.3.28-.62.39-.94L14 17h2l.34-.94c.32-.11.64-.24.94-.39l.86.5a1 1 0 0 0 1.2-.12l1.41-1.41a1 1 0 0 0 .12-1.2l-.5-.86c.15-.3.28-.62.39-.94z"/></svg>
            <span v-if="!inProjectScope">Settings</span>
          </router-link>
        </nav>
        <div class="spacer" />
        <div ref="accountWrapRef" class="account-wrap">
          <button type="button" class="account-row" @click="accountOpen = !accountOpen">
            <img v-if="avatarUrl" class="avatar" :src="avatarUrl + '?t=' + avatarCache" :title="fullName" alt="" />
            <div v-else class="avatar" :title="fullName">{{ avatarText }}</div>
            <div v-if="!inProjectScope" class="acc-name">{{ fullName }}</div>
          </button>
          <div v-if="accountOpen" class="account-menu ag-card">
            <div class="account-head">
              <img v-if="avatarUrl" class="avatar lg" :src="avatarUrl + '?t=' + avatarCache" alt="" />
              <div v-else class="avatar lg">{{ avatarText }}</div>
              <div>
                <p class="account-name">{{ fullName }}</p>
                <p class="account-email">{{ me.email || '' }}</p>
              </div>
            </div>
            <router-link class="account-link" :to="`/app/${wid}/settings`">Account Settings</router-link>
            <button type="button" class="account-link btn-like" @click="logout">Sign Out</button>
          </div>
        </div>
      </aside>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { setToken, useApi } from '@/composables/useApi'

const api = useApi()
const route = useRoute()
const router = useRouter()
const wid = computed(() => String(route.params.wid || ''))
const inProjectScope = computed(() => !!route.params.pid)
const menuOpen = ref(false)
const workspaces = ref([])
const me = ref({ first_name: '', last_name: '', email: '' })
const accountOpen = ref(false)
const avatarUrl = ref('')
const avatarCache = ref(Date.now())
const accountWrapRef = ref(null)

const currentWorkspace = computed(() => workspaces.value.find((w) => String(w.id) === wid.value) || null)
const fullName = computed(() => {
  const f = String(me.value.first_name || '').trim()
  const l = String(me.value.last_name || '').trim()
  const both = `${f} ${l}`.trim()
  if (both) return both
  const email = String(me.value.email || '').trim()
  if (!email) return 'User Account'
  const left = email.split('@')[0] || 'user'
  return left
    .split(/[._-]+/)
    .filter(Boolean)
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join(' ')
})
const avatarText = computed(() => {
  return fullName.value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((x) => x[0]?.toUpperCase() || '')
    .join('')
})

function initials(v) {
  return String(v || 'W')
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase() || '')
    .join('')
}

async function loadWorkspaces() {
  const data = await api.request('/workspaces')
  workspaces.value = data.workspaces || []
}

async function loadMe() {
  const data = await api.request('/me')
  me.value = {
    first_name: data.first_name || '',
    last_name: data.last_name || '',
    email: data.email || '',
  }
  avatarUrl.value = data.avatar_url || ''
  avatarCache.value = Date.now()
}

function logout() {
  setToken('')
  localStorage.removeItem('ag_last_workspace')
  router.push('/')
}

function onDocPointerDown(e) {
  if (!accountOpen.value) return
  const wrap = accountWrapRef.value
  if (!wrap) return
  const t = e.target
  if (t instanceof Node && wrap.contains(t)) return
  accountOpen.value = false
}

watch(
  () => route.fullPath,
  () => {
    menuOpen.value = false
    accountOpen.value = false
    if (wid.value) localStorage.setItem('ag_last_workspace', wid.value)
  },
)

onMounted(() => {
  if (wid.value) localStorage.setItem('ag_last_workspace', wid.value)
  loadWorkspaces().catch(() => {})
  loadMe().catch(() => {})
  document.addEventListener('pointerdown', onDocPointerDown, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointerDown, true)
})
</script>

<style scoped>
.layout { min-height: 100vh; background: var(--ag-bg); }
.shell { display: grid; grid-template-columns: auto 1fr; min-height: 100vh; }
.primary {
  width: 196px;
  border-right: 1px solid var(--ag-border);
  padding: 12px 8px;
  position: sticky;
  top: 0;
  height: 100vh;
  background: var(--ag-bg);
  display: flex;
  flex-direction: column;
  z-index: 2000;
  overflow: visible;
}
.primary.collapsed { width: 72px; }
.brand-block { margin-bottom: 22px; padding: 4px 6px 8px; }
.brand { font-weight: 800; letter-spacing: 0.14em; font-size: 14px; color: #fff; text-decoration: none; }
.ai { color: var(--ag-accent); }
.workspace-switch { width: 100%; border: 1px solid var(--ag-border); background: rgba(255,255,255,.04); color: var(--ag-text); border-radius: 10px; padding: 10px; display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.workspace-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.workspace-icon { margin: 0 auto; font-weight: 700; }
.ag-mark { color: var(--ag-accent); letter-spacing: 0.02em; }
.workspace-menu { margin-top: 8px; padding: 8px; display: flex; flex-direction: column; gap: 6px; }
.workspace-row { color: var(--ag-text); text-decoration: none; font-size: 12px; padding: 6px; border-radius: 8px; }
.workspace-row:hover { background: rgba(77, 107, 254, 0.14); }
.workspace-row.add { color: var(--ag-accent); }
.nav { margin-top: 12px; display: flex; flex-direction: column; gap: 6px; }
.nav-item { color: var(--ag-muted); text-decoration: none; border-radius: 10px; padding: 8px; display: flex; align-items: center; gap: 9px; font-size: 12px; font-weight: 500; }
.nav-item.router-link-active { color: var(--ag-text); background: rgba(77, 107, 254, 0.18); }
.primary.collapsed .nav-item { justify-content: center; padding: 10px 0; gap: 0; }
.primary.collapsed .brand-block { display: none; }
.primary.collapsed .workspace-switch { padding-left: 0; padding-right: 0; justify-content: center; }
.ic-svg { width: 18px; height: 18px; display: block; }
.spacer { flex: 1; }
.account-wrap { position: relative; margin-top: 8px; }
.account-row { width: 100%; border: 1px solid var(--ag-border); border-radius: 10px; background: rgba(255,255,255,.03); padding: 6px; display: flex; align-items: center; gap: 8px; text-align: left; cursor: pointer; font-family: inherit; }
.account-row:hover { border-color: var(--ag-accent); }
.avatar { width: 30px; height: 30px; border-radius: 999px; background: rgba(77, 107, 254, 0.24); color: #d9e3ff; display: inline-flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 700; flex: 0 0 auto; }
.avatar.lg { width: 36px; height: 36px; font-size: 12px; }
.acc-name { flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-size: 13px; color: var(--ag-text); }
.account-menu {
  position: absolute;
  left: calc(100% + 12px);
  bottom: 0;
  width: 300px;
  padding: 10px;
  z-index: 9999;
  box-shadow: 0 26px 70px rgba(0, 0, 0, 0.58);
}
.account-head { display: flex; gap: 9px; align-items: center; padding-bottom: 8px; border-bottom: 1px solid var(--ag-border); margin-bottom: 6px; }
.account-name { margin: 0; font-size: 13px; font-weight: 600; color: var(--ag-text); }
.account-email { margin: 3px 0 0; font-size: 12px; color: var(--ag-muted); word-break: break-all; }
.account-link { display: block; text-decoration: none; color: var(--ag-text); font-size: 13px; padding: 8px; border-radius: 8px; }
.account-link:hover { background: rgba(77, 107, 254, 0.14); }
.btn-like { width: 100%; text-align: left; border: none; background: transparent; font-family: inherit; cursor: pointer; }
.primary.collapsed .acc-name { display: none; }
.primary.collapsed .account-row { justify-content: center; padding: 6px 4px; gap: 4px; }
.primary.collapsed .account-menu { left: calc(100% + 10px); width: 280px; }
.content { min-width: 0; background: var(--ag-bg); position: relative; z-index: 1; }
</style>
