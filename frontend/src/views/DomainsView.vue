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
    <div class="sw-page-header">
      <div>
        <h1 class="sw-title">Domains</h1>
        <p class="sw-subtitle">Add domains, verify ownership, and set a primary domain.</p>
      </div>
      <div class="flex items-center gap-2">
        <button class="sw-btn" @click="fetchDomains">Refresh</button>
      </div>
    </div>

    <div class="sw-tile">
      <div class="sw-tile-body">
        <div class="sw-tile-top">
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-slate-100">Add a domain</div>
            <div class="sw-tile-label">Verify via DNS TXT, then set it as primary.</div>
          </div>
          <div class="sw-tile-icon">
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z" />
              <path d="M3 12h18" />
              <path d="M12 3c2.5 2.5 4 5.9 4 9s-1.5 6.5-4 9c-2.5-2.5-4-5.9-4-9s1.5-6.5 4-9Z" />
            </svg>
          </div>
        </div>

        <div class="mt-4 flex gap-3">
          <input v-model="hostname" class="sw-input" placeholder="example.com" />
          <button class="sw-btn sw-btn-primary" @click="addDomain">Add</button>
        </div>
        <p class="mt-2 text-xs text-slate-400">
          TXT record:
          <code class="rounded border border-white/10 bg-[#1c1f2a] px-1 text-slate-200">_shortwarden-challenge.&lt;domain&gt;</code>
        </p>
        <div v-if="error" class="sw-error mt-3">{{ error }}</div>
      </div>
    </div>

    <div class="sw-card">
      <div class="flex items-center justify-between border-b border-white/5 px-5 py-4 text-sm font-semibold">
        <div class="text-slate-100">Your domains</div>
        <div class="flex items-center gap-2">
          <button class="sw-btn px-2 py-1 text-xs" :disabled="offset===0" @click="prevPage">
            Prev
          </button>
          <button class="sw-btn px-2 py-1 text-xs" @click="nextPage">Next</button>
        </div>
      </div>
      <div v-if="loading" class="p-4 text-sm text-slate-400">Loading…</div>
      <div v-else class="overflow-auto">
        <table class="sw-table">
          <thead class="sw-thead">
            <tr>
              <th class="py-3 pl-5">Domain</th>
              <th class="py-3">Status</th>
              <th class="py-3">Token</th>
              <th class="py-3">Default tags</th>
              <th class="py-3 pr-5 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in domains" :key="d.id" class="sw-row">
              <td class="py-3 pl-5">
                <div class="flex flex-wrap items-center gap-2">
                  <div class="font-medium text-slate-100">{{ d.hostname }}</div>
                  <span
                    v-if="d.is_primary"
                    class="rounded-full border border-lime-400/25 bg-lime-400/10 px-2 py-0.5 text-xs font-medium text-lime-200"
                    >Primary</span
                  >
                </div>
              </td>
              <td class="py-3 text-slate-300">
                <span v-if="d.status === 'verified'" class="text-lime-200">Verified</span>
                <span v-else class="text-slate-400">Pending</span>
              </td>
              <td class="py-3">
                <code class="rounded border border-white/10 bg-[#1c1f2a] px-2 py-1 text-xs text-slate-200">{{ d.dns_token }}</code>
              </td>
              <td class="py-3">
                <input
                  class="sw-input px-2 py-1 text-xs"
                  :value="(d.default_tags ?? []).join(', ')"
                  @change="saveDefaultTags(d, ($event.target as HTMLInputElement).value)"
                />
              </td>
              <td class="py-3 pr-5">
                <div class="flex justify-end gap-2">
                  <button v-if="d.status === 'pending'" class="sw-btn px-2 py-1 text-xs" @click="verifyDomain(d.id)">Verify</button>
                  <button
                    v-if="d.status === 'verified' && !d.is_primary"
                    class="sw-btn px-2 py-1 text-xs"
                    @click="setPrimary(d.id)"
                  >
                    Make primary
                  </button>
                  <button class="sw-btn sw-btn-danger px-2 py-1 text-xs" @click="removeDomain(d.id)">Delete</button>
                </div>
              </td>
            </tr>
            <tr v-if="!domains.length">
              <td class="py-6 pl-5 text-sm text-slate-400" colspan="5">No domains yet. Add one above.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

