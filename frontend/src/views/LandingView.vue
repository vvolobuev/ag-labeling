<template>
  <div class="landing">
    <div class="hero">
      <AgHeader />
      <div class="hero-grid">
        <div class="hero-copy">
          <p class="pill ag-pill fade-slow-enter-from">Data annotation platform</p>
          <h1 class="title">
            <span class="line">Ship better vision models</span>
            <span class="line accent">with precise labels &amp; dataset versions.</span>
          </h1>
          <p class="sub">
            Workspaces, roles, YOLO text labels, Roboflow-style imports, and immutable version exports —
            everything teams need from first prototype to production datasets.
          </p>
          <div class="cta">
            <router-link class="ag-btn ag-btn-primary" to="/register">Create account</router-link>
            <router-link class="ag-btn ag-btn-ghost" to="/login">Sign in</router-link>
          </div>
          <ul class="trust">
            <li>
              <span class="t-num">YOLO</span>
              <span class="t-lbl">Native label text</span>
            </li>
            <li>
              <span class="t-num">ZIP</span>
              <span class="t-lbl">Folder &amp; archive import</span>
            </li>
            <li>
              <span class="t-num">RBAC</span>
              <span class="t-lbl">Owner to viewer</span>
            </li>
          </ul>
        </div>
        <div class="hero-visual ag-card lift-hover">
          <div class="visual-inner">
            <p class="visual-kicker">Live-style preview</p>
            <p class="visual-title">Bounding boxes &amp; class tags</p>
            <LandingShowcase />
          </div>
        </div>
      </div>

      <section class="flow">
        <h2 class="sec-title">From raw media to exportable datasets</h2>
        <div class="steps">
          <article v-for="(s, i) in steps" :key="s.t" class="step ag-card">
            <span class="step-ix">{{ i + 1 }}</span>
            <h3>{{ s.t }}</h3>
            <p>{{ s.d }}</p>
          </article>
        </div>
      </section>

      <section class="features">
        <h2 class="sec-title">Built for professional ML teams</h2>
        <div class="feat-grid">
          <article v-for="f in features" :key="f.t" class="feat ag-card lift-hover">
            <div class="feat-ic" v-html="f.icon" />
            <h3>{{ f.t }}</h3>
            <p>{{ f.d }}</p>
          </article>
        </div>
      </section>

      <footer class="footer">
        <p class="foot-line">
          <strong>ALPHA GUARD AI</strong>
          — workspace isolation, audit-friendly exports, HTTPS in production.
        </p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import AgHeader from '@/components/AgHeader.vue'
import LandingShowcase from '@/components/LandingShowcase.vue'

const steps = [
  { t: 'Ingest', d: 'Upload folders or ZIP with train / valid / test splits and metadata.' },
  { t: 'Label', d: 'Edit YOLO lines in place while seeing boxes on the image — no drift from disk.' },
  { t: 'Version', d: 'Freeze snapshots, track class names, and download reproducible packages.' },
  { t: 'Collaborate', d: 'Invite annotators and viewers; keep owners in control of visibility.' },
]

const icFolder =
  '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M10 4H4a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-8z"/></svg>'
const icLayers =
  '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2l10 7-10 5L2 9zm0 11.45L21.75 18 12 23l-9.75-5L12 13.45z"/></svg>'
const icShield =
  '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2l8 4v6c0 5-3.4 9.5-8 10-4.6-.5-8-5-8-10V6z"/></svg>'
const icGit =
  '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a10 10 0 0 1 3.3 19.45L12 17l-3.3 4.45A10 10 0 0 1 12 2m0 4a6 6 0 1 0 0 12 6 6 0 0 0 0-12m0 2a4 4 0 1 1 0 8 4 4 0 0 1 0-8"/></svg>'

