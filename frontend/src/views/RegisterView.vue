<template>
  <div class="page">
    <AgHeader />
    <div class="wrap">
      <div class="ag-card form-card">
        <h1>Sign Up</h1>
        <p class="hint">Minimum 8 characters in password. A verification link will be sent to email.</p>
        <form class="stack" @submit.prevent="submit">
          <label>First name</label>
          <input v-model.trim="firstName" class="ag-input" type="text" required autocomplete="given-name" />
          <label>Last name</label>
          <input v-model.trim="lastName" class="ag-input" type="text" required autocomplete="family-name" />
          <label>Email</label>
          <input v-model.trim="email" class="ag-input" type="email" required autocomplete="email" />
          <label>Password</label>
          <input v-model="password" class="ag-input" type="password" required minlength="8" autocomplete="new-password" />
          <div v-if="err" class="err">{{ err }}</div>
          <div v-if="verifyUrl" class="verify-box">
            <span class="verify-label">Verification link (if SMTP is not configured):</span>
            <a class="ag-link break" :href="verifyUrl" target="_blank" rel="noopener">{{ verifyUrl }}</a>
          </div>
          <div v-if="hint" class="ok">{{ hint }}</div>
          <button class="ag-btn ag-btn-primary" type="submit" :disabled="loading">Sign up</button>
        </form>
        <p class="foot">
          Already have an account?
          <router-link class="ag-link" to="/login">Sign in</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import AgHeader from '@/components/AgHeader.vue'
import { useApi } from '@/composables/useApi'

const api = useApi()
const firstName = ref('')
const lastName = ref('')
const email = ref('')
const password = ref('')
const err = ref('')
const hint = ref('')
const verifyUrl = ref('')
const loading = ref(false)

async function submit() {
  err.value = ''
  hint.value = ''
  verifyUrl.value = ''
  loading.value = true
  try {
    const data = await api.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({
        first_name: firstName.value,
        last_name: lastName.value,
        email: email.value,
        password: password.value,
      }),
    })
    if (data.verification_url) {
      verifyUrl.value = data.verification_url
      hint.value = 'SMTP is not configured: open the link above to verify email.'
    } else if (data.message) {
      hint.value = 'Verification email was sent to the specified email address.'
    }
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

.ok {
  color: #8be38b;
  font-size: 14px;
}

.foot {
  margin-top: 18px;
  font-size: 14px;
  color: var(--ag-muted);
}

.verify-box {
  margin-top: 10px;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid rgba(77, 107, 254, 0.45);
  background: rgba(77, 107, 254, 0.12);
}

.verify-label {
  display: block;
  font-size: 12px;
  color: var(--ag-muted);
  margin-bottom: 6px;
}

.break {
  word-break: break-all;
}
</style>
