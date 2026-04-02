<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { Lineicons } from '@lineiconshq/vue-lineicons'
import {
  BookmarkCircleOutlined,
  ChevronLeftCircleOutlined,
  DashboardSquare1Outlined,
  ExitUpOutlined,
  Gear1Outlined,
  Globe1Outlined,
  Key1Outlined,
  Paperclip1Outlined,

  User4Outlined,
  Share1Outlined,
} from '@lineiconshq/free-icons'

declare const __APP_VERSION__: string
declare const __REPO_URL__: string

const auth = useAuthStore()
const route = useRoute()
const envVersion = (import.meta.env.VITE_APP_VERSION as string | undefined) ?? ''
const envRepoUrl = (import.meta.env.VITE_REPO_URL as string | undefined) ?? ''
const appVersionText = computed(() => envVersion || __APP_VERSION__ || '')
const repoUrlText = computed(() => envRepoUrl || __REPO_URL__ || '')

type Domain = { id: string; hostname: string; status: 'pending' | 'verified'; is_primary?: boolean }

const SIDEBAR_COLLAPSED_KEY = 'sw.sidebar.collapsed'

const collapsed = ref(false)
try {
  collapsed.value = localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1'
} catch {
  // ignore
}

watch(
  collapsed,
  (v) => {
    try {
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, v ? '1' : '0')
    } catch {
      // ignore
    }
  },
  { flush: 'post' },
)

type NavItem = { to: string; label: string; icon: unknown; badge?: string }
type NavSection = { id: string; title: string; items: NavItem[] }

const navSections = computed<NavSection[]>(() => [
  {
    id: 'apps',
    title: 'Apps & pages',
    items: [
      { to: '/app/home', label: 'App', icon: DashboardSquare1Outlined },
      { to: '/app/links', label: 'Links', icon: Paperclip1Outlined },
      { to: '/app/domains', label: 'Domains', icon: Globe1Outlined },
      { to: '/app/tags', label: 'Tags', icon: BookmarkCircleOutlined },
      { to: '/app/api-keys', label: 'API Keys', icon: Key1Outlined },
    ],
  },
  {
    id: 'tools',
    title: 'Tools',
    items: [
      { to: '/app/import-export', label: 'Export/import', icon: ExitUpOutlined },
      { to: '/app/settings', label: 'Settings', icon: Gear1Outlined },
    ],
  },
])

function isActive(path: string) {
  return route.path === path || (path !== '/app/home' && route.path.startsWith(path))
}

const latestVersion = ref<string>('')
const updateAvailable = ref(false)

function parseVersion(v: string) {
  const clean = v.trim().replace(/^v/i, '')
  const parts = clean.split('.').map((x) => parseInt(x, 10))
  return [parts[0] || 0, parts[1] || 0, parts[2] || 0] as const
}

function isVersionNewer(current: string, latest: string) {
  const a = parseVersion(current)
  const b = parseVersion(latest)
  for (let i = 0; i < 3; i++) {
    if (b[i] > a[i]) return true
    if (b[i] < a[i]) return false
  }
  return false
}

function parseRepo(url: string) {
  const m = url.match(/github\.com\/([^/]+)\/([^/\s]+)/i)
  if (!m) return null
  return { owner: m[1], repo: m[2].replace(/\.git$/i, '') }
}

async function checkForUpdates() {
  if (!repoUrlText.value) return
  const parsed = parseRepo(repoUrlText.value)
  if (!parsed) return
  try {
    let latest = ''
    const releaseRes = await fetch(`https://api.github.com/repos/${parsed.owner}/${parsed.repo}/releases/latest`)
    if (releaseRes.ok) {
      const j = (await releaseRes.json()) as { tag_name?: string; name?: string }
      latest = (j.tag_name || j.name || '').trim()
    } else if (releaseRes.status === 404) {
      const tagsRes = await fetch(`https://api.github.com/repos/${parsed.owner}/${parsed.repo}/tags?per_page=1`)
      if (!tagsRes.ok) return
      const tags = (await tagsRes.json()) as Array<{ name?: string }> 
      latest = (tags[0]?.name || '').trim()
    } else {
      return
    }
    if (!latest) return
    latestVersion.value = latest
    updateAvailable.value = isVersionNewer(appVersionText.value || '0.0.0', latest)
  } catch {
    // ignore
  }
}

