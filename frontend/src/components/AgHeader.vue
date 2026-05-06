<template>
  <header class="ag-header lift-hover">
    <div class="inner">
      <router-link to="/" class="brand">ALPHA GUARD <span class="ai">AI</span></router-link>
      <nav class="nav">
        <template v-if="hasToken">
          <router-link class="ag-link nav-item" to="/workspaces">Workspace</router-link>
          <button type="button" class="ag-btn ag-btn-ghost nav-item" @click="logout">Logout</button>
        </template>
        <template v-else>
          <router-link class="ag-link nav-item" to="/login">Sign in</router-link>
          <router-link class="ag-btn ag-btn-primary nav-cta" to="/register">Sign up</router-link>
        </template>
      </nav>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { getToken, setToken } from '@/composables/useApi'

const router = useRouter()
const hasToken = computed(() => !!getToken())

function logout() {
  setToken('')
  router.push('/')
}
</script>

<style scoped>
.ag-header {
  position: sticky;
  top: 0;
  z-index: 100;
  backdrop-filter: blur(18px);
  background: rgba(27, 27, 28, 0.92);
  border-bottom: 1px solid var(--ag-border);
}

.inner {
  width: 100%;
  margin: 0;
  padding: 14px 14px 14px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.brand {
  font-weight: 800;
  letter-spacing: 0.14em;
  font-size: 14px;
  color: #fff;
  text-decoration: none;
}

.ai {
  color: var(--ag-accent);
}

.nav {
  display: flex;
  align-items: center;
  gap: 16px;
}

.nav-item {
  font-size: 14px;
}

.nav-cta {
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  padding: 10px 18px;
}

@media (max-width: 560px) {
  .inner {
    padding: 12px 14px;
    flex-wrap: wrap;
  }

  .brand {
    letter-spacing: 0.08em;
  }
}
</style>
