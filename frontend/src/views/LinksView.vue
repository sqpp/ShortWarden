<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

type Link = {
  id: string
  alias: string
  target_url: string
  short_url?: string
  created_at: string
}
type Domain = { id: string; hostname: string; status: 'pending' | 'verified' }

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const links = ref<Link[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const targetUrl = ref('')
const alias = ref('')
const domainId = ref<string>('')
const tagFilter = ref<string>('')
const limit = ref(25)
const offset = ref(0)
const domains = ref<Domain[]>([])

function copyLink(l: Link) {
  const text = l.short_url ?? `${window.location.origin}/r/${l.alias}`
  void navigator.clipboard.writeText(text)
}

async function fetchLinks() {
  loading.value = true
  error.value = null
  try {
    const tagQ = tagFilter.value ? `&tag=${encodeURIComponent(tagFilter.value)}` : ''
    const res = await fetch(`/v1/links?limit=${limit.value}&offset=${offset.value}${tagQ}`, { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    links.value = (await res.json()) as Link[]
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function fetchDomains() {
  try {
    const res = await fetch('/v1/domains?limit=200&offset=0', { credentials: 'include' })
    if (!res.ok) return
    const ds = (await res.json()) as Domain[]
    domains.value = ds.filter((d) => d.status === 'verified')
    if (!domainId.value && domains.value.length) {
      // leave empty to use primary by default; user can pick explicitly
    }
  } catch {
    // ignore
  }
}

async function createLink() {
  if (!auth.csrf) {
    await auth.bootstrap()
  }
  const csrf = auth.csrf
  if (!csrf) return

  loading.value = true
  error.value = null
  try {
    const res = await fetch('/v1/links', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrf },
      body: JSON.stringify({
        target_url: targetUrl.value,
        alias: alias.value ? alias.value : undefined,
        domain_id: domainId.value ? domainId.value : undefined,
      }),
    })
    if (!res.ok) throw new Error(await res.text())
    targetUrl.value = ''
    alias.value = ''
    domainId.value = ''
    await fetchLinks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function deleteLink(id: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/links/${id}`, {
      method: 'DELETE',
      credentials: 'include',
      headers: { 'X-CSRF-Token': auth.csrf },
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchLinks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void fetchDomains()
  void fetchLinks()
})

watch(
  () => route.query.tag,
  (v) => {
    tagFilter.value = typeof v === 'string' ? v : ''
    offset.value = 0
    void fetchLinks()
  },
  { immediate: true },
)
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h1 class="sw-title">Links</h1>
        <p class="sw-subtitle">Create and manage your short links.</p>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="grid gap-3 md:grid-cols-4">
        <div class="md:col-span-2">
          <label class="sw-label">Target URL</label>
          <input v-model="targetUrl" class="sw-input mt-1" placeholder="https://…" />
        </div>
        <div>
          <label class="sw-label">Custom alias (optional)</label>
          <input v-model="alias" class="sw-input mt-1" placeholder="my-alias" />
        </div>
        <div>
          <label class="sw-label">Domain</label>
          <select v-model="domainId" class="sw-select mt-1">
            <option value="">Use primary domain</option>
            <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.hostname }}</option>
          </select>
        </div>
      </div>
      <div class="mt-3">
        <label class="sw-label">Tag filter</label>
        <input
          v-model="tagFilter"
          class="sw-input mt-1"
          placeholder="tag"
          @change="fetchLinks"
        />
      </div>
      <div class="mt-3 flex items-center justify-between">
        <div v-if="error" class="text-sm text-red-200">{{ error }}</div>
        <button
          class="sw-btn sw-btn-primary"
          :disabled="loading"
          @click="createLink"
        >
          Create
        </button>
      </div>
      </div>
    </div>

    <div class="sw-card">
      <div class="flex items-center justify-between border-b border-slate-800 px-4 py-3 text-sm font-medium">
        <div class="text-slate-100">Your links</div>
        <div class="flex items-center gap-2">
          <button class="sw-btn px-2 py-1 text-xs" :disabled="offset===0" @click="offset=Math.max(0, offset-limit); fetchLinks()">
            Prev
          </button>
          <button class="sw-btn px-2 py-1 text-xs" @click="offset=offset+limit; fetchLinks()">
            Next
          </button>
        </div>
      </div>
      <div v-if="loading" class="p-4 text-sm text-slate-400">Loading…</div>
      <div v-else class="divide-y">
        <div v-for="l in links" :key="l.id" class="flex items-center justify-between gap-4 p-4">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">
              <a class="hover:underline" :href="l.short_url ?? ('/r/' + l.alias)" target="_blank" rel="noreferrer">
                {{ l.short_url ?? ('/r/' + l.alias) }}
              </a>
            </div>
            <div class="truncate text-xs text-slate-400">{{ l.target_url }}</div>
            <button class="mt-2 text-xs font-medium text-slate-200 hover:underline" @click="router.push(`/app/links/${l.id}`)">
              View clicks & analytics
            </button>
          </div>
          <div class="flex items-center gap-2">
            <button class="sw-btn px-2 py-1 text-xs" @click="copyLink(l)">Copy</button>
            <button class="sw-btn sw-btn-danger px-2 py-1 text-xs" @click="deleteLink(l.id)">Delete</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

