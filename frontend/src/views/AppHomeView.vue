<template>
  <section class="page">
    <h1>Home</h1>
    <p class="muted">Workspace home page in unified platform style.</p>
    <div class="grid">
      <article class="ag-card tile">
        <h3>Projects</h3>
        <p class="muted">Open datasets, apply filters, and navigate to versions.</p>
      </article>
      <article class="ag-card tile">
        <h3>Explore</h3>
        <p class="muted">Search public datasets across all workspaces.</p>
      </article>
      <article class="ag-card tile">
        <h3>Settings</h3>
        <p class="muted">Team and member settings.</p>
      </article>
    </div>
  </section>
</template>

<script setup></script>

<style scoped>
.page { padding: 20px 22px; }
.muted { color: var(--ag-muted); }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 12px; margin-top: 14px; }
.tile h3 { margin: 0 0 8px; }
.tile p { margin: 0; }
</style>
