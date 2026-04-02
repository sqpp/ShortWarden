import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const LoginView = () => import('./views/LoginView.vue')
const RegisterView = () => import('./views/RegisterView.vue')
const AppLayout = () => import('./views/AppLayout.vue')
const HomeView = () => import('./views/HomeView.vue')
const LinksView = () => import('./views/LinksView.vue')
const LinkDetailView = () => import('./views/LinkDetailView.vue')
const DomainsView = () => import('./views/DomainsView.vue')
const ApiKeysView = () => import('./views/ApiKeysView.vue')
const ImportExportView = () => import('./views/ImportExportView.vue')
const TagsView = () => import('./views/TagsView.vue')
const SettingsView = () => import('./views/SettingsView.vue')
const CustomizationView = () => import('./views/CustomizationView.vue')

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/app/home' },
    { path: '/login', component: LoginView, meta: { public: true, title: 'Login', description: 'Sign in to your ShortWarden account.' } },
    { path: '/register', component: RegisterView, meta: { public: true, title: 'Register', description: 'Create your ShortWarden account.' } },
    {
      path: '/app',
      component: AppLayout,
      children: [
        { path: 'home', component: HomeView, meta: { title: 'Dashboard', description: 'Track performance and recent activity.' } },
        { path: 'links', component: LinksView, meta: { title: 'Links', description: 'Create, search, and manage links.' } },
        { path: 'links/:id', component: LinkDetailView, meta: { title: 'Link analytics', description: 'Inspect clicks and analytics for a link.' } },
        { path: 'domains', component: DomainsView, meta: { title: 'Domains', description: 'Manage verified domains and defaults.' } },
        { path: 'api-keys', component: ApiKeysView, meta: { title: 'API keys', description: 'Create and revoke integration keys.' } },
        { path: 'import-export', component: ImportExportView, meta: { title: 'Import / export', description: 'Move links in and out of ShortWarden.' } },
        { path: 'tags', component: TagsView, meta: { title: 'Tags', description: 'Browse and manage tags across links.' } },
        { path: 'settings', component: SettingsView, meta: { title: 'Settings', description: 'Account preferences and security.' } },
        { path: 'customization', component: CustomizationView, meta: { title: 'Customization', description: 'Configure redirect page behavior and actions.' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.user && !auth.loading) {
    await auth.bootstrap()
  }
  if (to.meta.public) return true
  if (!auth.user) return { path: '/login', query: { next: to.fullPath } }
  return true
})

router.afterEach((to) => {
  const base = 'ShortWarden'
  const title = (to.meta.title as string | undefined)?.trim()
  document.title = title ? `${title} - ${base}` : base
  const description = (to.meta.description as string | undefined)?.trim()
  if (!description) return
  let meta = document.querySelector('meta[name="description"]')
  if (!meta) {
    meta = document.createElement('meta')
    meta.setAttribute('name', 'description')
    document.head.appendChild(meta)
  }
  meta.setAttribute('content', description)
})

