<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const email = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit() {
  error.value = null
  loading.value = true
  try {
    await auth.login(email.value, password.value)
    const next = typeof route.query.next === 'string' ? route.query.next : '/app/links'
    await router.replace(next)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Login failed'
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
        <div class="mt-1 text-xs text-slate-400">Sign in to manage your short links.</div>
      </div>

      <div class="sw-card">
        <div class="sw-card-body">
          <h1 class="text-lg font-semibold">Login</h1>

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
            autocomplete="current-password"
            required
          />
        </div>
        <div v-if="error" class="sw-error">
          {{ error }}
        </div>
        <button
          class="sw-btn sw-btn-primary w-full"
          :disabled="loading"
        >
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>

      <p class="mt-4 text-sm text-slate-400">
        No account?
        <RouterLink class="font-medium text-slate-100 hover:underline" to="/register">Register</RouterLink>
      </p>
        </div>
      </div>
    </div>
  </div>
</template>

