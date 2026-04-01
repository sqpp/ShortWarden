<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

const email = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit() {
  error.value = null
  loading.value = true
  try {
    await auth.register(email.value, password.value)
    await router.replace('/app/links')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-slate-950 px-4 py-12 text-slate-100">
    <div class="mx-auto max-w-md">
      <div class="mb-6 text-center">
        <div class="text-sm font-semibold tracking-wide">ShortWarden</div>
        <div class="mt-1 text-xs text-slate-400">Create an account to start shortening URLs.</div>
      </div>

      <div class="sw-card">
        <div class="sw-card-body">
          <h1 class="text-lg font-semibold">Register</h1>

      <form class="mt-6 space-y-4" @submit.prevent="onSubmit">
        <div>
          <label class="sw-label">Email</label>
          <input
            v-model="email"
            class="sw-input mt-1"
            type="email"
            autocomplete="email"
            required
          />
        </div>
        <div>
          <label class="sw-label">Password</label>
          <input
            v-model="password"
            class="sw-input mt-1"
            type="password"
            autocomplete="new-password"
            minlength="8"
            required
          />
          <p class="mt-1 text-xs text-slate-500">Minimum 8 characters.</p>
        </div>
        <div v-if="error" class="sw-error">
          {{ error }}
        </div>
        <button
          class="sw-btn sw-btn-primary w-full"
          :disabled="loading"
        >
          {{ loading ? 'Creating…' : 'Create account' }}
        </button>
      </form>

      <p class="mt-4 text-sm text-slate-400">
        Already have an account?
        <RouterLink class="font-medium text-slate-100 hover:underline" to="/login">Login</RouterLink>
      </p>
        </div>
      </div>
    </div>
  </div>
</template>

