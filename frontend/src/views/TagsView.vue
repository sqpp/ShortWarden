<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'

type Tag = { name: string; link_count: number; curated?: boolean }
type Domain = { hostname: string; default_tags?: string[] }
type TagView = { name: string; link_count: number; curated?: boolean; domains: string[] }

const auth = useAuthStore()
const router = useRouter()

const tags = ref<TagView[]>([])
const name = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

async function fetchTags() {
  loading.value = true
  error.value = null
  try {
    const [tagsRes, domainsRes] = await Promise.all([
      fetch('/v1/tags?limit=200&offset=0', { credentials: 'include' }),
      fetch('/v1/domains?limit=200&offset=0', { credentials: 'include' }),
    ])
    if (!tagsRes.ok) throw new Error(await tagsRes.text())
    if (!domainsRes.ok) throw new Error(await domainsRes.text())
    const tagRows = (await tagsRes.json()) as Tag[]
    const domainRows = (await domainsRes.json()) as Domain[]

    const domainUsage = new Map<string, string[]>()
    for (const d of domainRows) {
      for (const raw of d.default_tags ?? []) {
        const t = raw.trim()
        if (!t) continue
        const arr = domainUsage.get(t) ?? []
        arr.push(d.hostname)
        domainUsage.set(t, arr)
      }
    }

    const byName = new Map<string, TagView>()
    for (const t of tagRows) {
      byName.set(t.name, {
        name: t.name,
        link_count: t.link_count,
        curated: t.curated,
        domains: domainUsage.get(t.name) ?? [],
      })
    }
    for (const [name, domainsUsing] of domainUsage.entries()) {
      if (byName.has(name)) continue
      byName.set(name, { name, link_count: 0, curated: false, domains: domainsUsing })
    }
    tags.value = [...byName.values()].sort((a, b) => a.name.localeCompare(b.name))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function createTag() {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch('/v1/tags', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({ name: name.value }),
    })
    if (!res.ok) throw new Error(await res.text())
    name.value = ''
    await fetchTags()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function deleteTag(tagName: string) {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`/v1/tags/${encodeURIComponent(tagName)}`, {
      method: 'DELETE',
      credentials: 'include',
      headers: { 'X-CSRF-Token': auth.csrf },
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchTags()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

function openTag(tagName: string) {
  router.push({ path: '/app/links', query: { tag: tagName } })
}

onMounted(fetchTags)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="sw-title">Tags</h1>
      <p class="sw-subtitle">Browse tags and jump to the links that use them.</p>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="flex gap-3">
        <input v-model="name" class="sw-input" placeholder="New tag name" />
        <button class="sw-btn sw-btn-primary" @click="createTag">
          Add
        </button>
      </div>
      <div v-if="error" class="sw-error mt-3">{{ error }}</div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-header">Your tags</div>
      <div v-if="loading" class="p-4 text-sm text-slate-400">Loading…</div>
      <div v-else class="divide-y">
        <div v-for="t in tags" :key="t.name" class="flex items-center justify-between gap-4 p-4">
          <button class="min-w-0 text-left" @click="openTag(t.name)">
            <div class="truncate text-sm font-medium">{{ t.name }}</div>
            <div class="text-xs text-slate-400">{{ t.link_count }} links</div>
            <div v-if="t.domains.length" class="truncate text-xs text-slate-500">Domains: {{ t.domains.join(', ') }}</div>
          </button>
          <button class="sw-btn sw-btn-danger px-2 py-1 text-xs" @click="deleteTag(t.name)">
            Delete
          </button>
        </div>
        <div v-if="!tags.length" class="p-4 text-sm text-slate-400">No tags yet.</div>
      </div>
    </div>
  </div>
</template>

