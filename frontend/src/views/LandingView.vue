<template>
  <div class="landing">
    <div class="hero">
      <AgHeader />
      <div class="hero-body">
        <p class="pill ag-pill fade-slow-enter-from">Data annotation platform</p>
        <h1 class="title">
          <span class="line">Image annotation,</span>
          <span class="line accent">dataset versions, teams.</span>
        </h1>
        <p class="sub">
          ALPHA GUARD AI - workspaces, projects, and YOLO / Roboflow-structure import. Roles as in Roboflow:
          owner, admin, annotator, and viewer.
        </p>
        <div class="cta">
          <router-link class="ag-btn ag-btn-primary" to="/register">Create account</router-link>
          <router-link class="ag-btn ag-btn-ghost" to="/login">Sign in</router-link>
        </div>
        <div class="grid">
          <article v-for="f in features" :key="f.t" class="ag-card feature lift-hover">
            <h3>{{ f.t }}</h3>
            <p>{{ f.d }}</p>
          </article>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import AgHeader from '@/components/AgHeader.vue'

const features = [
  { t: 'Import', d: 'ZIP archive or folder with data.yaml, train/valid/test and labels in original plain text.' },
  { t: 'Workspace', d: 'Isolated workspaces and projects. Invite your team by email and role.' },
  { t: 'Annotation', d: 'Image viewer and annotation editor: raw YOLO text exactly as on disk.' },
]
</script>

<style scoped>
.landing {
  min-height: 100vh;
  background: radial-gradient(1200px 800px at 10% -10%, rgba(77, 107, 254, 0.35), transparent 55%),
    radial-gradient(900px 600px at 100% 0%, rgba(77, 107, 254, 0.22), transparent 50%), var(--ag-bg);
}

.hero-body {
  max-width: 1100px;
  margin: 0 auto;
  padding: 28px 22px 80px;
}

.pill {
  margin-bottom: 18px;
  display: inline-flex;
}

.title {
  margin: 0 0 16px;
  font-size: clamp(28px, 5vw, 46px);
  font-weight: 800;
  line-height: 1.12;
}

.line {
  display: block;
}

.accent {
  color: var(--ag-accent);
}

.sub {
  max-width: 640px;
  color: var(--ag-muted);
  font-size: 16px;
  line-height: 1.65;
  margin-bottom: 28px;
}

.cta {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 40px;
}

.cta .ag-btn {
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.feature h3 {
  margin: 0 0 10px;
  font-size: 16px;
  font-weight: 700;
}

.feature p {
  margin: 0;
  font-size: 14px;
  color: var(--ag-muted);
  line-height: 1.55;
}
</style>
