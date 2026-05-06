<template>
  <div class="home">
    <div v-if="loading" class="state ag-card">Loading …</div>
    <template v-else>
      <header class="hero">
        <div class="hero-text">
          <p class="kicker">Workspace</p>
          <h1>{{ workspace?.name || 'Workspace' }}</h1>
          <p v-if="workspace?.slug" class="slug muted">/{{ workspace.slug }}</p>
          <p class="lead">
            Manage datasets, annotation batches, and version exports from one place. Use the sidebar to
            switch between projects, explore public listings, and workspace settings.
          </p>
        </div>
        <div class="hero-stats ag-card">
          <div class="st">
            <span class="st-val">{{ stats.projects }}</span>
            <span class="st-lbl">Projects</span>
          </div>
          <div class="st">
            <span class="st-val">{{ stats.images }}</span>
            <span class="st-lbl">Images in active datasets</span>
          </div>
          <div class="st">
            <span class="st-val">{{ stats.publicProjects }}</span>
            <span class="st-lbl">Public projects</span>
          </div>
        </div>
      </header>

      <section class="quick">
        <h2 class="sec-title">Quick actions</h2>
        <div class="cards">
          <router-link class="card ag-card lift-hover" :to="`/app/${wid}/projects`">
            <span class="card-ic" aria-hidden="true">
              <svg viewBox="0 0 24 24"><path fill="currentColor" d="M3 6a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v2H3zm0 4h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
            </span>
            <div>
              <h3>Projects</h3>
              <p>Open datasets, upload data, annotate, and create versions.</p>
            </div>
            <span class="arrow">→</span>
          </router-link>
          <router-link class="card ag-card lift-hover" :to="`/app/${wid}/explore`">
            <span class="card-ic" aria-hidden="true">
              <svg viewBox="0 0 24 24"><path fill="currentColor" d="M10 2a8 8 0 1 1-5.293 14L2.354 18.35l1.414-1.414 2.353-2.353A8 8 0 0 1 10 2m0 2a6 6 0 1 0 .001 12A6 6 0 0 0 10 4"/></svg>
            </span>
            <div>
              <h3>Explore</h3>
              <p>Browse public projects across workspaces you can access.</p>
            </div>
            <span class="arrow">→</span>
          </router-link>
          <router-link class="card ag-card lift-hover" :to="`/app/${wid}/settings`">
            <span class="card-ic" aria-hidden="true">
              <svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8m9 4-.94-.34a7.8 7.8 0 0 0-.39-.94l.5-.86a1 1 0 0 0-.12-1.2l-1.41-1.41a1 1 0 0 0-1.2-.12l-.86.5a8 8 0 0 0-.94-.39L16 3h-2l-.34.94c-.32.11-.64.24-.94.39l-.86-.5a1 1 0 0 0-1.2.12L8.25 5.36l.5.86c-.15.3-.28.62-.39.94L8 9H6l-.34.94c-.32.11-.64.24-.94.39l-.86-.5a1 1 0 0 0-1.2.12L1.25 11.36l.5.86c-.15.3-.28.62-.39.94L1 15v2l.94.34.39.94-.5.86a1 1 0 0 0 .12 1.2l1.41 1.41a1 1 0 0 0 1.2.12l.86-.5c.3.15.62.28.94.39l.34.94h2l.34-.94c.32-.11.64-.24.94-.39l.86.5a1 1 0 0 0 1.2-.12l1.41-1.41a1 1 0 0 0 .12-1.2l-.5-.86c.15-.3.28-.62.39-.94L14 17h2l.34-.94c.32-.11.64-.24.94-.39l.86.5a1 1 0 0 0 1.2-.12l1.41-1.41a1 1 0 0 0 .12-1.2l-.5-.86c.15-.3.28-.62.39-.94z"/></svg>
            </span>
            <div>
              <h3>Settings</h3>
              <p>Members, profile, and workspace preferences.</p>
            </div>
            <span class="arrow">→</span>
          </router-link>
        </div>
      </section>

      <section class="tips ag-card">
        <h2 class="sec-title sm">Tips</h2>
        <ul>
          <li>Use <strong>Dataset</strong> for the working set of images; <strong>Versions</strong> freezes an export for training.</li>
          <li>Assign <strong>train / valid / test</strong> per image before publishing a version.</li>
          <li>Public projects are read-only for guests — keep sensitive data in private projects.</li>
        </ul>
      </section>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'

