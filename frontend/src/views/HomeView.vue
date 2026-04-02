<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { useAuthStore } from '../stores/auth'

type Stats = { links_total: number; clicks_total: number; clicks_24h: number; clicks_7d: number }
type Domain = { id: string; hostname: string; status: 'pending' | 'verified'; is_primary?: boolean }

const auth = useAuthStore()

type TopLink = { link: { id: string; alias: string; short_url?: string | null; target_url: string }; clicks: number }
type ListedLink = {
  id: string
  alias: string
  target_url: string
  short_url?: string | null
  created_at: string
}

const stats = ref<Stats | null>(null)
const statsError = ref<string | null>(null)
const recentLinks = ref<ListedLink[]>([])
const topLinks = ref<TopLink[]>([])
const homeError = ref<string | null>(null)

const domains = ref<Domain[]>([])
const targetUrl = ref('')
const alias = ref('')
const domainId = ref<string>('')

const expiryPreset = ref<string>('')
const expiresCustom = ref<string>('')

const loading = ref(false)
const error = ref<string | null>(null)
const createSuccess = ref(false)
const createdShortUrl = ref<string | null>(null)
const createdUrlInput = ref<HTMLInputElement | null>(null)
const copyHint = ref('')

function pickStr(v: unknown): string {
  return typeof v === 'string' ? v.trim() : ''
}

function shortUrlFromCreateBody(raw: string, selectedDomainId: string, domainList: Domain[]): string {
  let j: Record<string, unknown> = {}
  try {
    if (raw) j = JSON.parse(raw) as Record<string, unknown>
  } catch {
    return ''
  }
  const direct = pickStr(j.short_url) || pickStr(j.ShortUrl) || pickStr(j.shortUrl)
  if (direct) return direct
  const al = pickStr(j.alias) || pickStr(j.Alias)
  if (!al) return ''
  const did = pickStr(j.domain_id) || pickStr(j.DomainId) || selectedDomainId
  const host = domainList.find((d) => d.id === did)?.hostname
  if (host) return `${window.location.protocol}//${host}/r/${al}`
  return `${window.location.origin}/r/${al}`
}

async function copyTextToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // fall through
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}

async function copyCreatedUrl() {
  const t = createdShortUrl.value
  if (!t) return
  copyHint.value = ''
  const ok = await copyTextToClipboard(t)
  copyHint.value = ok ? 'Copied.' : 'Select the text and press Ctrl+C.'
  if (ok) {
    window.setTimeout(() => {
      if (copyHint.value === 'Copied.') copyHint.value = ''
    }, 2000)
  }
}

function dismissCreated() {
  createSuccess.value = false
  createdShortUrl.value = null
  copyHint.value = ''
}

watch([createSuccess, createdShortUrl], async ([ok, url]) => {
  if (!ok || !url) return
  await nextTick()
  const el = createdUrlInput.value
  el?.focus()
  el?.select()
  const copied = await copyTextToClipboard(url)
  if (copied) {
    copyHint.value = 'Copied to clipboard.'
    window.setTimeout(() => {
      if (copyHint.value === 'Copied to clipboard.') copyHint.value = ''
    }, 2500)
  }
})

function computeExpiresISO() {
  if (!expiryPreset.value) return undefined
  if (expiryPreset.value === 'custom') return expiresCustom.value.trim() || undefined
  const now = new Date()
  const add = (ms: number) => new Date(now.getTime() + ms).toISOString()
  switch (expiryPreset.value) {
    case '5m':
      return add(5 * 60 * 1000)
    case '15m':
      return add(15 * 60 * 1000)
    case '30m':
      return add(30 * 60 * 1000)
    case '1h':
      return add(60 * 60 * 1000)
    case '6h':
      return add(6 * 60 * 60 * 1000)
    case '24h':
      return add(24 * 60 * 60 * 1000)
    case '3d':
      return add(3 * 24 * 60 * 60 * 1000)
    case '7d':
      return add(7 * 24 * 60 * 60 * 1000)
    case '1mo': {
      const d = new Date(now)
      d.setMonth(d.getMonth() + 1)
      return d.toISOString()
    }
    default:
      return undefined
  }
}

