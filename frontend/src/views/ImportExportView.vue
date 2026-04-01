<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const error = ref<string | null>(null)
const loading = ref(false)
const dryRun = ref(true)

async function exportJson() {
  window.open('/v1/links/export?format=json', '_blank')
}

async function exportCsv() {
  window.open('/v1/links/export?format=csv', '_blank')
}

async function importFile(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.[0]) return
  if (!auth.csrf) await auth.bootstrap()
  if (!auth.csrf) return

  loading.value = true
  error.value = null
  try {
    const file = input.files[0]
    const isCsv = file.name.toLowerCase().endsWith('.csv')
    const text = await file.text()

    const res = await fetch(`/v1/links/import?dry_run=${dryRun.value ? 'true' : 'false'}`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'content-type': isCsv ? 'text/csv' : 'application/json',
        'X-CSRF-Token': auth.csrf,
      },
      body: text,
    })
    if (!res.ok) throw new Error(await res.text())
    const j = (await res.json()) as { created: number; skipped: number; errors: string[] }
    if (j.errors?.length) {
      error.value = j.errors.join('\n')
    } else {
      error.value = `Import ok. created=${j.created}, skipped=${j.skipped}`
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Import failed'
  } finally {
    loading.value = false
    input.value = ''
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="sw-title">Import / export</h1>
      <p class="sw-subtitle">Export links as JSON/CSV and import them back.</p>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
      <div class="flex flex-wrap gap-2">
        <button class="sw-btn" @click="exportJson">Export JSON</button>
        <button class="sw-btn" @click="exportCsv">Export CSV</button>
      </div>
      </div>
    </div>

    <div class="sw-card">
      <div class="sw-card-body">
        <div class="flex items-center gap-3">
          <label class="sw-label">Dry-run</label>
          <input v-model="dryRun" type="checkbox" />
          <span class="text-sm text-slate-400">Validate only (no writes)</span>
        </div>
      <div class="mt-3">
        <input class="text-sm text-slate-200" type="file" accept=".json,.csv,application/json,text/csv" @change="importFile" />
      </div>
      <div v-if="loading" class="mt-3 text-sm text-slate-400">Importing…</div>
      <pre v-if="error" class="mt-3 whitespace-pre-wrap rounded-md border border-slate-800 bg-slate-950 p-3 text-sm text-slate-200">{{ error }}</pre>
      </div>
    </div>
  </div>
</template>