const route = useRoute()
const api = useApi()
const wid = computed(() => String(route.params.wid || ''))

const loading = ref(true)
const workspace = ref(null)
const projects = ref([])

const stats = computed(() => {
  const list = projects.value || []
  let images = 0
  let publicProjects = 0
  for (const p of list) {
    images += Number(p.image_count) || 0
    if (p.is_public) publicProjects += 1
  }
  return {
    projects: list.length,
    images,
    publicProjects,
  }
})

async function load() {
  loading.value = true
  workspace.value = null
  projects.value = []
  const id = wid.value
  if (!id) {
    loading.value = false
    return
  }
  try {
    const [ws, pr] = await Promise.all([
      api.request(`/workspaces/${id}`),
      api.request(`/workspaces/${id}/projects`),
    ])
    workspace.value = ws
    projects.value = pr.projects || []
  } catch {
    workspace.value = { name: 'Workspace', slug: '' }
    projects.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(wid, load)
</script>

<style scoped>
.home {
  padding: 8px 22px 48px;
  max-width: 1100px;
}

.state {
  padding: 28px 22px;
  text-align: center;
  color: var(--ag-muted);
}

.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr);
  gap: 28px;
  align-items: start;
  margin-bottom: 36px;
}

@media (max-width: 900px) {
  .hero {
    grid-template-columns: 1fr;
  }
}

.kicker {
  margin: 0 0 8px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--ag-accent);
}

.hero h1 {
  margin: 0 0 6px;
  font-size: clamp(26px, 3.5vw, 36px);
  font-weight: 800;
  letter-spacing: -0.03em;
  line-height: 1.15;
}

.slug {
  margin: 0 0 16px;
  font-size: 14px;
}

.lead {
  margin: 0;
  font-size: 15px;
  line-height: 1.65;
  color: var(--ag-muted);
  max-width: 560px;
}

.hero-stats {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 8px 0;
  overflow: hidden;
}

.st {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 18px 22px;
  border-bottom: 1px solid var(--ag-border);
}

.st:last-child {
  border-bottom: 0;
}

.st-val {
  font-size: 28px;
  font-weight: 800;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.st-lbl {
  font-size: 12px;
  color: var(--ag-muted);
  line-height: 1.4;
}

.muted {
  color: var(--ag-muted);
}

.sec-title {
  margin: 0 0 16px;
  font-size: 18px;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.sec-title.sm {
  font-size: 16px;
  margin-bottom: 12px;
}

.quick {
  margin-bottom: 28px;
}

.cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.card {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 16px;
  align-items: center;
  padding: 18px 20px;
  text-decoration: none;
  color: inherit;
  border: 1px solid var(--ag-border);
}

.card:hover {
  border-color: rgba(77, 107, 254, 0.35);
}

.card-ic {
  width: 40px;
  height: 40px;
  color: var(--ag-accent);
  opacity: 0.95;
}

.card-ic svg {
  width: 100%;
  height: 100%;
  display: block;
}

.card h3 {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 700;
}

.card p {
  margin: 0;
  font-size: 13px;
  color: var(--ag-muted);
  line-height: 1.5;
}

.arrow {
  font-size: 18px;
  color: var(--ag-accent);
  opacity: 0.85;
}

.tips {
  padding: 20px 22px 22px;
}

.tips ul {
  margin: 0;
  padding-left: 1.2em;
  color: var(--ag-muted);
  font-size: 14px;
  line-height: 1.65;
}

.tips li {
  margin-bottom: 8px;
}

.tips li:last-child {
  margin-bottom: 0;
}

.tips strong {
  color: var(--ag-text);
  font-weight: 700;
}
</style>
