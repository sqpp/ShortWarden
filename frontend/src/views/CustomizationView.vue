<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Lineicons } from '@lineiconshq/vue-lineicons'
import { Share1Outlined } from '@lineiconshq/free-icons'
import { useAuthStore } from '../stores/auth'

declare const __REPO_URL__: string

type CustomButton = { label: string; url: string }
type RedirectCustomization = {
  delay_seconds: number
  mode: 'auto' | 'click'
  show_screenshot: boolean
  custom_buttons: CustomButton[]
}

const auth = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)
const success = ref('')
const model = ref<RedirectCustomization>({
  delay_seconds: 0,
  mode: 'auto',
  show_screenshot: false,
  custom_buttons: [],
})

const envRepo = (import.meta.env.VITE_REPO_URL as string | undefined) ?? ''
const repoUrl = computed(() => envRepo || __REPO_URL__ || '')
const repoLink = computed(() => repoUrl.value.replace(/\.git$/i, ''))
const instanceOrigin = computed(() => (typeof window !== 'undefined' ? window.location.origin : ''))

function addButton() {
  model.value.custom_buttons.push({ label: '', url: '' })
}

function removeButton(i: number) {
  model.value.custom_buttons.splice(i, 1)
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const res = await fetch('/v1/me/redirect-customization', { credentials: 'include' })
    if (!res.ok) throw new Error(await res.text())
    model.value = (await res.json()) as RedirectCustomization
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load customization.'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) {
    error.value = 'Missing CSRF token. Refresh and try again.'
    return
  }
  saving.value = true
  error.value = null
  success.value = ''
  try {
    const res = await fetch('/v1/me/redirect-customization', {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': auth.csrf },
      body: JSON.stringify(model.value),
    })
    if (!res.ok) throw new Error(await res.text())
    model.value = (await res.json()) as RedirectCustomization
    success.value = 'Saved.'
    setTimeout(() => {
      if (success.value === 'Saved.') success.value = ''
    }, 2000)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to save.'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="space-y-6">
    <div class="sw-page-header">
      <div>
        <h1 class="sw-title">Customization</h1>
        <p class="sw-subtitle">Tune the redirect page visitors see before they open the destination link.</p>
      </div>
    </div>

    <div class="sw-tile">
      <div class="sw-tile-body">
        <div class="sw-tile-top">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <div
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-lime-400/20 ring-1 ring-inset ring-lime-400/25"
            >
              <Lineicons :icon="Share1Outlined" :size="20" class="text-lime-300" :stroke-width="1.5" />
            </div>
            <div class="min-w-0">
              <div class="text-sm font-semibold text-slate-100">Powered by ShortWarden</div>
              <p class="mt-1 text-sm leading-relaxed text-slate-400">
                Every short link you create is generated with
                <span class="font-medium text-slate-300">ShortWarden</span>
                on this instance
                <code class="mx-0.5 rounded border border-white/10 bg-[#1c1f2a] px-1.5 py-0.5 text-xs text-slate-200">{{
                  instanceOrigin || '—'
                }}</code
                >. The redirect page can show a countdown, optional page preview, extra buttons, and a ShortWarden
                footer before visitors continue to the target URL.
              </p>
              <p v-if="repoLink" class="mt-3 text-sm">
                <a
                  :href="repoLink"
                  class="font-medium text-lime-300 hover:text-lime-200"
                  target="_blank"
                  rel="noreferrer"
                  >ShortWarden on GitHub</a
                >
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-header">Redirect page</div>
      <div class="sw-card-body space-y-5">
        <p class="text-sm text-slate-400">
          These settings apply to your account’s redirect interstitial (when delay or “click to continue” is used).
          Screenshot previews use the self-hosted
          <code class="rounded border border-white/10 bg-[#1c1f2a] px-1 text-xs text-slate-200">screenshotd</code>
          service and the public path
          <code class="rounded border border-white/10 bg-[#1c1f2a] px-1 text-xs text-slate-200">/preview/screenshot</code>
          on this app.
        </p>
        <div v-if="loading" class="text-sm text-slate-400">Loading…</div>
        <template v-else>
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <label class="sw-label">Redirect mode</label>
              <select v-model="model.mode" class="sw-select mt-1">
                <option value="auto">Automatic</option>
                <option value="click">Click to continue</option>
              </select>
            </div>
            <div>
              <label class="sw-label">Delay (seconds)</label>
              <input v-model.number="model.delay_seconds" type="number" min="0" max="30" class="sw-input mt-1" />
            </div>
            <div class="flex items-end pb-1">
              <label class="inline-flex cursor-pointer items-start gap-2.5 text-sm text-slate-200">
                <input
                  v-model="model.show_screenshot"
                  type="checkbox"
                  class="mt-0.5 h-4 w-4 shrink-0 rounded border-white/20 bg-[#1c1f2a] text-lime-400 focus:ring-lime-400/30"
                />
                <span>Show page screenshot preview (self-hosted Chromium)</span>
              </label>
            </div>
          </div>

          <div class="space-y-3 rounded-xl border border-white/5 bg-white/[0.02] p-4">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="text-sm font-semibold text-slate-100">Custom buttons</div>
              <button type="button" class="sw-btn px-3 py-1.5 text-xs" @click="addButton">Add button</button>
            </div>
            <div v-if="!model.custom_buttons.length" class="text-sm text-slate-500">No custom buttons yet.</div>
            <div v-for="(b, i) in model.custom_buttons" :key="i" class="grid gap-2 md:grid-cols-[1fr_2fr_auto]">
              <input v-model="b.label" class="sw-input" placeholder="Label" />
              <input v-model="b.url" class="sw-input" placeholder="https://…" />
              <button type="button" class="sw-btn px-3" @click="removeButton(i)">Remove</button>
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-white/5 pt-4">
            <div class="min-w-0 flex-1">
              <div v-if="error" class="sw-error">{{ error }}</div>
              <div v-else-if="success" class="text-sm text-lime-300">{{ success }}</div>
            </div>
            <button type="button" class="sw-btn sw-btn-primary px-4 py-2" :disabled="saving" @click="save">
              {{ saving ? 'Saving…' : 'Save customization' }}
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
