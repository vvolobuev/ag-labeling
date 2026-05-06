<template>
  <div class="page">
    <AgHeader />
    <div class="wrap">
      <div class="ag-card form-card">
        <h1>Sign in</h1>
        <p class="hint">After signing up, verify your email via the link in the message.</p>
        <form class="stack" @submit.prevent="submit">
          <label>Email</label>
          <input v-model.trim="email" class="ag-input" type="email" required autocomplete="email" />
          <label>Password</label>
          <input v-model="password" class="ag-input" type="password" required autocomplete="current-password" />
          <div v-if="err" class="err">{{ err }}</div>
          <button class="ag-btn ag-btn-primary" type="submit" :disabled="loading">Sign in</button>
        </form>
        <p class="foot">
          No account yet?
          <router-link class="ag-link" to="/register">Sign up</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import AgHeader from '@/components/AgHeader.vue'
import { useApi, setToken } from '@/composables/useApi'

const api = useApi()
const router = useRouter()
const route = useRoute()
const email = ref('')
const password = ref('')
const err = ref('')
const loading = ref(false)

async function submit() {
  err.value = ''
  loading.value = true
  try {
    const data = await api.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email: email.value, password: password.value }),
    })
    setToken(data.token)
    router.push(route.query.redirect || '/workspaces')
  } catch (e) {
    err.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page {
  min-height: 100vh;
}

.wrap {
  max-width: 440px;
  margin: 0 auto;
  padding: 40px 20px 80px;
}

.form-card h1 {
  margin: 0 0 8px;
  font-size: 26px;
}

.hint {
  color: var(--ag-muted);
  font-size: 14px;
  margin-bottom: 20px;
  line-height: 1.5;
}

.stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

label {
  font-size: 13px;
  color: var(--ag-muted);
  margin-top: 6px;
}

.err {
  color: #ff7b7b;
  font-size: 14px;
}

.foot {
  margin-top: 18px;
  font-size: 14px;
  color: var(--ag-muted);
}
</style>
