<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { Lineicons } from '@lineiconshq/vue-lineicons'
import { Home2Outlined } from '@lineiconshq/free-icons'

type Link = {
  id: string
  alias: string
  target_url: string
  domain_id?: string | null
  short_url?: string
  created_at: string
  expires_at?: string | null
  click_count?: number | null
}

type Domain = { id: string; hostname: string; status: 'pending' | 'verified'; is_primary?: boolean }

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const links = ref<Link[]>([])
const domains = ref<Domain[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const tagFilter = ref<string>('')
const domainFilter = ref<string>('')
const statusFilter = ref<string>('all') // all | active | expired
const q = ref<string>('')
const createdFrom = ref<string>('') // YYYY-MM-DD
const createdTo = ref<string>('') // YYYY-MM-DD
const limit = ref(25)
const offset = ref(0)
const selected = ref<Record<string, boolean>>({})

function copyLink(l: Link) {
  const text = l.short_url ?? `${window.location.origin}/r/${l.alias}`
  void navigator.clipboard.writeText(text)
}

function isExpired(l: Link) {
  if (!l.expires_at) return false
  return new Date(l.expires_at).getTime() <= Date.now()
}

const filteredLinks = computed(() => {
  const qq = q.value.trim().toLowerCase()
  return links.value.filter((l) => {
    if (domainFilter.value && l.domain_id !== domainFilter.value) return false
    if (statusFilter.value === 'active' && isExpired(l)) return false
    if (statusFilter.value === 'expired' && !isExpired(l)) return false
    if (createdFrom.value) {
      const from = new Date(`${createdFrom.value}T00:00:00`).getTime()
      const c = new Date(l.created_at).getTime()
      if (Number.isFinite(from) && c < from) return false
    }
    if (createdTo.value) {
      const to = new Date(`${createdTo.value}T23:59:59.999`).getTime()
      const c = new Date(l.created_at).getTime()
      if (Number.isFinite(to) && c > to) return false
    }
    if (!qq) return true
    return l.alias.toLowerCase().includes(qq) || l.target_url.toLowerCase().includes(qq) || (l.short_url ?? '').toLowerCase().includes(qq)
  })
})

const kpiLinks = computed(() => links.value.length)
const kpiClicks = computed(() => links.value.reduce((acc, l) => acc + (l.click_count ?? 0), 0))
const kpiExpired = computed(() => links.value.reduce((acc, l) => acc + (isExpired(l) ? 1 : 0), 0))
const kpiActive = computed(() => Math.max(0, kpiLinks.value - kpiExpired.value))

const allSelected = computed(() => {
  const rows = filteredLinks.value
  if (!rows.length) return false
  return rows.every((l) => !!selected.value[l.id])
})

function toggleAll() {
  const next = !allSelected.value
  const map: Record<string, boolean> = { ...selected.value }
  for (const l of filteredLinks.value) map[l.id] = next
  selected.value = map
}

async function fetchDomains() {
  try {
    const res = await fetch('/v1/domains?limit=200&offset=0', { credentials: 'include' })
    if (!res.ok) return
    domains.value = (await res.json()) as Domain[]
  } catch {
    // ignore
  }
}

async function fetchLinks() {
  loading.value = true
  error.value = null
  try {
    const tagQ = tagFilter.value ? `&tag=${encodeURIComponent(tagFilter.value)}` : ''
    const res = await fetch(`/v1/links?limit=${limit.value}&offset=${offset.value}${tagQ}`, { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    links.value = (await res.json()) as Link[]
    selected.value = {}
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

function exportCsv() {
  window.open('/v1/links/export?format=csv', '_blank')
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
    <div class="sw-page-header">
      <div>
        <h1 class="sw-title">Links</h1>
        <p class="sw-subtitle">Browse, filter, and manage your links.</p>
      </div>
      <div class="flex items-center gap-2">
        <button class="sw-btn" @click="exportCsv">Export</button>
        <RouterLink class="sw-btn sw-btn-primary" to="/app/home">New link</RouterLink>
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-4">
      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ kpiLinks }}</div>
              <div class="sw-tile-label">Links</div>
            </div>
            <div class="sw-tile-icon">
              <Lineicons :icon="Home2Outlined" :size="20" class="text-lime-300" :stroke-width="1.5" />
            </div>
          </div>
        </div>
      </div>
      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ kpiClicks }}</div>
              <div class="sw-tile-label">Clicks (shown page)</div>
            </div>
            <div class="sw-tile-icon">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 19V5m0 14h16" />
                <path d="M8 17v-6m4 6V9m4 8v-4" />
              </svg>
            </div>
          </div>
        </div>
      </div>
      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ kpiActive }}</div>
              <div class="sw-tile-label">Active</div>
            </div>
            <div class="sw-tile-icon">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 7L10 17l-5-5" />
              </svg>
            </div>
          </div>
        </div>
      </div>
      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ kpiExpired }}</div>
              <div class="sw-tile-label">Expired</div>
            </div>
            <div class="sw-tile-icon">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 8v4l3 3" />
                <path d="M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
        <div class="text-sm font-semibold text-slate-100">Filters</div>
        <div class="mt-3 grid gap-3 md:grid-cols-6">
          <div>
            <label class="sw-label">Domain</label>
            <select v-model="domainFilter" class="sw-select mt-1">
              <option value="">All domains</option>
              <option v-for="d in domains.filter((x) => x.status === 'verified')" :key="d.id" :value="d.id">{{ d.hostname }}</option>
            </select>
          </div>
          <div>
            <label class="sw-label">Status</label>
            <select v-model="statusFilter" class="sw-select mt-1">
              <option value="all">All</option>
              <option value="active">Active</option>
              <option value="expired">Expired</option>
            </select>
          </div>
          <div>
            <label class="sw-label">Created from</label>
            <input v-model="createdFrom" class="sw-input mt-1" type="date" />
          </div>
          <div>
            <label class="sw-label">Created to</label>
            <input v-model="createdTo" class="sw-input mt-1" type="date" />
          </div>
          <div>
            <label class="sw-label">Tag</label>
            <input v-model="tagFilter" class="sw-input mt-1" placeholder="tag" @change="fetchLinks" />
          </div>
          <div>
            <label class="sw-label">Search</label>
            <input v-model="q" class="sw-input mt-1" placeholder="alias or target url" />
          </div>
        </div>

        <div class="mt-4 flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <button class="sw-btn px-3 py-2" @click="fetchLinks">Apply</button>
            <button
              class="sw-btn px-3 py-2"
              @click="
                domainFilter = '';
                statusFilter = 'all';
                createdFrom = '';
                createdTo = '';
                tagFilter = '';
                q = '';
                fetchLinks();
              "
            >
              Reset
            </button>
          </div>
          <div class="text-xs text-slate-400">
            Showing {{ offset + 1 }}–{{ offset + links.length }} (page size {{ limit }})
          </div>
        </div>
        <div v-if="error" class="sw-error mt-3">{{ error }}</div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-widget-header">
        <div class="sw-widget-title">Links</div>
        <div class="sw-widget-actions">
          <button class="sw-btn px-2 py-1 text-xs" :disabled="offset===0" @click="offset=Math.max(0, offset-limit); fetchLinks()">Prev</button>
          <button class="sw-btn px-2 py-1 text-xs" @click="offset=offset+limit; fetchLinks()">Next</button>
          <button class="sw-kebab" title="Actions">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
              <circle cx="12" cy="5" r="1.6" />
              <circle cx="12" cy="12" r="1.6" />
              <circle cx="12" cy="19" r="1.6" />
            </svg>
          </button>
        </div>
      </div>
      <div v-if="loading" class="p-4 text-sm text-slate-400">Loading…</div>
      <div v-else class="sw-table-wrap">
        <table class="sw-table">
          <thead class="sw-thead">
            <tr>
              <th class="sw-th pl-5">
                <input class="sw-check" type="checkbox" :checked="allSelected" @change="toggleAll" />
              </th>
              <th class="sw-th">Link</th>
              <th class="sw-th">Target</th>
              <th class="sw-th">Status</th>
              <th class="sw-th text-right">Clicks</th>
              <th class="sw-th pr-5 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in filteredLinks" :key="l.id" class="sw-row">
              <td class="sw-td pl-5">
                <input class="sw-check" type="checkbox" :checked="!!selected[l.id]" @change="selected[l.id] = !selected[l.id]" />
              </td>
              <td class="sw-td">
                <div class="truncate font-medium text-slate-100">
                  <a class="hover:underline" :href="l.short_url ?? ('/r/' + l.alias)" target="_blank" rel="noreferrer">
                    {{ l.short_url ?? ('/r/' + l.alias) }}
                  </a>
                </div>
                <div class="mt-1 text-xs text-slate-500">{{ l.alias }}</div>
              </td>
              <td class="sw-td pr-4">
                <div class="truncate text-slate-200">{{ l.target_url }}</div>
              </td>
              <td class="sw-td">
                <span v-if="isExpired(l)" class="sw-chip sw-chip-warning">Expired</span>
                <span v-else class="sw-chip sw-chip-success">Active</span>
              </td>
              <td class="sw-td text-right font-semibold sw-accent">{{ l.click_count ?? 0 }}</td>
              <td class="sw-td pr-5">
                <div class="flex justify-end gap-2">
                  <button class="sw-icon-btn" title="Copy" @click="copyLink(l)">
                    <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M9 9h10v10H9z" />
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                    </svg>
                  </button>
                  <button class="sw-icon-btn" title="Analytics" @click="router.push(`/app/links/${l.id}`)">
                    <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M4 19V5m0 14h16" />
                      <path d="M8 17v-6m4 6V9m4 8v-4" />
                    </svg>
                  </button>
                  <button class="sw-icon-btn" title="Delete" @click="deleteLink(l.id)">
                    <svg class="h-4 w-4 text-red-200" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M3 6h18" />
                      <path d="M8 6V4h8v2" />
                      <path d="M19 6l-1 14H6L5 6" />
                      <path d="M10 11v6M14 11v6" />
                    </svg>
                  </button>
                  <button class="sw-icon-btn" title="More">
                    <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                      <circle cx="12" cy="5" r="1.6" />
                      <circle cx="12" cy="12" r="1.6" />
                      <circle cx="12" cy="19" r="1.6" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!filteredLinks.length">
              <td class="py-6 pl-5 text-sm text-slate-400" colspan="6">No links match your filters.</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="flex items-center justify-between gap-2 px-5 py-4 text-xs text-slate-400">
        <div>Showing {{ offset + 1 }} to {{ offset + links.length }} entries</div>
        <div class="flex items-center gap-2">
          <button class="sw-btn px-2 py-1" :disabled="offset===0" @click="offset=Math.max(0, offset-limit); fetchLinks()">‹</button>
          <button class="sw-btn px-2 py-1" @click="offset=offset+limit; fetchLinks()">›</button>
        </div>
      </div>
    </div>
  </div>
</template>

