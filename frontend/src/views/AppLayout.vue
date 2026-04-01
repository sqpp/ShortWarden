<script setup lang="ts">
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
</script>

<template>
  <div class="sw-shell flex">
    <aside class="sw-sidebar">
      <div class="sw-sidebar-header">
        <div class="sw-brand">ShortWarden</div>
      </div>

      <nav class="sw-nav space-y-1">
        <RouterLink class="sw-nav-link" to="/app/links">Links</RouterLink>
        <RouterLink class="sw-nav-link" to="/app/domains">Domains</RouterLink>
        <RouterLink class="sw-nav-link" to="/app/api-keys">API keys</RouterLink>
        <RouterLink class="sw-nav-link" to="/app/import-export">Import/export</RouterLink>
        <RouterLink class="sw-nav-link" to="/app/tags">Tags</RouterLink>
        <RouterLink class="sw-nav-link" to="/app/settings">Settings</RouterLink>
      </nav>

      <div class="mt-auto border-t border-slate-800 p-3">
        <button class="sw-btn w-full" @click="auth.logout()">Logout</button>
      </div>
    </aside>

    <div class="sw-content">
      <header class="sw-topbar">
        <div class="min-w-0">
          <div class="truncate text-sm font-medium text-slate-200">Dashboard</div>
          <div class="truncate text-xs text-slate-500">Manage links, domains, tags, and API keys</div>
        </div>
      </header>

      <main class="sw-page">
        <RouterView />
      </main>
    </div>
  </div>
</template>