const features = [
  {
    t: 'Imports that match your stack',
    d: 'Bring Roboflow-style layouts, `data.yaml`, and plain YOLO text without proprietary lock-in.',
    icon: icFolder,
  },
  {
    t: 'Draft vs snapshot',
    d: 'Keep editing in the active dataset while publishing immutable version exports for training.',
    icon: icLayers,
  },
  {
    t: 'Roles & public read',
    d: 'Mirror real teams: owner, admin, annotator, viewer — optional public discovery for demos.',
    icon: icShield,
  },
  {
    t: 'Transparent labels',
    d: 'What you see in the editor is what is stored — ideal for audits and regression checks.',
    icon: icGit,
  },
]
</script>

<style scoped>
.landing {
  min-height: 100vh;
  background:
    radial-gradient(1000px 700px at 8% -8%, rgba(77, 107, 254, 0.38), transparent 55%),
    radial-gradient(800px 500px at 100% 5%, rgba(94, 234, 212, 0.12), transparent 45%),
    radial-gradient(600px 400px at 50% 100%, rgba(77, 107, 254, 0.15), transparent 50%), var(--ag-bg);
}

.hero-grid {
  max-width: 1180px;
  margin: 0 auto;
  padding: 32px 22px 48px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr);
  gap: 36px;
  align-items: start;
}

@media (max-width: 1100px) {
  .hero-grid {
    grid-template-columns: 1fr;
  }
}

.pill {
  margin-bottom: 18px;
  display: inline-flex;
}

.title {
  margin: 0 0 18px;
  font-size: clamp(28px, 4.8vw, 48px);
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -0.02em;
}

.line {
  display: block;
}

.accent {
  color: var(--ag-accent);
}

.sub {
  max-width: 520px;
  color: var(--ag-muted);
  font-size: 16px;
  line-height: 1.7;
  margin-bottom: 26px;
}

.cta {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 28px;
}

.cta .ag-btn {
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 12px 22px;
  font-weight: 600;
}

.trust {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 20px 28px;
}

.trust li {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.t-num {
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.14em;
  color: var(--ag-accent);
}

.t-lbl {
  font-size: 12px;
  color: var(--ag-muted);
}

.hero-visual {
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--ag-border);
  background: linear-gradient(160deg, rgba(36, 36, 38, 0.98), rgba(27, 27, 28, 0.99));
}

.visual-inner {
  padding: 20px 18px 8px;
}

.visual-kicker {
  margin: 0 0 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--ag-muted);
}

.visual-title {
  margin: 0 0 8px;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.flow,
.features {
  max-width: 1180px;
  margin: 0 auto;
  padding: 24px 22px 56px;
}

.sec-title {
  margin: 0 0 22px;
  font-size: clamp(20px, 2.5vw, 28px);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.steps {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}

.step {
  position: relative;
  padding: 20px 18px 20px 52px;
  min-height: 120px;
}

.step-ix {
  position: absolute;
  left: 14px;
  top: 18px;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--ag-accent), #6366f1);
  color: #fff;
  font-size: 13px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
}

.step h3 {
  margin: 0 0 8px;
  font-size: 15px;
  font-weight: 700;
}

.step p {
  margin: 0;
  font-size: 13px;
  color: var(--ag-muted);
  line-height: 1.55;
}

.feat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.feat {
  padding: 22px 20px;
}

.feat-ic {
  width: 40px;
  height: 40px;
  margin-bottom: 14px;
  color: var(--ag-accent);
  opacity: 0.95;
}

.feat-ic :deep(svg) {
  width: 100%;
  height: 100%;
}

.feat h3 {
  margin: 0 0 10px;
  font-size: 16px;
  font-weight: 700;
}

.feat p {
  margin: 0;
  font-size: 14px;
  color: var(--ag-muted);
  line-height: 1.55;
}

.footer {
  max-width: 1180px;
  margin: 0 auto;
  padding: 0 22px 48px;
}

.foot-line {
  margin: 0;
  padding: 20px 0 0;
  border-top: 1px solid var(--ag-border);
  font-size: 13px;
  color: var(--ag-muted);
  line-height: 1.6;
}

.foot-line strong {
  color: var(--ag-text);
  letter-spacing: 0.12em;
  font-weight: 800;
}
</style>
