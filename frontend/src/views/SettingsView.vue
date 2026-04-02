<script setup lang="ts">
import { useAuthStore } from '../stores/auth'
import { computed, onMounted, ref } from 'vue'

declare const __APP_VERSION__: string
declare const __BUILD_TIME__: string
declare const __GIT_SHA__: string
declare const __REPO_URL__: string

const auth = useAuthStore()
const envVersion = (import.meta.env.VITE_APP_VERSION as string | undefined) ?? ''
const envBuildTime = (import.meta.env.VITE_BUILD_TIME as string | undefined) ?? ''
const envGitSha = (import.meta.env.VITE_GIT_SHA as string | undefined) ?? ''
const envRepoUrl = (import.meta.env.VITE_REPO_URL as string | undefined) ?? ''
const envDockerImage = (import.meta.env.VITE_DOCKER_IMAGE as string | undefined) ?? ''
const appVersionText = computed(() => envVersion || __APP_VERSION__ || '')
const buildTimeText = computed(() => envBuildTime || __BUILD_TIME__ || '')
const gitShaText = computed(() => envGitSha || __GIT_SHA__ || '')
const repoUrlText = computed(() => envRepoUrl || __REPO_URL__ || '')
const dockerImageText = computed(() => envDockerImage || 'sqpp/shortwarden')

const currentPassword = ref('')
const newPassword = ref('')
const msg = ref<string | null>(null)
const loading = ref(false)

const redirectDelaySeconds = ref(0)
const keepExpiredLinks = ref(false)
const timezone = ref('UTC')
const timezoneOptions = ref<string[]>([])
const timezoneQuery = ref('')
const timezoneOpen = ref(false)
let blurCloseTimer: number | null = null
const checkingUpdates = ref(false)
const updateError = ref<string | null>(null)
const latestVersion = ref<string>('')
const updateAvailable = ref(false)
const currentRuntimeVersion = ref<string>('')
const runningUpdate = ref(false)
const updateOutput = ref('')
const updateRunError = ref<string | null>(null)

function getTimezoneOptions() {
  const sup = (Intl as unknown as { supportedValuesOf?: (key: string) => string[] }).supportedValuesOf
  if (typeof sup === 'function') {
    const v = sup('timeZone')
    if (Array.isArray(v) && v.length) return v
  }
  return [
    'UTC',
    'Europe/London',
    'Europe/Berlin',
    'Europe/Paris',
    'Europe/Warsaw',
    'Europe/Bucharest',
    'Europe/Istanbul',
    'America/New_York',
    'America/Chicago',
    'America/Denver',
    'America/Los_Angeles',
    'America/Sao_Paulo',
    'Asia/Dubai',
    'Asia/Kolkata',
    'Asia/Bangkok',
    'Asia/Singapore',
    'Asia/Tokyo',
    'Asia/Seoul',
    'Australia/Sydney',
  ]
}

const timezoneHint = computed(() => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
})

const timezoneDisplay = computed(() => timezone.value || timezoneHint.value)

const filteredTimezones = computed(() => {
  const q = timezoneQuery.value.trim().toLowerCase()
  if (!q) return timezoneOptions.value.slice(0, 100)
  return timezoneOptions.value.filter((z) => z.toLowerCase().includes(q)).slice(0, 100)
})

function pickTimezone(z: string) {
  timezone.value = z
  timezoneQuery.value = z
  timezoneOpen.value = false
}

function scheduleCloseTimezone() {
  if (blurCloseTimer != null) {
    clearTimeout(blurCloseTimer)
  }
  blurCloseTimer = setTimeout(() => {
    timezoneOpen.value = false
    blurCloseTimer = null
  }, 150) as unknown as number
}

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

function parseDockerImage(image: string) {
  const parts = image.trim().split('/')
  if (parts.length !== 2 || !parts[0] || !parts[1]) return null
  return { namespace: parts[0], repo: parts[1] }
}

