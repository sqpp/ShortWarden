<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

type Link = {
  id: string
  alias: string
  target_url: string
  short_url?: string
  created_at: string
}

type Analytics = { from: string; to: string; days: { day: string; clicks: number }[] }
type Click = { id: number; clicked_at: string; referrer?: string; user_agent?: string; ip?: string }

const route = useRoute()
const id = computed(() => String(route.params.id))

const link = ref<Link | null>(null)
const analytics = ref<Analytics | null>(null)
const clicks = ref<Click[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const clickLimit = ref(25)
const clickOffset = ref(0)

const rangeTo = computed(() => new Date())
const rangeFrom = computed(() => {
  const d = new Date()
  d.setDate(d.getDate() - 14)
  return d
})

async function fetchAll() {
  loading.value = true
  error.value = null
  try {
    // link
    {
      const res = await fetch(`/v1/links/${id.value}`, { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      link.value = (await res.json()) as Link
    }
    // analytics
    {
      const from = encodeURIComponent(rangeFrom.value.toISOString())
      const to = encodeURIComponent(rangeTo.value.toISOString())
      const res = await fetch(`/v1/links/${id.value}/analytics?from=${from}&to=${to}`, { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      analytics.value = (await res.json()) as Analytics
    }
    await fetchClicks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
}

async function fetchClicks() {
  const res = await fetch(
    `/v1/links/${id.value}/clicks?limit=${clickLimit.value}&offset=${clickOffset.value}`,
    { credentials: 'include' },
  )
  if (!res.ok) throw new Error(await res.text())
  clicks.value = (await res.json()) as Click[]
}

function nextPage() {
  clickOffset.value += clickLimit.value
}
function prevPage() {
  clickOffset.value = Math.max(0, clickOffset.value - clickLimit.value)
}

watch([clickLimit, clickOffset], async () => {
  try {
    await fetchClicks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  }
})

onMounted(fetchAll)
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="sw-title">Link analytics</h1>
        <p v-if="link" class="sw-subtitle truncate">
          <a
            class="font-medium text-slate-100 hover:underline"
            :href="link.short_url ?? ('/r/' + link.alias)"
            target="_blank"
            rel="noreferrer"
          >
            {{ link.short_url ?? ('/r/' + link.alias) }}
          </a>
          → {{ link.target_url }}
        </p>
      </div>
      <button class="sw-btn" @click="fetchAll">Refresh</button>
    </div>

    <div v-if="error" class="sw-error">{{ error }}</div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="sw-card">
        <div class="sw-card-body">
        <div class="text-sm font-medium text-slate-100">Clicks (last 14 days)</div>
        <div v-if="!analytics" class="mt-2 text-sm text-slate-400">Loading…</div>
        <div v-else class="mt-3">
          <div class="overflow-auto">
            <table class="sw-table">
              <thead class="sw-thead">
                <tr>
                  <th class="py-2">Day</th>
                  <th class="py-2">Clicks</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="d in analytics.days" :key="d.day" class="sw-row">
                  <td class="py-2">{{ new Date(d.day).toLocaleDateString() }}</td>
                  <td class="py-2 font-medium">{{ d.clicks }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        </div>
      </div>

      <div class="sw-card">
        <div class="sw-card-body">
        <div class="flex items-center justify-between">
          <div class="text-sm font-medium text-slate-100">Recent clicks</div>
          <div class="flex items-center gap-2 text-sm">
            <button class="sw-btn px-2 py-1 text-xs" :disabled="clickOffset===0" @click="prevPage">Prev</button>
            <button class="sw-btn px-2 py-1 text-xs" @click="nextPage">Next</button>
          </div>
        </div>

        <div class="mt-3 overflow-auto">
          <table class="sw-table">
            <thead class="sw-thead">
              <tr>
                <th class="py-2">Time</th>
                <th class="py-2">IP</th>
                <th class="py-2">Referrer</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in clicks" :key="c.id" class="sw-row">
                <td class="py-2">{{ new Date(c.clicked_at).toLocaleString() }}</td>
                <td class="py-2 text-slate-300">{{ c.ip ?? '-' }}</td>
                <td class="py-2 truncate max-w-[220px] text-slate-300">{{ c.referrer ?? '-' }}</td>
              </tr>
              <tr v-if="!clicks.length">
                <td class="py-3 text-sm text-slate-400" colspan="3">No clicks yet.</td>
              </tr>
            </tbody>
          </table>
        </div>
        </div>
      </div>
    </div>
  </div>
</template>