async function fetchStats() {
  statsError.value = null
  try {
    const res = await fetch('/v1/stats', { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    stats.value = (await res.json()) as Stats
  } catch (e) {
    statsError.value = e instanceof Error ? e.message : 'Failed'
  }
}

async function fetchDomains() {
  try {
    const res = await fetch('/v1/domains?limit=200&offset=0', { credentials: 'include' })
    if (!res.ok) return
    const ds = (await res.json()) as Domain[]
    const verified = ds.filter((d) => d.status === 'verified')
    domains.value = verified
    if (!domainId.value && verified.length) {
      const primary = verified.find((d) => d.is_primary)
      domainId.value = primary?.id ?? verified[0].id
    }
  } catch {
    // ignore
  }
}

async function createLink() {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) {
    error.value = 'Session not ready. Refresh and try again.'
    return
  }

  loading.value = true
  error.value = null
  createSuccess.value = false
  createdShortUrl.value = null
  copyHint.value = ''
  try {
    const expires_at = computeExpiresISO()
    const domainSnap = domainId.value
    const res = await fetch('/v1/links', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({
        target_url: targetUrl.value,
        alias: alias.value ? alias.value : undefined,
        domain_id: domainId.value ? domainId.value : undefined,
        expires_at,
      }),
    })
    const raw = await res.text()
    if (!res.ok) throw new Error(raw || `Failed (${res.status})`)
    const shortUrl = shortUrlFromCreateBody(raw, domainSnap, domains.value)
    targetUrl.value = ''
    alias.value = ''
    expiryPreset.value = ''
    expiresCustom.value = ''
    createSuccess.value = true
    createdShortUrl.value = shortUrl || null
    if (!createdShortUrl.value) {
      copyHint.value = 'Link created. See Recent links on the right for the short URL.'
    }
    await Promise.all([fetchStats(), fetchRecentLinks(), fetchHomePanels()])
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function fetchRecentLinks() {
  try {
    const res = await fetch('/v1/links?limit=5&offset=0', { credentials: 'include' })
    if (!res.ok) return
    recentLinks.value = (await res.json()) as ListedLink[]
  } catch {
    // ignore
  }
}

async function fetchHomePanels() {
  homeError.value = null
  try {
    const tl = await fetch('/v1/home/top-links?limit=10&days=7', { credentials: 'include' })
    if (!tl.ok) throw new Error(await tl.text())
    topLinks.value = (await tl.json()) as TopLink[]
  } catch (e) {
    homeError.value = e instanceof Error ? e.message : 'Failed'
  }
}

function refreshHomeWidgets() {
  void fetchHomePanels()
  void fetchRecentLinks()
}

onMounted(() => {
  void fetchStats()
  void fetchHomePanels()
  void fetchRecentLinks()
  void fetchDomains()
})
</script>

<template>
  <div class="space-y-6">
    <div class="sw-page-header">
      <div>
        <h1 class="sw-title">Analytics</h1>
        <p class="sw-subtitle">Create links, monitor activity, and review performance.</p>
      </div>
      <div class="flex items-center gap-2">
        <button class="sw-btn" :disabled="loading" @click="refreshHomeWidgets">Refresh</button>
        <button class="sw-btn" :disabled="loading" @click="fetchStats">Refresh stats</button>
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-4">
      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ stats?.links_total ?? '—' }}</div>
              <div class="sw-tile-label">Total links</div>
              <div class="sw-tile-delta">
                <span class="sw-tile-delta-muted">All time</span>
              </div>
            </div>
            <div class="sw-tile-icon">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10 13a5 5 0 0 1 0-7l1.5-1.5a5 5 0 0 1 7 7L18 13" />
                <path d="M14 11a5 5 0 0 1 0 7L12.5 19.5a5 5 0 0 1-7-7L6 11" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ stats?.clicks_24h ?? '—' }}</div>
              <div class="sw-tile-label">Clicks (24h)</div>
              <div class="sw-tile-delta">
                <span class="sw-tile-delta-muted">Last 24 hours</span>
              </div>
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
              <div class="sw-tile-value">{{ stats?.clicks_7d ?? '—' }}</div>
              <div class="sw-tile-label">Clicks (7d)</div>
              <div class="sw-tile-delta">
                <span class="sw-tile-delta-muted">Last 7 days</span>
              </div>
            </div>
            <div class="sw-tile-icon">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 12h18" />
                <path d="M7 12v7m5-7v7m5-7v7" />
                <path d="M7 12V7m5 5V5m5 7V9" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div class="sw-tile">
        <div class="sw-tile-body">
          <div class="sw-tile-top">
            <div>
              <div class="sw-tile-value">{{ stats?.clicks_total ?? '—' }}</div>
              <div class="sw-tile-label">Clicks (all)</div>
              <div class="sw-tile-delta">
                <span class="sw-tile-delta-muted">All time</span>
              </div>
            </div>
            <div class="sw-tile-icon">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2l3 7h7l-5.5 4 2 7-6.5-4.5L5.5 20l2-7L2 9h7l3-7Z" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="statsError" class="sw-error">{{ statsError }}</div>
    <div v-if="homeError" class="sw-error">{{ homeError }}</div>

    <div class="sw-card">
      <div class="sw-card-header">Create a new link</div>
      <div class="sw-card-body space-y-4">
        <div class="grid gap-3 md:grid-cols-4">
          <div class="md:col-span-2">
            <label class="sw-label">Target URL</label>
            <input v-model="targetUrl" class="sw-input mt-1" placeholder="https://…" />
          </div>
          <div>
            <label class="sw-label">Alias (optional)</label>
            <input v-model="alias" class="sw-input mt-1" placeholder="my-alias" />
          </div>
          <div>
            <label class="sw-label">Domain</label>
            <select v-model="domainId" class="sw-select mt-1">
              <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.hostname }}</option>
            </select>
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-4">
          <div>
            <label class="sw-label">Expiry</label>
            <select v-model="expiryPreset" class="sw-select mt-1">
              <option value="">No expiry</option>
              <option value="5m">5 minutes</option>
              <option value="15m">15 minutes</option>
              <option value="30m">30 minutes</option>
              <option value="1h">1 hour</option>
              <option value="6h">6 hours</option>
              <option value="24h">24 hours</option>
              <option value="3d">3 days</option>
              <option value="7d">7 days</option>
              <option value="1mo">1 month</option>
              <option value="custom">Custom (RFC3339)</option>
            </select>
          </div>
          <div class="md:col-span-3">
            <label class="sw-label">Custom expires_at (RFC3339)</label>
            <input
              v-model="expiresCustom"
              class="sw-input mt-1"
              :disabled="expiryPreset !== 'custom'"
              placeholder="2026-12-31T00:00:00Z"
            />
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3">
          <div v-if="error" class="text-sm text-red-200">{{ error }}</div>
          <button type="button" class="sw-btn sw-btn-primary" :disabled="loading" @click="createLink">
            {{ loading ? 'Creating…' : 'Create link' }}
          </button>
        </div>

        <div
          v-if="createSuccess"
          class="space-y-3 rounded-xl border border-lime-400/25 bg-lime-400/10 p-4 ring-1 ring-inset ring-lime-400/15"
        >
          <div class="text-sm font-semibold text-lime-200">Link created</div>
          <template v-if="createdShortUrl">
            <p class="text-sm text-slate-300">Your short URL is below. Use Copy if it was not copied automatically.</p>
            <div class="flex flex-col gap-2 sm:flex-row sm:items-stretch">
              <input
                ref="createdUrlInput"
                readonly
                class="sw-input min-w-0 flex-1 font-mono text-sm"
                :value="createdShortUrl"
                @click="createdUrlInput?.select()"
              />
              <button type="button" class="sw-btn sw-btn-primary shrink-0 px-4" @click="copyCreatedUrl">Copy</button>
              <button type="button" class="sw-btn shrink-0" @click="dismissCreated">Dismiss</button>
            </div>
          </template>
          <template v-else>
            <p class="text-sm text-slate-300">The link was created. Open <strong class="text-slate-200">Recent links</strong> for the short URL.</p>
            <button type="button" class="sw-btn" @click="dismissCreated">Dismiss</button>
          </template>
          <p v-if="copyHint" class="text-sm" :class="copyHint.startsWith('Copied') ? 'text-lime-300' : 'text-amber-200'">
            {{ copyHint }}
          </p>
        </div>
      </div>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="sw-card">
        <div class="sw-card-header">Top links (7d)</div>
        <div class="sw-card-body">
          <div class="overflow-auto">
            <table class="sw-table">
              <thead class="sw-thead">
                <tr>
                  <th class="py-2">Link</th>
                  <th class="py-2 text-right">Clicks</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="t in topLinks" :key="t.link.id" class="sw-row">
                  <td class="py-2">
                    <div class="truncate font-medium">
                      <a class="hover:underline" :href="t.link.short_url ?? ('/r/' + t.link.alias)" target="_blank" rel="noreferrer">
                        {{ t.link.short_url ?? ('/r/' + t.link.alias) }}
                      </a>
                    </div>
                    <div class="truncate text-xs text-slate-400">{{ t.link.target_url }}</div>
                  </td>
                  <td class="py-2 text-right font-semibold sw-accent">{{ t.clicks }}</td>
                </tr>
                <tr v-if="!topLinks.length">
                  <td class="py-3 text-sm text-slate-400" colspan="2">No data yet.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="sw-card">
        <div class="sw-card-header">Recent links</div>
        <div class="sw-card-body">
          <div class="overflow-auto">
            <table class="sw-table">
              <thead class="sw-thead">
                <tr>
                  <th class="py-2">Short link</th>
                  <th class="py-2">Target</th>
                  <th class="py-2">Created</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="l in recentLinks" :key="l.id" class="sw-row">
                  <td class="py-2">
                    <a
                      class="font-medium hover:underline sw-accent"
                      :href="l.short_url ?? '/r/' + l.alias"
                      target="_blank"
                      rel="noreferrer"
                    >
                      {{ l.short_url ?? '/r/' + l.alias }}
                    </a>
                  </td>
                  <td class="py-2 max-w-[14rem] truncate text-slate-400" :title="l.target_url">{{ l.target_url }}</td>
                  <td class="py-2 whitespace-nowrap text-sm text-slate-400">
                    {{ new Date(l.created_at).toLocaleString() }}
                  </td>
                </tr>
                <tr v-if="!recentLinks.length">
                  <td class="py-3 text-sm text-slate-400" colspan="3">No links yet.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