async function fetchLatestFromDockerHub(image: string) {
  const parsed = parseDockerImage(image)
  if (!parsed) throw new Error('Invalid Docker image name')
  const res = await fetch(`https://hub.docker.com/v2/repositories/${parsed.namespace}/${parsed.repo}/tags?page_size=100`)
  if (!res.ok) throw new Error(`Docker Hub error (${res.status})`)
  const j = (await res.json()) as { results?: Array<{ name?: string }> }
  const tags = (j.results ?? [])
    .map((t) => (t.name || '').trim())
    .filter((t) => /^v?\d+\.\d+\.\d+$/.test(t))
  if (!tags.length) return ''
  tags.sort((a, b) => (isVersionNewer(a, b) ? -1 : isVersionNewer(b, a) ? 1 : 0))
  return tags[0] || ''
}

async function checkForUpdates() {
  updateError.value = null
  latestVersion.value = ''
  updateAvailable.value = false
  checkingUpdates.value = true
  try {
    const res = await fetch('/v1/system/latest-version', { credentials: 'include' })
    let latest = ''
    if (res.ok) {
      const j = (await res.json()) as { latest_version?: string }
      latest = (j.latest_version || '').trim()
    } else if (res.status === 404) {
      latest = await fetchLatestFromDockerHub(dockerImageText.value)
    } else {
      throw new Error(await res.text())
    }
    if (!latest) throw new Error('No Docker Hub semver tag found')
    latestVersion.value = latest
    updateAvailable.value = isVersionNewer(currentRuntimeVersion.value || appVersionText.value || '0.0.0', latest)
  } catch (e) {
    updateError.value = e instanceof Error ? e.message : 'Failed checking updates'
  } finally {
    checkingUpdates.value = false
  }
}

async function fetchRuntimeVersion() {
  try {
    const res = await fetch(`/v1/system/version?t=${Date.now()}`, { credentials: 'include', cache: 'no-store' })
    if (!res.ok) return
    const j = (await res.json()) as { app_version?: string }
    currentRuntimeVersion.value = (j.app_version || '').trim()
  } catch {
    // ignore
  }
}

type UpdateStatus = {
  running: boolean
  last_started?: string
  last_finished?: string
  exit_code?: number
  output?: string
  error?: string
}

