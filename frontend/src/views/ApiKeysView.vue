<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '../stores/auth'

type ApiKey = { id: string; name: string; created_at: string; last_used_at?: string | null; revoked_at?: string | null }
type Domain = { id: string; hostname: string }

const auth = useAuthStore()
const keys = ref<ApiKey[]>([])
const domains = ref<Domain[]>([])
const name = ref('')
const createdRawKey = ref<string | null>(null)
const error = ref<string | null>(null)
const loading = ref(false)
const limit = ref(25)
const offset = ref(0)

const editingKeyId = ref<string | null>(null)
const allowedDomainIds = ref<string[]>([])

async function fetchKeys() {
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/api-keys?limit=${limit.value}&offset=${offset.value}`, { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    keys.value = (await res.json()) as ApiKey[]
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function fetchDomains() {
  const res = await fetch('/v1/domains?limit=200&offset=0', { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  const ds = (await res.json()) as any[]
  domains.value = ds.map((d) => ({ id: d.id, hostname: d.hostname }))
}

async function createKey() {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  createdRawKey.value = null
  try {
    const res = await fetch('/v1/api-keys', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({ name: name.value }),
    })
    if (!res.ok) throw new Error(await res.text())
    const j = (await res.json()) as { api_key: ApiKey; raw_key: string }
    createdRawKey.value = j.raw_key
    name.value = ''
    await fetchKeys()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function revokeKey(id: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/api-keys/${id}`, {
      method: 'DELETE',
      credentials: 'include',
      headers: { 'X-CSRF-Token': auth.csrf },
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchKeys()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function openDomainEditor(keyId: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    await fetchDomains()
    const res = await fetch(`/v1/api-keys/${keyId}/domains`, { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    allowedDomainIds.value = (await res.json()) as string[]
    editingKeyId.value = keyId
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function saveDomainEditor() {
  if (!auth.csrf || !editingKeyId.value) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/api-keys/${editingKeyId.value}/domains`, {
      method: 'PUT',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify(allowedDomainIds.value),
    })
    if (!res.ok) throw new Error(await res.text())
    editingKeyId.value = null
    allowedDomainIds.value = []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

function toggleDomain(id: string) {
  const set = new Set(allowedDomainIds.value)
  if (set.has(id)) set.delete(id)
  else set.add(id)
  allowedDomainIds.value = [...set]
}

function nextPage() {
  offset.value += limit.value
  void fetchKeys()
}
function prevPage() {
  offset.value = Math.max(0, offset.value - limit.value)
  void fetchKeys()
}

onMounted(fetchKeys)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="sw-title">API keys</h1>
      <p class="sw-subtitle">Create API keys for integrations and the Chrome extension.</p>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="flex gap-3">
        <input v-model="name" class="sw-input" placeholder="Key name (e.g. chrome)" />
        <button class="sw-btn sw-btn-primary" @click="createKey">
          Create
        </button>
      </div>
      <div v-if="createdRawKey" class="mt-3 rounded-md border border-amber-900/50 bg-amber-950/30 p-3 text-sm text-amber-200">
        <div class="font-medium">Copy this key now — it will only be shown once:</div>
        <code class="mt-2 block select-all rounded border border-slate-800 bg-slate-950 px-2 py-1 text-slate-100">{{ createdRawKey }}</code>
      </div>
      <div v-if="error" class="sw-error mt-3">{{ error }}</div>
      </div>
    </div>

    <div class="sw-card">
      <div class="flex items-center justify-between border-b border-slate-800 px-4 py-3 text-sm font-medium">
        <div class="text-slate-100">Your keys</div>
        <div class="flex items-center gap-2">
          <button class="sw-btn px-2 py-1 text-xs" :disabled="offset===0" @click="prevPage">
            Prev
          </button>
          <button class="sw-btn px-2 py-1 text-xs" @click="nextPage">Next</button>
        </div>
      </div>
      <div v-if="loading" class="p-4 text-sm text-slate-400">Loading…</div>
      <div v-else class="divide-y">
        <div v-for="k in keys" :key="k.id" class="flex items-center justify-between gap-4 p-4">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">{{ k.name }}</div>
            <div class="text-xs text-slate-400">Created: {{ k.created_at }}</div>
          </div>
          <div class="flex items-center gap-2">
            <button class="sw-btn px-2 py-1 text-xs" @click="openDomainEditor(k.id)">
              Allowed domains
            </button>
            <button class="sw-btn sw-btn-danger px-2 py-1 text-xs" @click="revokeKey(k.id)">
              Revoke
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="editingKeyId" class="sw-card">
      <div class="sw-card-body">
      <div class="text-sm font-medium text-slate-100">Allowed domains</div>
      <p class="mt-1 text-sm text-slate-400">Select domains this API key may use. Empty selection means unrestricted.</p>
      <div class="mt-3 grid gap-2 md:grid-cols-2">
        <label v-for="d in domains" :key="d.id" class="flex items-center gap-2 rounded-md border border-slate-800 bg-slate-950 px-3 py-2 text-sm">
          <input type="checkbox" :checked="allowedDomainIds.includes(d.id)" @change="toggleDomain(d.id)" />
          <span class="truncate">{{ d.hostname }}</span>
        </label>
      </div>
      <div class="mt-3 flex items-center justify-end gap-2">
        <button class="sw-btn" @click="editingKeyId=null">Cancel</button>
        <button class="sw-btn sw-btn-primary" @click="saveDomainEditor">
          Save
        </button>
      </div>
      </div>
    </div>
  </div>
</template>

