<template>
  <section class="page">
    <h1>Settings</h1>
    <div class="ag-card box">
      <h2>User Profile</h2>
      <div class="profile-head">
        <img v-if="avatarUrl" :src="avatarUrl + '?t=' + cacheBust" class="avatar" alt="" />
        <div v-else class="avatar ph">{{ initials }}</div>
        <div class="upload-wrap">
          <input ref="avatarRef" class="hid" type="file" accept="image/*" @change="onAvatarPicked" />
          <button class="ag-btn ag-btn-ghost" type="button" :disabled="savingAvatar" @click="pickAvatar">
            {{ savingAvatar ? 'Uploading...' : 'Upload Photo' }}
          </button>
        </div>
      </div>

      <div class="grid">
        <label>
          <span class="muted">First Name</span>
          <input v-model.trim="firstName" class="ag-input" />
        </label>
        <label>
          <span class="muted">Last Name</span>
          <input v-model.trim="lastName" class="ag-input" />
        </label>
      </div>

      <div class="actions">
        <button class="ag-btn ag-btn-primary" type="button" :disabled="savingProfile" @click="saveProfile">
          {{ savingProfile ? 'Saving...' : 'Save Changes' }}
        </button>
      </div>

      <p v-if="ok" class="ok">{{ ok }}</p>
      <p v-if="err" class="err">{{ err }}</p>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useApi } from '@/composables/useApi'

const api = useApi()
const firstName = ref('')
const lastName = ref('')
const avatarUrl = ref('')
const cacheBust = ref(Date.now())
const ok = ref('')
const err = ref('')
const savingProfile = ref(false)
const savingAvatar = ref(false)
const avatarRef = ref(null)

const initials = computed(() => {
  const f = (firstName.value || '').trim()
  const l = (lastName.value || '').trim()
  return `${f[0] || ''}${l[0] || ''}`.toUpperCase() || 'U'
})

async function loadMe() {
  const me = await api.request('/me')
  firstName.value = me.first_name || ''
  lastName.value = me.last_name || ''
  avatarUrl.value = me.avatar_url || ''
}

function pickAvatar() {
  avatarRef.value?.click()
}

async function onAvatarPicked() {
  err.value = ''
  ok.value = ''
  const file = avatarRef.value?.files?.[0]
  if (!file) return
  savingAvatar.value = true
  try {
    const fd = new FormData()
    fd.append('avatar', file)
    const res = await api.request('/me/avatar', { method: 'POST', body: fd })
    avatarUrl.value = res.avatar_url || '/api/me/avatar'
    cacheBust.value = Date.now()
    ok.value = 'Photo updated.'
  } catch (e) {
    err.value = e.message || 'Avatar upload failed.'
  } finally {
    savingAvatar.value = false
    if (avatarRef.value) avatarRef.value.value = ''
  }
}

async function saveProfile() {
  err.value = ''
  ok.value = ''
  savingProfile.value = true
  try {
    await api.request('/me', {
      method: 'PATCH',
      body: JSON.stringify({ first_name: firstName.value, last_name: lastName.value }),
    })
    ok.value = 'Profile saved.'
  } catch (e) {
    err.value = e.message || 'Profile save failed.'
  } finally {
    savingProfile.value = false
  }
}

onMounted(() => {
  loadMe().catch(() => {})
})
</script>

<style scoped>
.page { padding: 20px 22px; }
h1 { margin: 0 0 12px; font-size: 28px; }
.muted { color: var(--ag-muted); }
.box h2 { margin: 0 0 12px; font-size: 18px; }
.profile-head { display: flex; align-items: center; gap: 12px; margin-bottom: 14px; }
.avatar { width: 64px; height: 64px; border-radius: 999px; object-fit: cover; border: 1px solid var(--ag-border); }
.avatar.ph { display: inline-flex; align-items: center; justify-content: center; background: rgba(77, 107, 254, 0.2); color: #d9e3ff; font-weight: 700; }
.hid { display: none; }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 280px)); gap: 12px; }
label { display: flex; flex-direction: column; gap: 8px; }
.actions { margin-top: 14px; }
.ok { color: #8be38b; }
.err { color: #ff7b7b; }
@media (max-width: 760px) { .grid { grid-template-columns: 1fr; } }
</style>