async function fetchUpdateStatus() {
  const res = await fetch('/v1/system/update', { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  const s = (await res.json()) as UpdateStatus
  runningUpdate.value = !!s.running
  updateOutput.value = s.output ?? ''
  updateRunError.value = s.error ? `Update failed: ${s.error}` : null
  if (!s.running && typeof s.exit_code === 'number' && s.exit_code !== 0 && !updateRunError.value) {
    updateRunError.value = `Update failed with exit code ${s.exit_code}`
  }
}

async function runUpdateNow() {
  updateRunError.value = null
  updateOutput.value = ''
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) {
    updateRunError.value = 'Missing CSRF token'
    return
  }
  const res = await fetch('/v1/system/update', {
    method: 'POST',
    credentials: 'include',
    headers: { 'X-CSRF-Token': auth.csrf },
  })
  if (!res.ok) {
    updateRunError.value = await res.text()
    return
  }
  runningUpdate.value = true
  const timer = setInterval(async () => {
    try {
      await fetchUpdateStatus()
      if (!runningUpdate.value) {
        clearInterval(timer)
        // API process restarted; runtime endpoint can lag briefly, retry a few times.
        for (let i = 0; i < 10; i++) {
          const before = currentRuntimeVersion.value
          await fetchRuntimeVersion()
          if (currentRuntimeVersion.value && currentRuntimeVersion.value !== before) break
          await new Promise((resolve) => setTimeout(resolve, 1000))
        }
        await checkForUpdates()
      }
    } catch {
      // ignore transient polling errors
    }
  }, 2500)
}

async function loadSettings() {
  msg.value = null
  try {
    const res = await fetch('/v1/me/settings', { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    const j = (await res.json()) as {
      redirect_delay_seconds: number
      keep_expired_links: boolean
      timezone: string
    }
    redirectDelaySeconds.value = j.redirect_delay_seconds
    keepExpiredLinks.value = j.keep_expired_links
    timezone.value = j.timezone
    timezoneQuery.value = j.timezone
  } catch (e) {
    msg.value = e instanceof Error ? e.message : 'Failed'
  }
}

async function saveSettings() {
  msg.value = null
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  try {
    const res = await fetch('/v1/me/settings', {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({
        redirect_delay_seconds: redirectDelaySeconds.value,
        keep_expired_links: keepExpiredLinks.value,
        timezone: timezone.value,
      }),
    })
    if (!res.ok) throw new Error(await res.text())
    msg.value = 'Settings saved.'
  } catch (e) {
    msg.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function changePassword() {
  msg.value = null
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return
  loading.value = true
  try {
    const res = await fetch('/v1/me/password', {
      method: 'POST',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify({ current_password: currentPassword.value, new_password: newPassword.value }),
    })
    if (!res.ok) throw new Error(await res.text())
    currentPassword.value = ''
    newPassword.value = ''
    msg.value = 'Password updated.'
  } catch (e) {
    msg.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  timezoneOptions.value = getTimezoneOptions()
  void loadSettings()
  void fetchRuntimeVersion()
  void checkForUpdates()
  void fetchUpdateStatus()
})
</script>

<template>
  <div class="space-y-4 max-w-4xl">
    <div class="max-w-3xl">
      <h1 class="sw-title">Settings</h1>
      <p class="sw-subtitle">Account preferences and security.</p>
      <div class="mt-3 text-sm text-slate-400">
        Signed in as <span class="font-medium text-slate-200">{{ auth.user?.email }}</span>
      </div>
    </div>

    <div class="space-y-4">
      <div class="sw-card max-w-3xl">
        <div class="sw-card-body">
          <div class="text-sm font-medium text-slate-100">About</div>
          <div class="mt-3 grid gap-3 md:grid-cols-2">
            <div>
              <div class="text-xs uppercase tracking-wide text-slate-500">Version</div>
              <div class="mt-1 text-sm text-slate-200">{{ currentRuntimeVersion || appVersionText || '—' }}</div>
            </div>
            <div>
              <div class="text-xs uppercase tracking-wide text-slate-500">Build</div>
              <div class="mt-1 text-sm text-slate-200">{{ buildTimeText || '—' }}</div>
            </div>
            <div>
              <div class="text-xs uppercase tracking-wide text-slate-500">Commit</div>
              <div class="mt-1 text-sm text-slate-200">{{ gitShaText || '—' }}</div>
            </div>
            <div>
              <div class="text-xs uppercase tracking-wide text-slate-500">GitHub</div>
              <div class="mt-1">
                <a v-if="repoUrlText" class="text-sm text-lime-300 hover:underline" :href="repoUrlText" target="_blank" rel="noreferrer">
                  {{ repoUrlText }}
                </a>
                <div v-else class="text-sm text-slate-400">Set `VITE_REPO_URL` to show this.</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="sw-card max-w-3xl">
        <div class="sw-card-body">
          <div class="text-sm font-medium text-slate-100">Updates</div>
          <div class="mt-3 grid gap-3 md:grid-cols-2">
            <div>
              <div class="text-xs uppercase tracking-wide text-slate-500">Current</div>
              <div class="mt-1 text-sm text-slate-200">{{ currentRuntimeVersion || appVersionText || '—' }}</div>
            </div>
            <div>
              <div class="text-xs uppercase tracking-wide text-slate-500">Latest</div>
              <div class="mt-1 text-sm" :class="updateAvailable ? 'text-lime-300' : 'text-slate-200'">
                {{ latestVersion || '—' }}
                <span v-if="updateAvailable" class="ml-2 text-xs font-semibold uppercase tracking-wide">Update available</span>
              </div>
            </div>
          </div>
          <div class="mt-3 flex items-center gap-2">
            <button class="sw-btn" :disabled="checkingUpdates" @click="checkForUpdates">
              {{ checkingUpdates ? 'Checking…' : 'Check for updates' }}
            </button>
            <button class="sw-btn sw-btn-primary" :disabled="runningUpdate" @click="runUpdateNow">
              {{ runningUpdate ? 'Updating…' : 'Update now' }}
            </button>
            <a
              v-if="updateAvailable && repoUrlText"
              class="sw-btn sw-btn-primary"
              :href="`${repoUrlText.replace(/\\.git$/i, '')}/releases`"
              target="_blank"
              rel="noreferrer"
            >
              Open releases
            </a>
          </div>
          <div v-if="updateError" class="mt-2 text-sm text-red-200">{{ updateError }}</div>
          <div v-if="updateRunError" class="mt-2 text-sm text-red-200">{{ updateRunError }}</div>
          <pre
            v-if="updateOutput"
            class="mt-3 max-h-44 overflow-auto rounded border border-white/10 bg-[#1c1f2a] p-2 text-xs text-slate-300"
          >{{ updateOutput }}</pre>
        </div>
      </div>

      <div class="sw-card max-w-3xl">
        <div class="sw-card-body">
          <div class="text-sm font-medium text-slate-100">Preferences</div>
          <div class="mt-3 space-y-3">
            <div>
              <label class="sw-label">Redirect delay (seconds)</label>
              <input v-model.number="redirectDelaySeconds" class="sw-input mt-1" type="number" min="0" max="30" />
            </div>

            <div class="flex items-center gap-2">
              <input id="keepExpired" v-model="keepExpiredLinks" type="checkbox" />
              <label for="keepExpired" class="text-sm">Keep expired links</label>
            </div>

            <div class="relative">
              <label class="sw-label">Timezone</label>
              <input
                v-model="timezoneQuery"
                class="sw-input mt-1"
                :placeholder="timezoneDisplay"
                @focus="
                  timezoneOpen = true;
                  timezoneQuery = timezoneDisplay;
                "
                @input="timezoneOpen = true"
                @blur="scheduleCloseTimezone"
              />

              <div
                v-if="timezoneOpen"
                class="absolute z-20 mt-2 w-full overflow-hidden rounded-xl border border-white/10 bg-[#141826] shadow-[0_18px_60px_rgba(0,0,0,0.55)]"
              >
                <div class="max-h-64 overflow-auto py-1">
                  <button
                    v-for="z in filteredTimezones"
                    :key="z"
                    type="button"
                    class="block w-full truncate px-3 py-2 text-left text-sm text-slate-200 hover:bg-white/5"
                    @mousedown.prevent="pickTimezone(z)"
                  >
                    {{ z }}
                  </button>
                  <div v-if="!filteredTimezones.length" class="px-3 py-2 text-sm text-slate-400">No matches</div>
                </div>
              </div>

              <div class="mt-1 text-xs text-slate-500">Example: {{ timezoneHint }}</div>
            </div>
          </div>

          <div class="mt-4 flex items-center justify-between gap-3">
            <div v-if="msg" class="text-sm text-slate-300">{{ msg }}</div>
            <button class="sw-btn" :disabled="loading" @click="saveSettings">Save preferences</button>
          </div>
        </div>
      </div>

      <div class="sw-card max-w-3xl">
        <div class="sw-card-body">
          <div class="text-sm font-medium text-slate-100">Change password</div>
          <div class="mt-3 grid gap-3 md:grid-cols-2">
            <div>
              <label class="sw-label">Current password</label>
              <input v-model="currentPassword" class="sw-input mt-1" type="password" />
            </div>
            <div>
              <label class="sw-label">New password</label>
              <input v-model="newPassword" class="sw-input mt-1" type="password" minlength="8" />
            </div>
          </div>
          <div class="mt-4 flex items-center justify-between gap-3">
            <div v-if="msg" class="text-sm text-slate-300">{{ msg }}</div>
            <button class="sw-btn sw-btn-primary" :disabled="loading" @click="changePassword">
              {{ loading ? 'Saving…' : 'Update password' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

