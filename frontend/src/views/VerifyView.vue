<template>
  <div class="page">
    <AgHeader />
    <div class="wrap ag-card">
      <h1>Email verification</h1>
      <p v-if="busy" class="muted">Please wait...</p>
      <p v-else-if="ok" class="ok">
        Email verified.
        <router-link class="ag-link" to="/login">Sign in</router-link>
      </p>
      <p v-else class="err">{{ err }}</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AgHeader from '@/components/AgHeader.vue'
import { useApi } from '@/composables/useApi'

const route = useRoute()
const api = useApi()
const busy = ref(true)
const ok = ref(false)
const err = ref('')

onMounted(async () => {
  const token = route.query.token
  if (!token) {
    busy.value = false
    err.value = 'No token in the link.'
    return
  }
  try {
    await api.request(`/auth/verify?token=${encodeURIComponent(token)}`)
    ok.value = true
  } catch (e) {
    err.value = e.message
  } finally {
    busy.value = false
  }
})
</script>

<style scoped>
.page {
  min-height: 100vh;
}

.wrap {
  max-width: 520px;
  margin: 60px auto;
  padding: 28px 26px;
}

.muted {
  color: var(--ag-muted);
}

.ok {
  color: #cfd8ff;
}

.err {
  color: #ff7b7b;
}
</style>