const newLinkOpen = ref(false)
const newLinkDomains = ref<Domain[]>([])
const newLinkTargetUrl = ref('')
const newLinkAlias = ref('')
const newLinkDomainId = ref<string>('')
const newLinkExpiryPreset = ref<string>('')
const newLinkExpiresCustom = ref<string>('')
const newLinkLoading = ref(false)
const newLinkError = ref<string | null>(null)

async function fetchNewLinkDomains() {
  try {
    const res = await fetch('/v1/domains?limit=200&offset=0', { credentials: 'include' })
    if (!res.ok) return
    const ds = (await res.json()) as Domain[]
    const verified = ds.filter((d) => d.status === 'verified')
    newLinkDomains.value = verified
    if (!newLinkDomainId.value && verified.length) {
      const primary = verified.find((d) => d.is_primary)
      newLinkDomainId.value = primary?.id ?? verified[0].id
    }
  } catch {
    // ignore
  }
}

function computeNewLinkExpiresISO() {
  if (!newLinkExpiryPreset.value) return undefined
  if (newLinkExpiryPreset.value === 'custom') return newLinkExpiresCustom.value.trim() || undefined
  const now = new Date()
  const add = (ms: number) => new Date(now.getTime() + ms).toISOString()
  switch (newLinkExpiryPreset.value) {
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

async function createNewLink() {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return

  newLinkLoading.value = true
  newLinkError.value = null
  try {
    const expires_at = computeNewLinkExpiresISO()
    const res = await fetch('/v1/links', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({
        target_url: newLinkTargetUrl.value,
        alias: newLinkAlias.value ? newLinkAlias.value : undefined,
        domain_id: newLinkDomainId.value ? newLinkDomainId.value : undefined,
        expires_at,
      }),
    })
    if (!res.ok) throw new Error(await res.text())
    newLinkTargetUrl.value = ''
    newLinkAlias.value = ''
    newLinkExpiryPreset.value = ''
    newLinkExpiresCustom.value = ''
    newLinkOpen.value = false
  } catch (e) {
    newLinkError.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    newLinkLoading.value = false
  }
}

function openNewLink() {
  newLinkOpen.value = true
  newLinkError.value = null
  void fetchNewLinkDomains()
}

onMounted(() => {
  void fetchNewLinkDomains()
  void checkForUpdates()
})
</script>

<template>
  <div class="sw-shell flex">
    <aside class="sw-sidebar flex flex-col" :class="collapsed ? 'sw-sidebar-collapsed' : ''">
      <div class="sw-sidebar-header" :class="collapsed ? 'justify-center' : ''">
        <div class="flex items-center gap-2">
          <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-lime-400/20 ring-1 ring-inset ring-lime-400/25">
            <Lineicons :icon="Share1Outlined" :size="16" class="text-lime-300" :stroke-width="1.5" />
          </div>
          <div v-if="!collapsed" class="sw-brand">ShortWarden</div>
        </div>
      </div>

      <nav class="sw-nav">
        <div v-for="s in navSections" :key="s.id" class="sw-nav-section">
          <div class="sw-nav-section-title">{{ s.title }}</div>
          <div class="mt-1 space-y-1 pl-1">
            <RouterLink
              v-for="i in s.items"
              :key="i.to + ':' + i.label"
              class="sw-nav-item"
              :class="isActive(i.to) ? 'sw-nav-active' : ''"
              :to="i.to"
              :title="collapsed ? undefined : i.label"
            >
              <span class="sw-nav-left">
                <Lineicons
                  :icon="i.icon"
                  :size="18"
                  class="sw-nav-icon"
                  :class="isActive(i.to) ? 'text-white' : 'text-lime-300'"
                  :stroke-width="1.5"
                />
                <span class="sw-nav-label">{{ i.label }}</span>
              </span>
              <span v-if="i.badge" class="sw-nav-badge">{{ i.badge }}</span>
            </RouterLink>
          </div>
        </div>
      </nav>

      <div class="mt-auto border-t border-slate-700/60 p-3 flex flex-col items-center gap-3">
        <button class="sw-icon-btn mx-auto" title="Toggle sidebar" @click="collapsed = !collapsed">
          <Lineicons
            :icon="ChevronLeftCircleOutlined"
            :size="18"
            :stroke-width="1.5"
            class="text-lime-300 transition-transform"
            :style="{ transform: collapsed ? 'rotate(180deg)' : 'rotate(0deg)', transformOrigin: 'center' }"
          />
        </button>
      </div>
    </aside>

    <div class="sw-content">
      <header class="sw-topbar">
        <div class="sw-topbar-left">
          <button class="sw-btn sw-btn-primary px-3 py-2" @click="openNewLink">New link</button>
        </div>
        <div class="sw-topbar-right">
          <a
            v-if="updateAvailable && repoUrlText"
            class="sw-btn px-3 py-2 text-lime-300 border-lime-300/30"
            :href="`${repoUrlText.replace(/\\.git$/i, '')}/releases`"
            target="_blank"
            rel="noreferrer"
          >
            Update available {{ latestVersion }}
          </a>
          <div class="sw-user-chip" title="Account">
            <div class="sw-avatar flex items-center justify-center">
              <Lineicons :icon="User4Outlined" :size="18" class="text-lime-300" :stroke-width="1.5" />
            </div>
            <span class="hidden lg:inline">{{ auth.user?.email ?? 'Account' }}</span>
          </div>
        </div>
      </header>

      <main class="sw-page">
        <RouterView />
      </main>
    </div>
  </div>

  <div
    v-if="newLinkOpen"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-6"
    @keydown.esc="newLinkOpen = false"
  >
    <div class="w-full max-w-2xl">
      <div class="sw-card">
        <div class="sw-card-header flex items-center justify-between">
          <div>Create a new link</div>
          <button class="sw-icon-btn" title="Close" @click="newLinkOpen = false">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6 6 18" />
              <path d="M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="sw-card-body space-y-4">
          <div class="grid gap-3 md:grid-cols-3">
            <div class="md:col-span-2">
              <label class="sw-label">Target URL</label>
              <input v-model="newLinkTargetUrl" class="sw-input mt-1" placeholder="https://…" />
            </div>
            <div>
              <label class="sw-label">Domain</label>
              <select v-model="newLinkDomainId" class="sw-select mt-1" :disabled="!newLinkDomains.length">
                <option v-for="d in newLinkDomains" :key="d.id" :value="d.id">{{ d.hostname }}</option>
              </select>
            </div>
          </div>

          <div class="rounded-xl border border-white/5 bg-white/[0.03] p-3">
            <div class="text-xs font-semibold uppercase tracking-wide text-slate-400">Optional</div>
            <div class="mt-3 grid gap-3 md:grid-cols-3">
              <div>
                <label class="sw-label">Alias</label>
                <input v-model="newLinkAlias" class="sw-input mt-1" placeholder="my-alias" />
              </div>
              <div>
                <label class="sw-label">Expiry</label>
                <select v-model="newLinkExpiryPreset" class="sw-select mt-1">
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
              <div v-if="newLinkExpiryPreset === 'custom'">
                <label class="sw-label">Custom expires_at</label>
                <input v-model="newLinkExpiresCustom" class="sw-input mt-1" placeholder="2026-12-31T00:00:00Z" />
              </div>
            </div>
          </div>

          <div class="flex items-center justify-between gap-3">
            <div v-if="newLinkError" class="text-sm text-red-200">{{ newLinkError }}</div>
            <div class="flex items-center gap-2">
              <button class="sw-btn" :disabled="newLinkLoading" @click="newLinkOpen = false">Cancel</button>
              <button class="sw-btn sw-btn-primary" :disabled="newLinkLoading" @click="createNewLink">
                {{ newLinkLoading ? 'Creating…' : 'Create link' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

