<script setup lang="ts">
import { useAuthStore } from '../stores/auth'
import { onMounted, ref } from 'vue'

const auth = useAuthStore()

const currentPassword = ref('')
const newPassword = ref('')
const msg = ref<string | null>(null)
const loading = ref(false)

const redirectDelaySeconds = ref(0)
const keepExpiredLinks = ref(false)
const timezone = ref('UTC')

async function loadSettings() {
  msg.value = null
  try {
    const res = await fetch('/v1/me/settings', { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    const j = (await res.json()) as {
      redirect_delay_seconds: number
      keep_expired_links: boolean
      timezone: string
    }
    redirectDelaySeconds.value = j.redirect_delay_seconds
    keepExpiredLinks.value = j.keep_expired_links
    timezone.value = j.timezone
  } catch (e) {
    msg.value = e instanceof Error ? e.message : 'Failed'
  }
}

async function saveSettings() {
  msg.value = null
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  try {
    const res = await fetch('/v1/me/settings', {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({
        redirect_delay_seconds: redirectDelaySeconds.value,
        keep_expired_links: keepExpiredLinks.value,
        timezone: timezone.value,
      }),
    })
    if (!res.ok) throw new Error(await res.text())
    msg.value = 'Settings saved.'
  } catch (e) {
    msg.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function changePassword() {
  msg.value = null
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  try {
    const res = await fetch('/v1/me/password', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({ current_password: currentPassword.value, new_password: newPassword.value }),
    })
    if (!res.ok) throw new Error(await res.text())
    currentPassword.value = ''
    newPassword.value = ''
    msg.value = 'Password updated.'
  } catch (e) {
    msg.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-4">
    <div>
      <h1 class="sw-title">Settings</h1>
      <p class="sw-subtitle">Account preferences and security.</p>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="text-sm text-slate-400">Signed in as</div>
      <div class="mt-1 font-medium">{{ auth.user?.email }}</div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="text-sm font-medium text-slate-100">Preferences</div>
      <div class="mt-3 grid gap-3 md:grid-cols-3">
        <div>
          <label class="sw-label">Redirect delay (seconds)</label>
          <input v-model.number="redirectDelaySeconds" class="sw-input mt-1" type="number" min="0" max="30" />
        </div>
        <div class="flex items-center gap-2 pt-6">
          <input id="keepExpired" v-model="keepExpiredLinks" type="checkbox" />
          <label for="keepExpired" class="text-sm">Keep expired links</label>
        </div>
        <div>
          <label class="sw-label">Timezone</label>
          <input v-model="timezone" class="sw-input mt-1" placeholder="UTC" />
        </div>
      </div>
      <div class="mt-3 flex items-center justify-between">
        <div v-if="msg" class="text-sm text-slate-300">{{ msg }}</div>
        <button
          class="sw-btn"
          :disabled="loading"
          @click="saveSettings"
        >
          Save preferences
        </button>
      </div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="text-sm font-medium text-slate-100">Change password</div>
      <div class="mt-3 grid gap-3 md:grid-cols-2">
        <div>
          <label class="sw-label">Current password</label>
          <input v-model="currentPassword" class="sw-input mt-1" type="password" />
        </div>
        <div>
          <label class="sw-label">New password</label>
          <input v-model="newPassword" class="sw-input mt-1" type="password" minlength="8" />
        </div>
      </div>
      <div class="mt-3 flex items-center justify-between">
        <div v-if="msg" class="text-sm text-slate-300">{{ msg }}</div>
        <button
          class="sw-btn sw-btn-primary"
          :disabled="loading"
          @click="changePassword"
        >
          {{ loading ? 'Saving…' : 'Update password' }}
        </button>
      </div>
      </div>
    </div>
  </div>
</template>

