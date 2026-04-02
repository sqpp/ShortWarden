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

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/app/home' },
    { path: '/login', component: LoginView, meta: { public: true } },
    { path: '/register', component: RegisterView, meta: { public: true } },
    {
      path: '/app',
      component: AppLayout,
      children: [
        { path: 'home', component: HomeView },
        { path: 'links', component: LinksView },
        { path: 'links/:id', component: LinkDetailView },
        { path: 'domains', component: DomainsView },
        { path: 'api-keys', component: ApiKeysView },
        { path: 'import-export', component: ImportExportView },
        { path: 'tags', component: TagsView },
        { path: 'settings', component: SettingsView },
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

