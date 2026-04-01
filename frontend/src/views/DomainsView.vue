<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '../stores/auth'

type Domain = {
  id: string
  hostname: string
  is_primary: boolean
  status: 'pending' | 'verified'
  dns_token: string
  default_tags?: string[]
}

const auth = useAuthStore()
const domains = ref<Domain[]>([])
const hostname = ref('')
const error = ref<string | null>(null)
const loading = ref(false)
const limit = ref(25)
const offset = ref(0)

async function fetchDomains() {
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/domains?limit=${limit.value}&offset=${offset.value}`, { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    domains.value = (await res.json()) as Domain[]
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function addDomain() {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch('/v1/domains', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({ hostname: hostname.value }),
    })
    if (!res.ok) throw new Error(await res.text())
    hostname.value = ''
    await fetchDomains()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function verifyDomain(id: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/domains/${id}/verify`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-CSRF-Token': auth.csrf },
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchDomains()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function setPrimary(id: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/domains/${id}/primary`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-CSRF-Token': auth.csrf },
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchDomains()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function removeDomain(id: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/domains/${id}`, {
      method: 'DELETE',
      credentials: 'include',
      headers: { 'X-CSRF-Token': auth.csrf },
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchDomains()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function saveDefaultTags(d: Domain, raw: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const tags = raw
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    const res = await fetch(`/v1/domains/${d.id}/default-tags`, {
      method: 'PUT',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify(tags),
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchDomains()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

function nextPage() {
  offset.value += limit.value
  void fetchDomains()
}
function prevPage() {
  offset.value = Math.max(0, offset.value - limit.value)
  void fetchDomains()
}

onMounted(fetchDomains)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="sw-title">Domains</h1>
      <p class="sw-subtitle">Add and verify domains for shortening.</p>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="flex gap-3">
        <input v-model="hostname" class="sw-input" placeholder="example.com" />
        <button class="sw-btn sw-btn-primary" @click="addDomain">
          Add
        </button>
      </div>
      <p class="mt-2 text-xs text-slate-400">
        Verification TXT record:
        <code class="rounded border border-slate-800 bg-slate-950 px-1 text-slate-200">_shortwarden-challenge.&lt;domain&gt;</code>
      </p>
      <div v-if="error" class="sw-error mt-3">{{ error }}</div>
      </div>
    </div>

    <div class="sw-card">
      <div class="flex items-center justify-between border-b border-slate-800 px-4 py-3 text-sm font-medium">
        <div class="text-slate-100">Your domains</div>
        <div class="flex items-center gap-2">
          <button class="sw-btn px-2 py-1 text-xs" :disabled="offset===0" @click="prevPage">
            Prev
          </button>
          <button class="sw-btn px-2 py-1 text-xs" @click="nextPage">Next</button>
        </div>
      </div>
      <div v-if="loading" class="p-4 text-sm text-slate-400">Loading…</div>
      <div v-else class="divide-y">
        <div v-for="d in domains" :key="d.id" class="flex items-center justify-between gap-4 p-4">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">
              {{ d.hostname }}
              <span v-if="d.is_primary" class="ml-2 rounded border border-emerald-900/40 bg-emerald-950/30 px-2 py-0.5 text-xs text-emerald-200"
                >Primary</span
              >
            </div>
            <div class="text-xs text-slate-400">
              Status: {{ d.status }} • Token:
              <code class="rounded border border-slate-800 bg-slate-950 px-1 text-slate-200">{{ d.dns_token }}</code>
            </div>
            <div class="mt-2">
              <label class="sw-label">Default tags (comma-separated)</label>
              <div class="mt-1 flex gap-2">
                <input
                  class="sw-input px-2 py-1 text-xs"
                  :value="(d.default_tags ?? []).join(', ')"
                  @change="saveDefaultTags(d, ($event.target as HTMLInputElement).value)"
                />
              </div>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <button
              v-if="d.status === 'pending'"
              class="sw-btn px-2 py-1 text-xs"
              @click="verifyDomain(d.id)"
            >
              Verify
            </button>
            <button
              v-if="d.status === 'verified' && !d.is_primary"
              class="sw-btn px-2 py-1 text-xs"
              @click="setPrimary(d.id)"
            >
              Make primary
            </button>
            <button class="sw-btn sw-btn-danger px-2 py-1 text-xs" @click="removeDomain(d.id)">
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

